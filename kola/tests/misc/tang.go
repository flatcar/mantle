package misc

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/coreos/go-semver/semver"

	"io/fs"

	"github.com/anatol/tang.go"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/kola/tests/util"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/platform/machine/qemu"
	"github.com/flatcar/mantle/platform/machine/unprivqemu"
)

const (
	IgnitionConfigRootTang = `{
		"ignition": {
			"config": {},
			"timeouts": {},
			"version": "3.3.0"
		},
		"kernelArguments": {
			"shouldExist": ["rd.luks.name=12345678-9abc-def0-1234-56789abcdef0=rootencrypted", "rd.networkd=1", "systemd.mask=systemd-cryptsetup@rootencrypted.service"]
		},
		"storage": {
			"luks": [
				{
					"name": "rootencrypted",
					"device": "/dev/disk/by-partlabel/ROOT",
					"uuid": "12345678-9abc-def0-1234-56789abcdef0",
					"wipeVolume": true,
					"clevis": {
						"tang": [
							{
								"url": "http://{{ .TangIP }}:{{ .TangPort }}",
								"thumbprint": "HkwVNDeKhzaVqWhXtXwEIGNILRZt4cBWWb0kI1-a0NM"
							}
						]
					}
				}
			],
			"filesystems": [
				{
					"device": "/dev/disk/by-id/dm-name-rootencrypted",
					"format": "ext4",
					"label": "ROOT"
				}
			]
		}
	}`

	IgnitionConfigNonRootTang = `{
		"ignition": {
			"config": {},
			"timeouts": {},
			"version": "3.3.0"
		},
		"storage": {
			"disks": [
				{
					"device": "/dev/disk/by-id/virtio-secondary",
					"wipeTable": true,
					"partitions": [
						{
							"label": "data",
							"number": 1
						}
					]
				}
			],
			"luks": [
				{
					"name": "dataencrypted",
					"device": "/dev/disk/by-partlabel/data",
					"clevis": {
						"tang": [
							{
								"url": "http://{{ .TangIP }}:{{ .TangPort }}",
								"thumbprint": "HkwVNDeKhzaVqWhXtXwEIGNILRZt4cBWWb0kI1-a0NM"
							}
						]
					}
				}
			],
			"filesystems": [
				{
					"device": "/dev/disk/by-id/dm-name-dataencrypted",
					"format": "ext4",
					"label": "DATA",
					"path": "/mnt/data"
				}
			]
		},
		"systemd": {
			"units": [{
			"name": "mnt-data.mount",
			"enabled": true,
			"contents": "[Mount]\nWhat=/dev/disk/by-label/DATA\nWhere=/mnt/data\nType=ext4\n\n[Install]\nWantedBy=local-fs.target"
			}]
		}
	}`
)

func init() {
	// The Tang server started here will bind to an IP of a networking interface in the root networking namespace.
	// This works because traffic from inside the QEMU VM to that IP will be routed to the host.
	// If this causes issues in the future, we could alternatively add another TAP interface to the bridge and let
	// the Tang server bind to its IP. That would require the Tang setup to happen outside of these tests and
	// introduce complexity in different parts of the code base.
	tangPort := 8007
	tangIP, err := getIP()
	if err != nil {
		fmt.Printf("failed to find IP for the Tang server to bind to: %v\n", err)
		return
	}

	rootConfig, err := util.ExecTemplate(IgnitionConfigRootTang, map[string]string{
		"TangIP":   fmt.Sprintf("%v", tangIP),
		"TangPort": strconv.Itoa(tangPort),
	})
	if err != nil {
		fmt.Printf("failed to execute template: %v\n", err)
		return
	}
	runRootTang := func(c cluster.TestCluster) {
		tangTest(c, tangIP, tangPort, conf.Ignition(rootConfig), "/")
	}
	register.Register(&register.Test{
		Run:         runRootTang,
		ClusterSize: 0,
		Platforms:   []string{"qemu"},
		Name:        "cl.tang.root",
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 3880},
	})

	nonRootConfig, err := util.ExecTemplate(IgnitionConfigNonRootTang, map[string]string{
		"TangIP":   fmt.Sprintf("%v", tangIP),
		"TangPort": strconv.Itoa(tangPort),
	})
	if err != nil {
		fmt.Printf("failed to execute template: %v\n", err)
		return
	}
	runNonRootTang := func(c cluster.TestCluster) {
		tangTest(c, tangIP, tangPort, conf.Ignition(nonRootConfig), "/mnt/data")
	}
	register.Register(&register.Test{
		Run:         runNonRootTang,
		ClusterSize: 0,
		Platforms:   []string{"qemu"},
		Name:        "cl.tang.nonroot",
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 3880},
	})
}

