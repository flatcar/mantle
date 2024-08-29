// Copyright 2017 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package equinixmetal

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/pkg/capnslog"
	ignition "github.com/flatcar/ignition/config/v2_0/types"
	"github.com/packethost/packngo"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"

	"github.com/flatcar/mantle/auth"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/equinixmetal/storage"
	"github.com/flatcar/mantle/platform/api/equinixmetal/storage/gcs"
	"github.com/flatcar/mantle/platform/api/equinixmetal/storage/sshstorage"
	"github.com/flatcar/mantle/platform/api/gcloud"
	"github.com/flatcar/mantle/platform/conf"
	ms "github.com/flatcar/mantle/storage"
	"github.com/flatcar/mantle/util"
)

const (
	// Provisioning a VM is supposed to take < 8 minutes, but in practice can take longer.
	defaultLaunchTimeout  = 10 * time.Minute
	launchPollInterval    = 30 * time.Second
	defaultInstallTimeout = 45 * time.Minute
	installPollInterval   = 5 * time.Second
	apiRetries            = 3
	apiRetryInterval      = 5 * time.Second
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/api/equinixmetal")

	defaultInstallerImageBaseURL = map[string]string{
		"amd64-usr": "https://stable.release.flatcar-linux.net/amd64-usr/current",
		"arm64-usr": "https://alpha.release.flatcar-linux.net/arm64-usr/current",
	}
	defaultImageURL = map[string]string{
		"amd64-usr": "https://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_packet_image.bin.bz2",
		"arm64-usr": "https://alpha.release.flatcar-linux.net/arm64-usr/current/flatcar_production_packet_image.bin.bz2",
	}
	defaultPlan = map[string]string{
		"amd64-usr": "c3.small.x86",
		"arm64-usr": "c3.large.arm",
	}
	linuxConsole = map[string]string{
		"amd64-usr": "console=ttyS1,115200n8",
		"arm64-usr": "",
	}
)

type Options struct {
	*platform.Options

	// Config file. Defaults to $HOME/.config/equinixmetal.json.
	ConfigPath string
	// Profile name
	Profile string
	// API key (overrides config profile)
	ApiKey string
	// Project UUID (overrides config profile)
	Project string

	// Slug of the device type (e.g. "baremetal_0")
	Plan string
	// e.g. http://alpha.release.flatcar-linux.net/amd64-usr/current
	InstallerImageBaseURL string
	// e.g. http://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_pxe.vmlinuz
	InstallerImageKernelURL string
	// e.g. http://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_pxe_image.cpio.gz
	InstallerImageCpioURL string
	// e.g. https://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_packet_image.bin.bz2
	ImageURL string

	// Options for Google Storage
	GSOptions *gcloud.Options
	// Google Storage base URL for temporary uploads
	// e.g. gs://users.developer.core-os.net/bovik/mantle
	StorageURL string
	// StorageSSHPort is the port used to SSH into the "storage" server a SSH URL is provided.
	StorageSSHPort string
	// Metro is where you want your server to live.
	Metro string

	// RemoteOptions for remote storage

	// RemoteUser is the user for the SSH connection.
	RemoteUser string

	// RemoteSSHPrivateKeyPath is the private SSH key path used for the SSH authentication.
	RemoteSSHPrivateKeyPath string

	// RemoteDocumentRoot is the path served by the webserver.
	RemoteDocumentRoot string

	// LaunchTimeout specifies the timeout used for waiting for instance to launch.
	LaunchTimeout time.Duration
	// InstallTimeout specifies the timeout used for waiting for installation to finish.
	InstallTimeout time.Duration
}

type API struct {
	c       *packngo.Client
	storage storage.Storage
	opts    *Options
}

type Console interface {
	io.WriteCloser
	SSHClient(ip, user string) (*ssh.Client, error)
}

