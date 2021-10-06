// Copyright 2015 CoreOS, Inc.
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

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/flatcar-linux/mantle/auth"
	"github.com/flatcar-linux/mantle/kola"
	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/sdk"
)

var (
	outputDir          string
	kolaPlatform       string
	kolaChannel        string
	kolaOffering       string
	defaultTargetBoard = sdk.DefaultBoard()
	kolaArchitectures  = []string{"amd64"}
	kolaPlatforms      = []string{"aws", "azure", "do", "esx", "external", "gce", "openstack", "packet", "qemu", "qemu-unpriv"}
	kolaDistros        = []string{"cl", "fcos", "rhcos"}
	kolaChannels       = []string{"alpha", "beta", "stable", "edge", "lts"}
	kolaOfferings      = []string{"basic", "pro"}
	kolaDefaultImages  = map[string]string{
		"amd64-usr": sdk.BuildRoot() + "/images/amd64-usr/latest/flatcar_production_image.bin",
		"arm64-usr": sdk.BuildRoot() + "/images/arm64-usr/latest/flatcar_production_image.bin",
	}
	kolaIgnitionVersionDefaults = map[string]string{
		"cl":    "v2",
		"fcos":  "v3",
		"rhcos": "v3",
	}

	kolaDefaultBIOS = map[string]string{
		"amd64-usr": "bios-256k.bin",
		"arm64-usr": sdk.BuildRoot() + "/images/arm64-usr/latest/flatcar_production_qemu_uefi_efi_code.fd",
	}
)