func tangTest(c cluster.TestCluster, tangIP net.IP, tangPort int, userData *conf.UserData, mountpoint string) {
	terminateTangServer, err := startTang(tangIP, tangPort)
	if err != nil {
		c.Fatalf("could not start Tang server: %v", err)
	}
	defer terminateTangServer()

	options := platform.MachineOptions{
		AdditionalDisks: []platform.Disk{
			{Size: "520M", DeviceOpts: []string{"serial=secondary"}},
		},
	}

	var m platform.Machine
	switch pc := c.Cluster.(type) {
	// These cases have to be separated because otherwise the golang compiler doesn't type-check
	// the case bodies using the proper subtype of `pc`.
	case *qemu.Cluster:
		m, err = pc.NewMachineWithOptions(userData, options)
	case *unprivqemu.Cluster:
		m, err = pc.NewMachineWithOptions(userData, options)
	default:
		c.Fatal("unknown cluster type")
	}
	if err != nil {
		c.Fatal(err)
	}

	checkIfMountpointIsEncrypted(c, m, mountpoint)

	// Make sure the change is reboot-safe. This is especially important for the case of an encrypted root disk because the
	// initramfs decryption is not tested on the first boot, in which the initramfs starts with no encrypted disks and Ignition
	// only sets up the encryption while in initramfs.
	err = m.Reboot()
	if err != nil {
		c.Fatalf("could not reboot machine: %v", err)
	}

	checkIfMountpointIsEncrypted(c, m, mountpoint)
}

func checkIfMountpointIsEncrypted(c cluster.TestCluster, m platform.Machine, mountpoint string) {
	util.CheckMountpoint(c, m, mountpoint, func(b util.Blockdevice) bool { return b.Type == "crypt" })
}

func getIP() (net.IP, error) {
	networkInterfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, networkInterface := range networkInterfaces {
		addresses, err := networkInterface.Addrs()
		if err != nil {
			continue
		}
		ipAddress, ok := addresses[0].(*net.IPNet)
		if ok && networkInterface.Flags&net.FlagRunning != 0 && !ipAddress.IP.IsLoopback() && ipAddress.IP.To4() != nil {
			return ipAddress.IP, nil
		}
	}

	return nil, errors.New("failed to find an IP of a running network interface")
}

func startTang(ip net.IP, port int) (func(), error) {
	keyDirectory, err := makeTangKeyDirectory()
	if err != nil {
		return nil, err
	}

	srv := tang.NewServer()
	keySet, _ := tang.ReadKeys(keyDirectory)
	srv.Keys = keySet
	srv.Addr = fmt.Sprintf("%v:%v", ip, port)

	go func() {
		// ListenAndServe always returns a non-nil error. ErrServerClosed on graceful close
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatalf("Tang server returned error: %v", err)
		}
	}()

	terminateTangServer := func() {
		srv.Close()
		if strings.Contains(keyDirectory, "tang-db-") {
			os.RemoveAll(keyDirectory)
		}
	}

	return terminateTangServer, nil
}

func makeTangKeyDirectory() (string, error) {
	tangDir, err := os.MkdirTemp("", "tang-db-")
	if err != nil {
		return "", err
	}

	signKey := `{"alg": "ES512", "kty": "EC", "crv": "P-521", "x": "AIcFiZgNvNMYYTOMaRjFUEMGqaXe5JrSDeKe2cAp7B1sGJL8BMDaxJJuchN5kXrP_DEyFalB6n6LcOf8EPIblAXx", "y": "AAqlZU_AueDHMBF83McJboc-Fu-8z6c2X8_4BLcPdN61LH-u6mNT21QqcWnbP5FpcdgDeIkHgUU4-9q702dFyhs9", "d": "AK1qPAdmS55UoGIRTNVxVHjxYf4JknzUWNgO4sOQaoR7VbEkoZZesjxPBP52NlYsRAdeA3ZOZCsvI3qeUWh0tS2_", "key_ops": ["sign", "verify"]}`
	err = os.WriteFile(fmt.Sprintf("%v/HkwVNDeKhzaVqWhXtXwEIGNILRZt4cBWWb0kI1-a0NM.jwk", tangDir), []byte(signKey), fs.FileMode(0644))
	if err != nil {
		return "", err
	}

	deriveKey := `{"alg": "ECMR", "kty": "EC", "crv": "P-521", "x": "ACZadV-S4M2dNJMZS0mqgXqucAyMs_8nNwVRus8xq04WV26QPC3ab3n2kSSH1QIus3fIGoIZlglHSzFXZ8VnRTVM", "y": "AcnqORSJ_DPub2Js0vldfn3b79renKPP6f-Sb-oCCz4bc-JlN1muIB-MxvUCKDSbZvAVn9OTCifbyy1XIFJsYK6e", "d": "AMgFsJMyqSIbDA-eU3iIn-eYaXwhuDbLU_YrbupXeQZvHEnEJ0yWKx6U04W4-Gj_GO5iQUZs8taj81eS6QHPBc_4", "key_ops": ["deriveKey"]}`
	err = os.WriteFile(fmt.Sprintf("%v/0EP4pt0H7q-1fDEN70dCD__S_YVIu-bmrC5QMwongsU.jwk", tangDir), []byte(deriveKey), fs.FileMode(0644))
	if err != nil {
		return "", err
	}

	return tangDir, nil
}