func New(opts *Options) (*API, error) {
	if opts.ApiKey == "" || opts.Project == "" {
		profiles, err := auth.ReadEquinixMetalConfig(opts.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("couldn't read EquinixMetal config: %v", err)
		}

		if opts.Profile == "" {
			opts.Profile = "default"
		}
		profile, ok := profiles[opts.Profile]
		if !ok {
			return nil, fmt.Errorf("no such profile %q", opts.Profile)
		}
		if opts.ApiKey == "" {
			opts.ApiKey = profile.ApiKey
		}
		if opts.Project == "" {
			opts.Project = profile.Project
		}
	}

	_, ok := linuxConsole[opts.Board]
	if !ok {
		return nil, fmt.Errorf("unknown board %q", opts.Board)
	}
	if opts.Plan == "" {
		opts.Plan = defaultPlan[opts.Board]
	}
	if opts.InstallerImageBaseURL == "" {
		opts.InstallerImageBaseURL = defaultInstallerImageBaseURL[opts.Board]
	}
	if opts.InstallerImageKernelURL == "" {
		opts.InstallerImageKernelURL = strings.TrimRight(opts.InstallerImageBaseURL, "/") + "/flatcar_production_pxe.vmlinuz"
	}
	if opts.InstallerImageCpioURL == "" {
		opts.InstallerImageCpioURL = strings.TrimRight(opts.InstallerImageBaseURL, "/") + "/flatcar_production_pxe_image.cpio.gz"
	}
	if opts.ImageURL == "" {
		opts.ImageURL = defaultImageURL[opts.Board]
	}

	url, err := url.Parse(opts.StorageURL)
	if err != nil {
		return nil, fmt.Errorf("parsing storage URL: %w", err)
	}

	var storage storage.Storage

	switch url.Scheme {
	case "gs":
		gapi, err := gcloud.New(opts.GSOptions)
		if err != nil {
			return nil, fmt.Errorf("connecting to Google Storage: %w", err)
		}

		bucket, err := ms.NewBucket(gapi.Client(), opts.StorageURL)
		if err != nil {
			return nil, fmt.Errorf("connecting to Google Storage bucket: %w", err)
		}

		storage = gcs.New(bucket)
	case "ssh+http", "ssh+https", "ssh":
		key, err := ioutil.ReadFile(opts.RemoteSSHPrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("reading private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parsing private key: %w", err)
		}

		cfg := &ssh.ClientConfig{
			User: opts.RemoteUser,
			// this is only used for testing - it's ok to live with that.
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
		}

		client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", url.Hostname(), opts.StorageSSHPort), cfg)
		if err != nil {
			return nil, fmt.Errorf("creating SSH client: %w", err)
		}

		// default webserver protocol is HTTPs
		protocol := "https"

		// we try to extract the procol from the URL scheme of the
		// storage URL (ssh+http, ssh+https, etc.)
		protocols := strings.SplitN(url.Scheme, "+", 2)
		for _, proto := range protocols {
			if proto == "ssh" {
				continue
			}

			protocol = proto
		}

		storage = sshstorage.New(client, url.Host, opts.RemoteDocumentRoot, protocol)
	}

	if opts.LaunchTimeout == 0 {
		opts.LaunchTimeout = defaultLaunchTimeout
	}
	if opts.LaunchTimeout < 0 {
		return nil, fmt.Errorf("launch timeout can't be negative, is %v", opts.LaunchTimeout)
	}

	if opts.InstallTimeout == 0 {
		opts.InstallTimeout = defaultInstallTimeout
	}
	if opts.InstallTimeout < 0 {
		return nil, fmt.Errorf("install timeout can't be negative, is %v", opts.InstallTimeout)
	}

	client := packngo.NewClientWithAuth("github.com/flatcar/mantle", opts.ApiKey, nil)

	return &API{
		c:       client,
		storage: storage,
		opts:    opts,
	}, nil
}

func (a *API) PreflightCheck() error {
	_, _, err := a.c.Projects.Get(a.opts.Project, nil)
	if err != nil {
		return fmt.Errorf("querying project %v: %v", a.opts.Project, err)
	}
	return nil
}

// Close takes care of closing existing connections.
func (a *API) Close() error {
	return a.storage.Close()
}