func init() {
	sv := root.PersistentFlags().StringVar
	bv := root.PersistentFlags().BoolVar
	ss := root.PersistentFlags().StringSlice

	// general options
	sv(&outputDir, "output-dir", "", "Temporary output directory for test data and logs")
	sv(&kola.TorcxManifestFile, "torcx-manifest", "", "Path to a torcx manifest that should be made available to tests")
	root.PersistentFlags().StringVarP(&kolaPlatform, "platform", "p", "qemu", "VM platform: "+strings.Join(kolaPlatforms, ", "))
	root.PersistentFlags().StringVarP(&kolaChannel, "channel", "", "stable", "Channel: "+strings.Join(kolaChannels, ", "))
	root.PersistentFlags().StringVarP(&kolaOffering, "offering", "", "basic", "Offering: "+strings.Join(kolaOfferings, ", "))
	root.PersistentFlags().StringVarP(&kola.Options.Distribution, "distro", "b", "cl", "Distribution: "+strings.Join(kolaDistros, ", "))
	root.PersistentFlags().IntVarP(&kola.TestParallelism, "parallel", "j", 1, "number of tests to run in parallel")
	sv(&kola.TAPFile, "tapfile", "", "file to write TAP results to")
	sv(&kola.Options.BaseName, "basename", "kola", "Cluster name prefix")
	ss("debug-systemd-unit", []string{}, "full-unit-name.service to enable SYSTEMD_LOG_LEVEL=debug on. Specify multiple times for multiple units.")
	sv(&kola.UpdatePayloadFile, "update-payload", "", "Path to an update payload that should be made available to tests")
	sv(&kola.Options.IgnitionVersion, "ignition-version", "", "Ignition version override: v2, v3")

	// rhcos-specific options
	sv(&kola.Options.OSContainer, "oscontainer", "", "oscontainer image pullspec for pivot (RHCOS only)")

	// aws-specific options
	defaultRegion := os.Getenv("AWS_REGION")
	if defaultRegion == "" {
		defaultRegion = "us-west-2"
	}
	sv(&kola.AWSOptions.CredentialsFile, "aws-credentials-file", "", "AWS credentials file (default \"~/.aws/credentials\")")
	sv(&kola.AWSOptions.Region, "aws-region", defaultRegion, "AWS region")
	sv(&kola.AWSOptions.Profile, "aws-profile", "default", "AWS profile name")
	sv(&kola.AWSOptions.AMI, "aws-ami", "alpha", `AWS AMI ID, or (alpha|beta|stable) to use the latest image`)
	sv(&kola.AWSOptions.InstanceType, "aws-type", "m4.large", "AWS instance type")
	sv(&kola.AWSOptions.SecurityGroup, "aws-sg", "kola", "AWS security group name")
	sv(&kola.AWSOptions.IAMInstanceProfile, "aws-iam-profile", "kola", "AWS IAM instance profile name")

	// azure-specific options
	sv(&kola.AzureOptions.AzureProfile, "azure-profile", "", "Azure profile (default \"~/"+auth.AzureProfilePath+"\")")
	sv(&kola.AzureOptions.AzureAuthLocation, "azure-auth", "", "Azure auth location (default \"~/"+auth.AzureAuthPath+"\")")
	sv(&kola.AzureOptions.BlobURL, "azure-blob-url", "", "Azure source page blob to be copied from a public/SAS URL, recommended way (from \"plume pre-release\" or \"ore azure upload-blob-arm\")")
	sv(&kola.AzureOptions.ImageFile, "azure-image-file", "", "Azure image file (local image to upload in the temporary kola resource group)")
	sv(&kola.AzureOptions.DiskURI, "azure-disk-uri", "", "Azure disk uri (custom images)")
	sv(&kola.AzureOptions.Publisher, "azure-publisher", "CoreOS", "Azure image publisher (default \"CoreOS\"")
	sv(&kola.AzureOptions.Offer, "azure-offer", "CoreOS", "Azure image offer (default \"CoreOS\"")
	sv(&kola.AzureOptions.Sku, "azure-sku", "alpha", "Azure image sku/channel (default \"alpha\"")
	sv(&kola.AzureOptions.Version, "azure-version", "", "Azure image version")
	sv(&kola.AzureOptions.Location, "azure-location", "westus", "Azure location (default \"westus\"")
	sv(&kola.AzureOptions.Size, "azure-size", "Standard_DS2_v2", "Azure machine size (default \"Standard_DS2_v2\")")
	sv(&kola.AzureOptions.HyperVGeneration, "azure-hyper-v-generation", "V1", "Azure Hyper-V Generation (\"V1\" or \"V2\")")

	// do-specific options
	sv(&kola.DOOptions.ConfigPath, "do-config-file", "", "DigitalOcean config file (default \"~/"+auth.DOConfigPath+"\")")
	sv(&kola.DOOptions.Profile, "do-profile", "", "DigitalOcean profile (default \"default\")")
	sv(&kola.DOOptions.AccessToken, "do-token", "", "DigitalOcean access token (overrides config file)")
	sv(&kola.DOOptions.Region, "do-region", "sfo2", "DigitalOcean region slug")
	sv(&kola.DOOptions.Size, "do-size", "s-1vcpu-2gb", "DigitalOcean size slug")
	sv(&kola.DOOptions.Image, "do-image", "alpha", "DigitalOcean image ID, {alpha, beta, stable}, or user image name")

	// esx-specific options
	sv(&kola.ESXOptions.ConfigPath, "esx-config-file", "", "ESX config file (default \"~/"+auth.ESXConfigPath+"\")")
	sv(&kola.ESXOptions.Server, "esx-server", "", "ESX server")
	sv(&kola.ESXOptions.Profile, "esx-profile", "", "ESX profile (default \"default\")")
	sv(&kola.ESXOptions.BaseVMName, "esx-base-vm", "", "ESX base VM name")
	sv(&kola.ESXOptions.OvaPath, "esx-ova-path", "", "ESX VM image to upload instead of using the base VM (build with: ./image_to_vm.sh --format=vmware_ova ...)")
	root.PersistentFlags().IntVarP(&kola.ESXOptions.StaticIPs, "esx-static-ips", "", 0, "Instead of DHCP, use this amount of static IP addresses")
	sv(&kola.ESXOptions.StaticGatewayIp, "esx-gateway", "", "Public gateway (only needed for static IP addresses)")
	sv(&kola.ESXOptions.StaticGatewayIpPrivate, "esx-gateway-private", "", "Private gateway (only needed for static IP addresses)")
	sv(&kola.ESXOptions.FirstStaticIp, "esx-first-static-ip", "", "First available public IP (only needed for static IP addresses)")
	sv(&kola.ESXOptions.FirstStaticIpPrivate, "esx-first-static-ip-private", "", "First available private IP (only needed for static IP addresses)")
	root.PersistentFlags().IntVarP(&kola.ESXOptions.StaticSubnetSize, "esx-subnet-size", "", 0, "Subnet size (only needed for static IP addresses)")

	// external-specific options
	sv(&kola.ExternalOptions.ManagementUser, "external-user", "", "External platform management SSH user")
	sv(&kola.ExternalOptions.ManagementPassword, "external-password", "", "External platform management SSH password")
	sv(&kola.ExternalOptions.ManagementHost, "external-host", "", "External platform management SSH host in the format HOST:PORT")
	sv(&kola.ExternalOptions.ManagementSocks, "external-socks", "", "External platform management SSH via SOCKS5 proxy in the format HOST:PORT (optional)")
	sv(&kola.ExternalOptions.ProvisioningCmds, "external-provisioning-cmds", "", "External platform provisioning commands ran on management SSH host. Has access to variable USERDATA with ignition config (can serve it via pxe http server for ignition.config.url or use as contents of FILE in 'flatcar-install -i FILE'). Note: It should mask sshd.(service|socket) for any booted PXE installer, and handle setting to boot from disk, as well as finding a free device and print its IP address as sole stdout content.")
	sv(&kola.ExternalOptions.SerialConsoleCmd, "external-serial-console-cmd", "", "External platform serial console attach command ran on management SSH host. Has access to the variable IPADDR to identify the node.")
	sv(&kola.ExternalOptions.DeprovisioningCmds, "external-deprovisioning-cmds", "", "External platform deprovisioning commands ran on management SSH host. Has access to the variable IPADDR to identify the node.")

	// gce-specific options
	sv(&kola.GCEOptions.Image, "gce-image", "projects/coreos-cloud/global/images/family/coreos-alpha", "GCE image, full api endpoints names are accepted if resource is in a different project")
	sv(&kola.GCEOptions.Project, "gce-project", "flatcar-212911", "GCE project name")
	sv(&kola.GCEOptions.Zone, "gce-zone", "us-central1-a", "GCE zone name")
	sv(&kola.GCEOptions.MachineType, "gce-machinetype", "n1-standard-1", "GCE machine type")
	sv(&kola.GCEOptions.DiskType, "gce-disktype", "pd-ssd", "GCE disk type")
	sv(&kola.GCEOptions.Network, "gce-network", "default", "GCE network")
	bv(&kola.GCEOptions.ServiceAuth, "gce-service-auth", false, "for non-interactive auth when running within GCE")
	sv(&kola.GCEOptions.JSONKeyFile, "gce-json-key", "", "use a service account's JSON key for authentication")

	// openstack-specific options
	sv(&kola.OpenStackOptions.ConfigPath, "openstack-config-file", "", "OpenStack config file (default \"~/"+auth.OpenStackConfigPath+"\")")
	sv(&kola.OpenStackOptions.Profile, "openstack-profile", "", "OpenStack profile (default \"default\")")
	sv(&kola.OpenStackOptions.Region, "openstack-region", "", "OpenStack region")
	sv(&kola.OpenStackOptions.Image, "openstack-image", "", "OpenStack image ref")
	sv(&kola.OpenStackOptions.Flavor, "openstack-flavor", "1", "OpenStack flavor ref")
	sv(&kola.OpenStackOptions.Network, "openstack-network", "", "OpenStack network")
	sv(&kola.OpenStackOptions.Domain, "openstack-domain", "", "OpenStack domain ID")
	sv(&kola.OpenStackOptions.FloatingIPPool, "openstack-floating-ip-pool", "", "OpenStack floating IP pool for Compute v2 networking")

	// packet-specific options
	sv(&kola.PacketOptions.ConfigPath, "packet-config-file", "", "Packet config file (default \"~/"+auth.PacketConfigPath+"\")")
	sv(&kola.PacketOptions.Profile, "packet-profile", "", "Packet profile (default \"default\")")
	sv(&kola.PacketOptions.ApiKey, "packet-api-key", "", "Packet API key (overrides config file)")
	sv(&kola.PacketOptions.Project, "packet-project", "", "Packet project UUID (overrides config file)")
	sv(&kola.PacketOptions.Facility, "packet-facility", "sjc1", "Packet facility code")
	sv(&kola.PacketOptions.Plan, "packet-plan", "", "Packet plan slug (default board-dependent, e.g. \"baremetal_0\")")
	sv(&kola.PacketOptions.InstallerImageBaseURL, "packet-installer-image-base-url", "", "Packet installer image base URL, non-https (default board-dependent, e.g. \"http://stable.release.flatcar-linux.net/amd64-usr/current\")")
	sv(&kola.PacketOptions.InstallerImageKernelURL, "packet-installer-image-kernel-url", "", "Packet installer image kernel URL, (default packet-installer-image-base-url/flatcar_production_pxe.vmlinuz)")
	sv(&kola.PacketOptions.InstallerImageCpioURL, "packet-installer-image-cpio-url", "", "Packet installer image cpio URL, (default packet-installer-image-base-url/flatcar_production_pxe_image.cpio.gz)")
	sv(&kola.PacketOptions.ImageURL, "packet-image-url", "", "Packet image URL (default board-dependent, e.g. \"https://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_packet_image.bin.bz2\")")
	sv(&kola.PacketOptions.StorageURL, "packet-storage-url", "gs://users.developer.core-os.net/"+os.Getenv("USER")+"/mantle", "Google Storage base URL for temporary uploads")

	// QEMU-specific options
	sv(&kola.QEMUOptions.Board, "board", defaultTargetBoard, "target board")
	sv(&kola.QEMUOptions.DiskImage, "qemu-image", "", "path to CoreOS disk image")
	sv(&kola.QEMUOptions.BIOSImage, "qemu-bios", "", "BIOS to use for QEMU vm")
	bv(&kola.QEMUOptions.UseVanillaImage, "qemu-skip-mangle", false, "don't modify CL disk image to capture console log")
}

