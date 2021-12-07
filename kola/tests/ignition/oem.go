// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package ignition

import (
	"github.com/coreos/go-semver/semver"
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
)

func init() {
	register.Register(&register.Test{
		Name:        "cl.ignition.oem.regular",
		Run:         reusePartition,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 2983},
		UserData: conf.ContainerLinuxConfig(`storage:
  filesystems:
     - name: oem
       mount:
         device: "/dev/disk/by-label/OEM"
         format: "btrfs"
  files:
    - path: /grub.cfg
      filesystem: oem
      mode: 0644
      contents:
        inline: |
          set linux_append="flatcar.autologin"`),
	})
	register.Register(&register.Test{
		Name:        "cl.ignition.oem.reuse",
		Run:         reusePartition,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 2983},
		UserData: conf.ContainerLinuxConfig(`storage:
  filesystems:
     - name: oem
       mount:
         device: "/dev/disk/by-label/OEM"
         format: "ext4"
  files:
    - path: /grub.cfg
      filesystem: oem
      mode: 0644
      contents:
        inline: |
          set linux_append="flatcar.autologin"`),
	})
	register.Register(&register.Test{
		Name:        "cl.ignition.oem.wipe",
		Run:         wipeOEM,
		MinVersion:  semver.Version{Major: 2983},
		ClusterSize: 1,
		// `wiping` the OEM file system does not allow the instance to boot on platforms
		// different from QEMU.
		// More details: https://github.com/flatcar-linux/Flatcar/issues/514.
		Platforms: []string{"qemu", "qemu-unpriv"},
		UserData: conf.ContainerLinuxConfig(`storage:
  filesystems:
     - name: oem
       mount:
         device: "/dev/disk/by-label/OEM"
         format: "ext4"
         wipe_filesystem: true
         label: "OEM"
  files:
    - path: /grub.cfg
      filesystem: oem
      mode: 0644
      contents:
        inline: |
          set linux_append="flatcar.autologin"`),
	})
}

// reusePartition asserts that even if the config uses a different fs format, we keep using `btrfs`.
func reusePartition(c cluster.TestCluster) {
	out := c.MustSSH(c.Machines()[0], `lsblk --output FSTYPE,LABEL,MOUNTPOINT --json | jq -r '.blockdevices | .[] | select(.label=="OEM") | .fstype'`)

	if string(out) != "btrfs" {
		c.Fatalf("should get btrfs, got: %s", string(out))
	}
}

// wipeOEM asserts that if the config uses a different fs format with a wipe of the fs we effectively wipe the fs.
func wipeOEM(c cluster.TestCluster) {
	out := c.MustSSH(c.Machines()[0], `lsblk --output FSTYPE,LABEL,MOUNTPOINT --json | jq -r '.blockdevices | .[] | select(.label=="OEM") | .fstype'`)

	if string(out) != "ext4" {
		c.Fatalf("should get ext4, got: %s", string(out))
	}
}
