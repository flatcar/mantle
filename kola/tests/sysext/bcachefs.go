// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package sysext

import (
	"fmt"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/kola/tests/util"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
)

// The bcachefs-setup.service unit runs on every boot and is idempotent:
// it only formats the disk if it isn't already bcachefs, and only mounts
// /var/lib/bcachefs if it isn't already a mount point. This mirrors what the
// zfs test relies on zpool import + zfs-mount.service for, but bcachefs has
// no equivalent auto-import service so we drive mkfs+mount from a unit.
//
// The bcachefs.ko kernel module ships in the base image (built in-tree with
// coreos-modules), so it is available regardless of whether the sysext is
// merged. The sysext only provides the mkfs.bcachefs / mount.bcachefs
// userspace tools.
var bcachefsUserData = conf.Butane(`
variant: flatcar
version: 1.0.0
storage:
  files:
  - path: /etc/flatcar/enabled-sysext.conf
    contents:
      inline: |
        bcachefs
systemd:
  units:
  - name: bcachefs-setup.service
    enabled: true
    contents: |
      [Unit]
      Description=Format and mount bcachefs test volume
      After=systemd-sysext.service ensure-sysext.service
      ConditionPathIsMountPoint=!/var/lib/bcachefs

      [Service]
      Type=oneshot
      RemainAfterExit=yes
      ExecStart=/bin/sh -c 'blkid -t TYPE=bcachefs /dev/disk/by-id/virtio-bcachefs || mkfs.bcachefs -f /dev/disk/by-id/virtio-bcachefs'
      ExecStart=/bin/mkdir -p /var/lib/bcachefs
      ExecStart=/bin/mount -t bcachefs /dev/disk/by-id/virtio-bcachefs /var/lib/bcachefs

      [Install]
      WantedBy=multi-user.target
`)

var bcachefsNfsUserData = conf.Butane(`
variant: flatcar
version: 1.0.0
storage:
  files:
  - path: /etc/flatcar/enabled-sysext.conf
    contents:
      inline: |
        bcachefs
  - path: /etc/exports
    mode: 0644
    contents:
      inline: |
        /var/lib/bcachefs *(rw,no_root_squash,no_subtree_check,fsid=0)
systemd:
  units:
  - name: nfs-server.service
    enabled: true
    dropins:
    - name: bcachefs.conf
      contents: |
        [Unit]
        After=bcachefs-setup.service
        Requires=bcachefs-setup.service
  - name: bcachefs-setup.service
    enabled: true
    contents: |
      [Unit]
      Description=Format and mount bcachefs test volume
      After=systemd-sysext.service ensure-sysext.service
      ConditionPathIsMountPoint=!/var/lib/bcachefs

      [Service]
      Type=oneshot
      RemainAfterExit=yes
      ExecStart=/bin/sh -c 'blkid -t TYPE=bcachefs /dev/disk/by-id/virtio-bcachefs || mkfs.bcachefs -f /dev/disk/by-id/virtio-bcachefs'
      ExecStart=/bin/mkdir -p /var/lib/bcachefs
      ExecStart=/bin/mount -t bcachefs /dev/disk/by-id/virtio-bcachefs /var/lib/bcachefs

      [Install]
      WantedBy=multi-user.target
`)

func init() {
	register.Register(&register.Test{
		Name:        "sysext.bcachefs.reboot",
		Run:         checkSysextBcachefs,
		ClusterSize: 0,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv", "azure"},
		// Bump to the first Flatcar release that carries the bcachefs sysext
		// and the in-tree bcachefs.ko kernel module.
		MinVersion: semver.Version{Major: 4374},
		SkipFunc:   skipBcachefs,
	})

	register.Register(&register.Test{
		Name:        "sysext.bcachefs.nfs",
		Run:         checkBcachefsNfs,
		ClusterSize: 0,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms:  []string{"qemu", "qemu-unpriv", "azure"},
		MinVersion: semver.Version{Major: 4374},
		SkipFunc:   skipBcachefs,
	})
}

func skipBcachefs(version semver.Version, channel, arch, platform string) bool {
	// The initial bcachefs sysext ships amd64-only; skip arm64 until the
	// sysext is published for arm64 too.
	if arch == "arm64" {
		return true
	}
	return kola.SkipSecureboot(version, channel, arch, platform) || skipOnGha(version, channel, arch, platform)
}

func createBcachefsMachine(c cluster.TestCluster, userdata *conf.UserData) platform.Machine {
	options := platform.MachineOptions{
		AdditionalDisks: []platform.Disk{
			{Size: "1G", DeviceOpts: []string{"serial=bcachefs"}},
		},
	}
	m, err := util.NewMachineWithOptions(c, userdata, options)
	if err != nil {
		c.Fatalf("creating a machine failed: %v", err)
	}
	return m
}

func checkSysextBcachefs(c cluster.TestCluster) {
	m := createBcachefsMachine(c, bcachefsUserData)
	c.AssertCmdOutputContains(m, "findmnt /var/lib/bcachefs", "bcachefs")
	c.AssertCmdOutputContains(m, "lsmod", "bcachefs")
	c.AssertCmdOutputContains(m, "grep bcachefs /proc/filesystems", "bcachefs")
	c.MustSSH(m, "sudo chown core /var/lib/bcachefs/ && echo world >/var/lib/bcachefs/hello")
	err := m.Reboot()
	if err != nil {
		c.Fatalf("could not reboot: %v", err)
	}
	c.AssertCmdOutputContains(m, "lsmod", "bcachefs")
	c.AssertCmdOutputContains(m, "findmnt /var/lib/bcachefs", "bcachefs")
	c.AssertCmdOutputContains(m, "grep --with-filename . /var/lib/bcachefs/hello", "hello:world")
}

func checkBcachefsNfs(c cluster.TestCluster) {
	m := createBcachefsMachine(c, bcachefsNfsUserData)
	c.AssertCmdOutputContains(m, "findmnt /var/lib/bcachefs", "bcachefs")
	c.MustSSH(m, "sudo chown core /var/lib/bcachefs/")
	c.MustSSH(m, "echo world >/var/lib/bcachefs/hello")
	m2, err := c.NewMachine(nfsClientUserData)
	if err != nil {
		c.Fatalf("Cluster.NewMachine: %s", err)
	}
	c.MustSSH(m2, "sudo mkdir /mnt/nfs")
	c.MustSSH(m2, fmt.Sprintf("sudo mount -t nfs -o nfsvers=4 %s:/var/lib/bcachefs /mnt/nfs", m.PrivateIP()))
	c.AssertCmdOutputContains(m2, "cat /mnt/nfs/hello", "world")
}
