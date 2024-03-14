package misc

import (
	"fmt"
	"os"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/platform/machine/qemu"
	"github.com/flatcar/mantle/platform/machine/unprivqemu"
	"github.com/flatcar/mantle/system/exec"
	"github.com/flatcar/mantle/util"
)

const (
	IgnitionConfigRootTPM = `{
		"ignition": {
			"config": {},
			"timeouts": {},
			"version": "3.3.0"
		},
		"kernelArguments": {
			"shouldExist": ["rd.luks.name=12345678-9abc-def0-1234-56789abcdef0=rootencrypted", "systemd.mask=systemd-cryptsetup@rootencrypted.service"]
		},
		"storage": {
			"luks": [
				{
					"name": "rootencrypted",
					"device": "/dev/disk/by-partlabel/ROOT",
					"uuid": "12345678-9abc-def0-1234-56789abcdef0",
					"wipeVolume": true,
					"clevis": {
						"tpm2": true
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

	IgnitionConfigNonRootTPM = `{
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
						"tpm2": true
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
	runRootTPM := func(c cluster.TestCluster) {
		tpmTest(c, conf.Ignition(IgnitionConfigRootTPM), "/")
	}
	register.Register(&register.Test{
		Run:         runRootTPM,
		ClusterSize: 0,
		Platforms:   []string{"qemu"},
		Name:        "cl.tpm.root",
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 3880},
	})

	runNonRootTPM := func(c cluster.TestCluster) {
		tpmTest(c, conf.Ignition(IgnitionConfigNonRootTPM), "/mnt/data")
	}
	register.Register(&register.Test{
		Run:         runNonRootTPM,
		ClusterSize: 0,
		Platforms:   []string{"qemu"},
		Name:        "cl.tpm.nonroot",
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 3880},
	})
}

func tpmTest(c cluster.TestCluster, userData *conf.UserData, mountpoint string) {
	swtpm, err := startSwtpm()
	if err != nil {
		c.Fatalf("could not start software TPM emulation: %v", err)
	}
	defer swtpm.stop()

	options := platform.MachineOptions{
		AdditionalDisks: []platform.Disk{
			{Size: "520M", DeviceOpts: []string{"serial=secondary"}},
		},
		SoftwareTPMSocket: swtpm.socketPath,
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

type softwareTPM struct {
	process    *exec.ExecCmd
	socketPath string
	dir        string
}

func startSwtpm() (*softwareTPM, error) {
	swtpm := &softwareTPM{}

	swtpmDir, err := os.MkdirTemp("", "swtpm-")
	if err != nil {
		return nil, err
	}
	swtpm.dir = swtpmDir
	swtpm.socketPath = fmt.Sprintf("%v/swtpm-sock", swtpm.dir)

	swtpm.process = exec.Command("swtpm", "socket", "--tpmstate", fmt.Sprintf("dir=%v", swtpm.dir), "--ctrl", fmt.Sprintf("type=unixio,path=%v", swtpm.socketPath), "--tpm2")
	out, err := swtpm.process.StdoutPipe()
	if err != nil {
		return nil, err
	}
	go util.LogFrom(capnslog.INFO, out)

	if err = swtpm.process.Start(); err != nil {
		return nil, err
	}

	plog.Debugf("swtpm PID: %v", swtpm.process.Pid())

	return swtpm, nil
}

func (swtpm *softwareTPM) stop() {
	if err := swtpm.process.Kill(); err != nil {
		plog.Errorf("Error killing swtpm: %v", err)
	}
	// To be double sure that we do not delete the wrong directory, check that "tpm" occurs in the directory path we delete.
	if strings.Contains(swtpm.dir, "tpm") {
		plog.Debugf("Delete swtpm temporary directory %v", swtpm.dir)
		os.RemoveAll(swtpm.dir)
	}
}