// console is optional, and is closed on error or when the device is deleted.
func (a *API) CreateOrUpdateDevice(hostname string, conf *conf.Conf, console Console, id string) (*packngo.Device, error) {
	consoleStarted := false
	defer func() {
		if console != nil && !consoleStarted {
			console.Close()
		}
	}()

	userdata, err := a.wrapUserData(conf)
	if err != nil {
		return nil, err
	}

	// The Ignition config can't go in userdata via coreos.config.url=https://metadata.packet.net/userdata because Ignition supplies an Accept header that metadata.packet.net finds 406 Not Acceptable.
	// It can't go in userdata via coreos.oem.id=packet because the EquinixMetal OEM expects unit files in /oem (or /usr/share/oem) which the PXE image doesn't have.
	userdataName, userdataURL, err := a.uploadObject(hostname, "application/vnd.coreos.ignition+json", []byte(userdata))
	if err != nil {
		return nil, err
	}
	defer a.storage.Delete(context.TODO(), userdataName)

	plog.Debugf("user-data available at %s", userdataURL)

	// This can't go in userdata because the installed coreos-cloudinit will try to execute it.
	ipxeScriptName, ipxeScriptURL, err := a.uploadObject(hostname, "application/octet-stream", []byte(a.ipxeScript(userdataURL)))
	if err != nil {
		return nil, err
	}
	defer a.storage.Delete(context.TODO(), ipxeScriptName)

	plog.Debugf("iPXE script available at %s", ipxeScriptURL)

	device, err := a.createDevice(hostname, ipxeScriptURL, id)
	if err != nil {
		return nil, fmt.Errorf("couldn't create device: %v", err)
	}
	destroyDevice := true
	deviceID := device.ID
	defer func() {
		if destroyDevice {
			a.DeleteDevice(deviceID)
		}
	}()

	plog.Debugf("Created device: %q", deviceID)

	if console != nil {
		err := a.startConsole(deviceID, device.Facility.Code, console)
		consoleStarted = true
		if err != nil {
			return nil, err
		}
	}

	device, err = a.waitForActive(deviceID)
	if err != nil {
		return nil, err
	}

	ipAddress := a.GetDeviceAddress(device, 4, true)
	if ipAddress == "" {
		return nil, fmt.Errorf("no public IP address found for %v", deviceID)
	}

	plog.Debugf("Device active: %q", deviceID)

	err = waitForInstall(a.opts.InstallTimeout, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("timed out waiting for flatcar-install: %v", err)
	}

	// TCP discard service has been reached so `flatcar-install` is done.
	// We can deactivate `PXE` boot to avoid bootlooping.
	alwaysPXE := false
	if _, _, err = a.c.Devices.Update(deviceID, &packngo.DeviceUpdateRequest{
		AlwaysPXE: &alwaysPXE,
	}); err != nil {
		return nil, fmt.Errorf("unable to deactivate PXE boot: %v", err)
	}

	plog.Debugf("Finished installation of device: %q", deviceID)

	destroyDevice = false
	return device, nil
}

func (a *API) DeleteDevice(deviceID string) error {
	_, err := a.c.Devices.Delete(deviceID, true)
	if err != nil {
		return fmt.Errorf("deleting device %q: %v", deviceID, err)
	}
	return nil
}

func (a *API) GetDeviceAddress(device *packngo.Device, family int, public bool) string {
	for _, address := range device.Network {
		if address.AddressFamily == family && address.Public == public {
			return address.Address
		}
	}
	return ""
}

func (a *API) AddKey(name, key string) (string, error) {
	sshKey, _, err := a.c.SSHKeys.Create(&packngo.SSHKeyCreateRequest{
		Label: name,
		Key:   key,
	})
	if err != nil {
		return "", fmt.Errorf("couldn't create SSH key: %v", err)
	}
	return sshKey.ID, nil
}

func (a *API) DeleteKey(keyID string) error {
	_, err := a.c.SSHKeys.Delete(keyID)
	if err != nil {
		return fmt.Errorf("couldn't delete SSH key: %v", err)
	}
	return nil
}

