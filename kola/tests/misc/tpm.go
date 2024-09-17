package misc

import (
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/platform/machine/qemu"
	"github.com/flatcar/mantle/platform/machine/unprivqemu"
)

const (
	VariantDefault    string = ""
	VariantNoUpdate   string = "noupdate"
	VariantWithUpdate string = "withupdate"
)

var (
	// For now Ignition has no systemd-cryptenroll support and a helper service is used
	IgnitionConfigRootCryptenroll = conf.Butane(`---
variant: flatcar
version: 1.0.0
storage:
  luks:
    - name: rootencrypted
      wipe_volume: true
      device: "/dev/disk/by-partlabel/ROOT"
  filesystems:
    - device: /dev/mapper/rootencrypted
      format: ext4
      label: ROOT
systemd:
  units:
    - name: cryptenroll-helper.service
      enabled: true
      contents: |
        [Unit]
        ConditionFirstBoot=true
        OnFailure=emergency.target
        OnFailureJobMode=isolate
        [Service]
        Type=oneshot
        RemainAfterExit=yes
        ExecStart=systemd-cryptenroll --tpm2-device=auto --unlock-key-file=/etc/luks/rootencrypted --wipe-slot=0 /dev/disk/by-partlabel/ROOT
        ExecStart=rm /etc/luks/rootencrypted
        [Install]
        WantedBy=multi-user.target
`)
	// Note: Keep the two below configs in sync with those
	// documented in the Flatcar TPM docs.
	// Ideally the reboot wouldn't be needed (or done in the initrd?)
	IgnitionConfigRootCryptenrollPcrNoUpdate = conf.Butane(`---
variant: flatcar
version: 1.0.0
storage:
  files:
    - path: /etc/flatcar/update.conf
      overwrite: true
      contents:
        inline: |
          SERVER=disabled
  luks:
    - name: rootencrypted
      wipe_volume: true
      device: "/dev/disk/by-partlabel/ROOT"
  filesystems:
    - device: /dev/mapper/rootencrypted
      format: ext4
      label: ROOT
systemd:
  units:
    - name: cryptenroll-helper-first.service
      enabled: true
      contents: |
        [Unit]
        ConditionFirstBoot=true
        OnFailure=emergency.target
        OnFailureJobMode=isolate
        After=first-boot-complete.target multi-user.target
        [Service]
        Type=oneshot
        RemainAfterExit=yes
        ExecStart=systemd-cryptenroll --tpm2-device=auto --unlock-key-file=/etc/luks/rootencrypted --tpm2-pcrs= /dev/disk/by-partlabel/ROOT
        ExecStart=mv /etc/luks/rootencrypted /etc/luks/rootencrypted-bind
        ExecStart=sleep 10
        ExecStart=systemctl reboot
        [Install]
        WantedBy=multi-user.target
    - name: cryptenroll-helper-bind.service
      enabled: true
      contents: |
        [Unit]
        ConditionFirstBoot=false
        ConditionPathExists=/etc/luks/rootencrypted-bind
        OnFailure=emergency.target
        OnFailureJobMode=isolate
        [Service]
        Type=oneshot
        RemainAfterExit=yes
        ExecStart=systemd-cryptenroll --tpm2-device=auto --unlock-key-file=/etc/luks/rootencrypted-bind --tpm2-pcrs=4+7+8+9+11+12+13 --wipe-slot=tpm2 /dev/disk/by-partlabel/ROOT
        ExecStart=mv /etc/luks/rootencrypted-bind /etc/luks/rootencrypted
        [Install]
        WantedBy=multi-user.target
`)
	// The rebinding for the update is due to how GRUB measures things and
	// we can only make this work without rebinding if we switch to sd-boot
	IgnitionConfigRootCryptenrollPcrWithUpdate = conf.Butane(`---
variant: flatcar
version: 1.0.0
storage:
  files:
    - path: /oem/bin/oem-postinst
      overwrite: true
      mode: 0755
      contents:
        inline: |
          #!/bin/bash
          set -euo pipefail
          # When the update fails to correctly apply, this runs again
          if [ -e /etc/luks/rootencrypted-bound ]; then
            mv /etc/luks/rootencrypted-bound /etc/luks/rootencrypted-bind
          fi
          # But since a reboot inbetween could have bound it again,
          # remove the PCR binding for every run
          systemd-cryptenroll --tpm2-device=auto --unlock-key-file=/etc/luks/rootencrypted-bind --wipe-slot=tpm2 --tpm2-pcrs= /dev/disk/by-partlabel/ROOT
  luks:
    - name: rootencrypted
      wipe_volume: true
      device: "/dev/disk/by-partlabel/ROOT"
  filesystems:
    - device: /dev/mapper/rootencrypted
      format: ext4
      label: ROOT
systemd:
  units:
    - name: cryptenroll-helper-first.service
      enabled: true
      contents: |
        [Unit]
        ConditionFirstBoot=true
        OnFailure=emergency.target
        OnFailureJobMode=isolate
        After=first-boot-complete.target multi-user.target
        [Service]
        Type=oneshot
        RemainAfterExit=yes
        ExecStart=systemd-cryptenroll --tpm2-device=auto --unlock-key-file=/etc/luks/rootencrypted --tpm2-pcrs= /dev/disk/by-partlabel/ROOT
        ExecStart=mv /etc/luks/rootencrypted /etc/luks/rootencrypted-bind
        ExecStart=sleep 10
        ExecStart=systemctl reboot
        [Install]
        WantedBy=multi-user.target
    - name: cryptenroll-helper-bind.service
      enabled: true
      contents: |
        [Unit]
        ConditionFirstBoot=false
        ConditionPathExists=/etc/luks/rootencrypted-bind
        OnFailure=emergency.target
        OnFailureJobMode=isolate
        Before=update-engine.service
        [Service]
        Type=oneshot
        RemainAfterExit=yes
        ExecStart=systemd-cryptenroll --tpm2-device=auto --unlock-key-file=/etc/luks/rootencrypted-bind --tpm2-pcrs=4+7+8+9+11+12+13 --wipe-slot=tpm2 /dev/disk/by-partlabel/ROOT
        ExecStart=mv /etc/luks/rootencrypted-bind /etc/luks/rootencrypted-bound
        [Install]
        WantedBy=multi-user.target
`)
	IgnitionConfigRootTPM = `{
		"ignition": {
			"config": {},
			"timeouts": {},
			"version": "3.3.0"
		},
		"storage": {
			"luks": [
				{
					"name": "rootencrypted",
					"device": "/dev/disk/by-partlabel/ROOT",
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
	runRootTPMCryptenroll := func(c cluster.TestCluster) {
		tpmTest(c, IgnitionConfigRootCryptenroll, "/", VariantDefault)
	}
	register.Register(&register.Test{
		Run:         runRootTPMCryptenroll,
		ClusterSize: 0,
		Platforms:   []string{"qemu"},
		Name:        "cl.tpm.root-cryptenroll",
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 3913, Minor: 0, Patch: 1},
	})
	runRootTPMCryptenrollPcrNoUpdate := func(c cluster.TestCluster) {
		tpmTest(c, IgnitionConfigRootCryptenrollPcrNoUpdate, "/", VariantNoUpdate)
	}
	register.Register(&register.Test{
		Run:         runRootTPMCryptenrollPcrNoUpdate,
		ClusterSize: 0,
		Platforms:   []string{"qemu"},
		Name:        "cl.tpm.root-cryptenroll-pcr-noupdate",
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 3913, Minor: 0, Patch: 1},
	})
	runRootTPMCryptenrollPcrWithUpdate := func(c cluster.TestCluster) {
		tpmTest(c, IgnitionConfigRootCryptenrollPcrWithUpdate, "/", VariantWithUpdate)
	}
	register.Register(&register.Test{
		Run:         runRootTPMCryptenrollPcrWithUpdate,
		ClusterSize: 0,
		Platforms:   []string{"qemu"},
		Name:        "cl.tpm.root-cryptenroll-pcr-withupdate",
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 3913, Minor: 0, Patch: 1},
	})

	runRootTPM := func(c cluster.TestCluster) {
		tpmTest(c, conf.Ignition(IgnitionConfigRootTPM), "/", VariantDefault)
	}
	register.Register(&register.Test{
		Run:         runRootTPM,
		ClusterSize: 0,
		Platforms:   []string{"qemu"},
		Name:        "cl.tpm.root",
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 3913, Minor: 0, Patch: 1},
	})

	runNonRootTPM := func(c cluster.TestCluster) {
		tpmTest(c, conf.Ignition(IgnitionConfigNonRootTPM), "/mnt/data", VariantDefault)
	}
	register.Register(&register.Test{
		Run:         runNonRootTPM,
		ClusterSize: 0,
		Platforms:   []string{"qemu"},
		Name:        "cl.tpm.nonroot",
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 3913, Minor: 0, Patch: 1},
	})

	register.Register(&register.Test{
		Run:         eventLogTest,
		ClusterSize: 0,
		Platforms:   []string{"qemu"},
		Name:        "cl.tpm.eventlog",
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 4082},
	})
}

func tpmTest(c cluster.TestCluster, userData *conf.UserData, mountpoint string, variant string) {
	options := platform.MachineOptions{
		AdditionalDisks: []platform.Disk{
			{Size: "520M", DeviceOpts: []string{"serial=secondary"}},
		},
		EnableTPM: true,
	}
	var m platform.Machine
	var err error
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

	if variant == VariantNoUpdate || variant == VariantWithUpdate {
		// Wait for the first reboot
		time.Sleep(1 * time.Minute)
		// Verify that the machine rebooted
		_ = c.MustSSH(m, "grep -v flatcar.first_boot /proc/cmdline")
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

	if variant == VariantWithUpdate {
		// Simulate update effect by
		// affecting the GRUB PCR values for the next boot
		_ = c.MustSSH(m, "echo 'set linux_append=\"flatcar.autologin console=ttyS0,115200 quiet\"' | sudo tee -a /oem/grub.cfg")
		// and calling the OEM hook we set up
		_ = c.MustSSH(m, "sudo /oem/bin/oem-postinst")
		err := m.Reboot()
		if err != nil {
			c.Fatalf("could not reboot machine: %v", err)
		}
		checkIfMountpointIsEncrypted(c, m, "/")
	}
}

func eventLogTest(c cluster.TestCluster) {
	options := platform.MachineOptions{EnableTPM: true}
	var (
		m   platform.Machine
		err error
	)
	switch pc := c.Cluster.(type) {
	// These cases have to be separated because otherwise the golang compiler doesn't type-check
	// the case bodies using the proper subtype of `pc`.
	case *qemu.Cluster:
		m, err = pc.NewMachineWithOptions(nil, options)
	case *unprivqemu.Cluster:
		m, err = pc.NewMachineWithOptions(nil, options)
	default:
		c.Fatal("unknown cluster type")
	}
	if err != nil {
		c.Fatal(err)
	}

	// Verify that the TPM event log is working.
	_ = c.MustSSH(m, "sudo tpm2_eventlog /sys/kernel/security/tpm0/binary_bios_measurements")
}