// Sync up the command line options if there is dependency
func syncOptions() error {
	// sync `Board` option with other cloud provider
	// it seems kola has a strong dependency to qemu and it has been
	// build around that's why the `Board` is associated to `QEMU`
	// but it can be helpful for other provider to get access to the Board in the runtime
	board := kola.QEMUOptions.Board
	kola.OpenStackOptions.Board = board
	kola.GCEOptions.Board = board
	kola.ESXOptions.Board = board
	kola.ExternalOptions.Board = board
	kola.DOOptions.Board = board
	kola.AzureOptions.Board = board
	kola.AWSOptions.Board = board
	kola.PacketOptions.Board = board
	kola.PacketOptions.GSOptions = &kola.GCEOptions

	validateOption := func(name, item string, valid []string) error {
		for _, v := range valid {
			if v == item {
				return nil
			}
		}
		return fmt.Errorf("unsupported %v %q", name, item)
	}

	if err := validateOption("platform", kolaPlatform, kolaPlatforms); err != nil {
		return err
	}

	if err := validateOption("channel", kolaChannel, kolaChannels); err != nil {
		return err
	}

	if err := validateOption("offering", kolaOffering, kolaOfferings); err != nil {
		return err
	}

	if err := validateOption("distro", kola.Options.Distribution, kolaDistros); err != nil {
		return err
	}

	image, ok := kolaDefaultImages[kola.QEMUOptions.Board]
	if !ok {
		return fmt.Errorf("unsupport board %q", kola.QEMUOptions.Board)
	}

	if kola.QEMUOptions.DiskImage == "" {
		kola.QEMUOptions.DiskImage = image
	}

	if kola.QEMUOptions.BIOSImage == "" {
		kola.QEMUOptions.BIOSImage = kolaDefaultBIOS[kola.QEMUOptions.Board]
	}
	units, _ := root.PersistentFlags().GetStringSlice("debug-systemd-units")
	for _, unit := range units {
		kola.Options.SystemdDropins = append(kola.Options.SystemdDropins, platform.SystemdDropin{
			Unit:     unit,
			Name:     "10-debug.conf",
			Contents: "[Service]\nEnvironment=SYSTEMD_LOG_LEVEL=debug",
		})
	}

	if kola.Options.OSContainer != "" && kola.Options.Distribution != "rhcos" {
		return fmt.Errorf("oscontainer is only supported on rhcos")
	}

	if kola.Options.IgnitionVersion == "" {
		kola.Options.IgnitionVersion, ok = kolaIgnitionVersionDefaults[kola.Options.Distribution]
		if !ok {
			return fmt.Errorf("Distribution %q has no default Ignition version", kola.Options.Distribution)
		}
	}

	return nil
}