func (a *API) ListKeys() ([]packngo.SSHKey, error) {
	keys, _, err := a.c.SSHKeys.List()
	if err != nil {
		return nil, fmt.Errorf("couldn't list SSH keys: %v", err)
	}
	return keys, nil
}

func (a *API) wrapUserData(conf *conf.Conf) (string, error) {
	userDataOption := "-i"
	if !conf.IsIgnition() && conf.String() != "" {
		// By providing a no-op Ignition config, we prevent Ignition
		// from enabling oem-cloudinit.service, which is unordered
		// with respect to the cloud-config installed by the -c
		// option. Otherwise it might override settings in the
		// cloud-config with defaults obtained from the EquinixMetal
		// metadata endpoint.
		userDataOption = "-i /noop.ign -c"
	}
	escapedImageURL := strings.Replace(a.opts.ImageURL, "%", "%%", -1)

	// make systemd units
	discardSocketUnit := `
[Unit]
Description=Discard Socket

[Socket]
ListenStream=0.0.0.0:9
Accept=true

[Install]
WantedBy=multi-user.target
`
	discardServiceUnit := `
[Unit]
Description=Discard Service
Requires=discard.socket

[Service]
ExecStart=/usr/bin/cat
StandardInput=socket
StandardOutput=null
`
	installUnit := `
[Unit]
Description=Install Container Linux

Requires=network-online.target
After=network-online.target

[Service]
Type=oneshot
Restart=on-failure
RemainAfterExit=true
RestartSec=3s
# Prevent flatcar-install from validating cloud-config
Environment=PATH=/root/bin:/usr/sbin:/usr/bin

ExecStart=/opt/installer

StandardOutput=journal+console
StandardError=journal+console

[Install]
RequiredBy=multi-user.target
`

	// make workarounds
	noopIgnitionConfig := base64.StdEncoding.EncodeToString([]byte(`{"ignition": {"version": "2.1.0"}}`))
	coreosCloudInit := base64.StdEncoding.EncodeToString([]byte("#!/bin/sh\nexit 0"))
	installerScript := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(`#!/bin/bash
set -euo pipefail
curl --retry-delay 1 --retry 120 --retry-connrefused --retry-max-time 120 --connect-timeout 20 -fsSLo image.bin.bz2 "%v"
# We don't verify signatures because the iPXE script isn't verified either
# (and, in fact, is transferred over HTTP)
lvchange -an /dev/mapper/* || true
shopt -s nullglob
for disk in /dev/*d? /dev/nvme?n1; do
  wipefs --all --force "${disk}" || true
done
# 259 is a major number of NVMe devices. They need to be excluded, because
# the boot agent can't boot from them on s3.xlarge.x86.
INSTANCE=$(curl --retry-delay 1 --retry 120 --retry-connrefused --retry-max-time 120 --connect-timeout 20 -fsSL 'https://metadata.packet.net/metadata' | jq -r '.plan')
EXCLUDE=""
if [ "${INSTANCE}" = "s3.xlarge.x86" ]; then
  EXCLUDE="-e 259"
fi
flatcar-install -s ${EXCLUDE} -f image.bin.bz2 %v /userdata
systemctl --no-block isolate reboot.target
`, escapedImageURL, userDataOption)))

	// make Ignition config
	b64UserData := base64.StdEncoding.EncodeToString(conf.Bytes())
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(ignition.Config{
		Ignition: ignition.Ignition{
			Version: ignition.IgnitionVersion{Major: 2},
		},
		Storage: ignition.Storage{
			Files: []ignition.File{
				{
					Filesystem: "root",
					Path:       "/userdata",
					Contents: ignition.FileContents{
						Source: ignition.Url{
							Scheme: "data",
							Opaque: ";base64," + b64UserData,
						},
					},
					Mode: 0644,
				},
				{
					Filesystem: "root",
					Path:       "/noop.ign",
					Contents: ignition.FileContents{
						Source: ignition.Url{
							Scheme: "data",
							Opaque: ";base64," + noopIgnitionConfig,
						},
					},
					Mode: 0644,
				},
				{
					Filesystem: "root",
					Path:       "/root/bin/coreos-cloudinit",
					Contents: ignition.FileContents{
						Source: ignition.Url{
							Scheme: "data",
							Opaque: ";base64," + coreosCloudInit,
						},
					},
					Mode: 0755,
				},
				{
					Filesystem: "root",
					Path:       "/opt/installer",
					Contents: ignition.FileContents{
						Source: ignition.Url{
							Scheme: "data",
							Opaque: ";base64," + installerScript,
						},
					},
					Mode: 0755,
				},
			},
		},
		Systemd: ignition.Systemd{
			Units: []ignition.SystemdUnit{
				{
					// don't appear to be running while install is in progress
					Name: "sshd.socket",
					Mask: true,
				},
				{
					// future-proofing
					Name: "sshd.service",
					Mask: true,
				},
				{
					// allow remote detection of install in progress
					Name:     "discard.socket",
					Enable:   true,
					Contents: discardSocketUnit,
				},
				{
					Name:     "discard@.service",
					Contents: discardServiceUnit,
				},
				{
					Name:     "flatcar-install.service",
					Enable:   true,
					Contents: installUnit,
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("encoding Ignition config: %v", err)
	}

	return buf.String(), nil
}

func (a *API) uploadObject(hostname, contentType string, data []byte) (string, string, error) {
	if hostname == "" {
		hostname = "mantle"
	}
	b := make([]byte, 5)
	rand.Read(b)
	name := fmt.Sprintf("%s-%x", hostname, b)

	name, URL, err := a.storage.Upload(name, contentType, data)
	if err != nil {
		return "", "", fmt.Errorf("uploading content: %w", err)
	}

	return name, URL, nil
}

func (a *API) ipxeScript(userdataURL string) string {
	return fmt.Sprintf(`#!ipxe
kernel %s initrd=flatcar_production_pxe_image.cpio.gz flatcar.first_boot=1 flatcar.oem.id=packet ignition.config.url=%s %s
initrd %s
boot`, a.opts.InstallerImageKernelURL, userdataURL, linuxConsole[a.opts.Board], a.opts.InstallerImageCpioURL)
}

// device creation seems a bit flaky, so try a few times
func (a *API) createDevice(hostname, ipxeScriptURL, id string) (*packngo.Device, error) {
	var err error

	// we force a PXE boot in order to fetch the
	// new configuration and prevent to boot from a mis-installed Flatcar.
	alwaysPXE := true

	for tries := apiRetries; tries >= 0; tries-- {
		var (
			device   *packngo.Device
			response *packngo.Response
		)

		if id != "" {
			plog.Infof("Recycling instance: %s", id)
			device, response, err = a.c.Devices.Update(id, &packngo.DeviceUpdateRequest{
				AlwaysPXE:     &alwaysPXE,
				IPXEScriptURL: &ipxeScriptURL,
				Hostname:      &hostname,
			})
			if err != nil {
				err = fmt.Errorf("updating device: %w", err)
				continue
			}

			// we reboot the instance to apply the changes.
			response, err = a.c.Devices.Reboot(id)
			if err != nil {
				err = fmt.Errorf("rebooting device: %w", err)
				continue
			}

			plog.Infof("device rebooted: %s", id)
		} else {
			plog.Infof("Recycling is not possible, creating a new instance")

			device, response, err = a.c.Devices.Create(&packngo.DeviceCreateRequest{
				ProjectID:     a.opts.Project,
				Plan:          a.opts.Plan,
				BillingCycle:  "hourly",
				Hostname:      hostname,
				OS:            "custom_ipxe",
				IPXEScriptURL: ipxeScriptURL,
				Tags:          []string{"mantle"},
				AlwaysPXE:     alwaysPXE,
				Metro:         a.opts.Metro,
			})
		}

		if err == nil || response.StatusCode != 500 {
			return device, err
		}

		plog.Debugf("Retrying to create device after failure: %q %q %q \n", device, response, err)
		if device != nil && device.ID != "" {
			a.DeleteDevice(device.ID)
		}
		if tries > 0 {
			time.Sleep(apiRetryInterval)
		}
	}

	return nil, fmt.Errorf("reached maximum number of retries to create/update a device: %w", err)
}

func (a *API) startConsole(deviceID, facility string, console Console) error {
	ready := make(chan error)

	runner := func() error {
		defer console.Close()

		client, err := console.SSHClient("sos."+facility+".platformequinix.com", deviceID)
		if err != nil {
			return fmt.Errorf("couldn't create SSH client for %s console: %v", deviceID, err)
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			return fmt.Errorf("couldn't create SSH session for %s console: %v", deviceID, err)
		}
		defer session.Close()

		reader, writer := io.Pipe()
		defer writer.Close()

		session.Stdin = reader
		session.Stdout = console
		if err := session.Shell(); err != nil {
			return fmt.Errorf("couldn't start shell for %s console: %v", deviceID, err)
		}

		// cause startConsole to return
		ready <- nil

		err = session.Wait()
		_, ok := err.(*ssh.ExitMissingError)
		if err != nil && !ok {
			plog.Errorf("%s console session failed: %v", deviceID, err)
		}
		return nil
	}
	go func() {
		err := runner()

		if err != nil {
			ready <- err
		}
	}()

	return <-ready
}

func (a *API) waitForActive(deviceID string) (*packngo.Device, error) {
	var device *packngo.Device
	err := util.WaitUntilReady(a.opts.LaunchTimeout, launchPollInterval, func() (bool, error) {
		var err error
		device, _, err = a.c.Devices.Get(deviceID, nil)
		if err != nil {
			return false, fmt.Errorf("querying device: %v", err)
		}
		return device.State == "active", nil
	})
	if err != nil {
		return nil, err
	}
	return device, nil
}

// Connect to the discard port and wait for the connection to close,
// indicating that install is complete.
func waitForInstall(installTimeout time.Duration, address string) (err error) {
	deadline := time.Now().Add(installTimeout)
	dialer := net.Dialer{
		Timeout: installPollInterval,
	}
	for tries := installTimeout / installPollInterval; tries >= 0; tries-- {
		var conn net.Conn
		start := time.Now()
		conn, err = dialer.Dial("tcp", address+":9")
		if err == nil {
			defer conn.Close()
			conn.SetDeadline(deadline)
			_, err = conn.Read([]byte{0})
			if err == io.EOF {
				err = nil
			}
			return
		}
		if tries > 0 {
			// If Dial returned an error before the timeout,
			// e.g. because the device returned ECONNREFUSED,
			// wait out the rest of the interval.
			time.Sleep(installPollInterval - time.Now().Sub(start))
		}
	}
	return
}

func (a *API) GC(gracePeriod time.Duration) error {
	threshold := time.Now().Add(-gracePeriod)

	page := packngo.ListOptions{
		Page:    1,
		PerPage: 1000,
	}

	for {
		devices, _, err := a.c.Devices.List(a.opts.Project, &page)
		if err != nil {
			return fmt.Errorf("listing devices: %v", err)
		}
		for _, device := range devices {
			tagged := false
			for _, tag := range device.Tags {
				if tag == "mantle" {
					tagged = true
					break
				}
			}
			if !tagged {
				continue
			}

			switch device.State {
			case "queued", "provisioning":
				continue
			}

			if device.Locked {
				continue
			}

			created, err := time.Parse(time.RFC3339, device.Created)
			if err != nil {
				return fmt.Errorf("couldn't parse %q: %v", device.Created, err)
			}
			if created.After(threshold) {
				continue
			}

			if err := a.DeleteDevice(device.ID); err != nil {
				return fmt.Errorf("couldn't delete device %v: %v", device.ID, err)
			}
		}
		if len(devices) < page.PerPage {
			return nil
		}
		page.Page += 1
	}
}