func GetSSHKeys(sshKeys []string) ([]agent.Key, error) {
	var allKeys []agent.Key
	// if no keys specified, use keys from agent plus ~/.ssh/id_{rsa,dsa,ecdsa,ed25519}.pub
	if len(sshKeys) == 0 {
		// add keys directly from the agent
		agentEnv := os.Getenv("SSH_AUTH_SOCK")
		if agentEnv != "" {
			f, err := net.Dial("unix", agentEnv)
			if err != nil {
				return nil, fmt.Errorf("Couldn't connect to unix socket %q: %v", agentEnv, err)
			}
			defer f.Close()

			agent := agent.NewClient(f)
			keys, err := agent.List()
			if err != nil {
				return nil, fmt.Errorf("Couldn't talk to ssh-agent: %v", err)
			}
			for _, key := range keys {
				allKeys = append(allKeys, *key)
			}
		}

		// populate list of key files
		userInfo, err := user.Current()
		if err != nil {
			return nil, err
		}
		for _, name := range []string{"id_rsa.pub", "id_dsa.pub", "id_ecdsa.pub", "id_ed25519.pub"} {
			path := filepath.Join(userInfo.HomeDir, ".ssh", name)
			if _, err := os.Stat(path); err == nil {
				sshKeys = append(sshKeys, path)
			}
		}
	}
	// read key files, failing if any are missing
	for _, path := range sshKeys {
		keybytes, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		pkey, comment, _, _, err := ssh.ParseAuthorizedKey(keybytes)
		if err != nil {
			return nil, err
		}
		key := agent.Key{
			Format:  pkey.Type(),
			Blob:    pkey.Marshal(),
			Comment: comment,
		}
		allKeys = append(allKeys, key)
	}

	return allKeys, nil
}
