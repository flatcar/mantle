// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package sysext

import (
	"fmt"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/platform/machine/qemu"
	"github.com/flatcar/mantle/platform/machine/unprivqemu"
)

var zfsUserData = conf.Butane(`
variant: flatcar
version: 1.0.0
storage:
  files:
  - path: /etc/flatcar/enabled-sysext.conf
    contents:
      inline: |
        zfs
systemd:
  units:
  - name: zpool-create.service
    enabled: true
    contents: |
      [Unit]
      ConditionFirstBoot=1
      Before=first-boot-complete.target
      Wants=first-boot-complete.target

      [Service]
      Type=oneshot
      ExecStart=zpool create tank /dev/disk/by-id/virtio-zfs

      [Install]
      WantedBy=multi-user.target
`)

var zfsDockerUserData = conf.Butane(`
variant: flatcar
version: 1.0.0
storage:
  files:
  - path: /etc/flatcar/enabled-sysext.conf
    contents:
      inline: |
        zfs
  - path: /etc/docker/daemon.json
    contents:
      inline: |
        {
          "storage-driver": "zfs"
        }

systemd:
  units:
  - name: docker.service
    dropins:
    - name: zfs.conf
      contents: |
        [Unit]
        After=zfs.target

  - name: zpool-create.service
    enabled: true
    contents: |
      [Unit]
      ConditionFirstBoot=1
      Before=first-boot-complete.target
      Wants=first-boot-complete.target

      [Service]
      Type=oneshot
      ExecStart=zpool create tank /dev/disk/by-id/virtio-zfs
      ExecStart=zfs create -o mountpoint=/var/lib/docker tank/docker

      [Install]
      WantedBy=multi-user.target
`)

var zfsNfsUserData = conf.Butane(`
variant: flatcar
version: 1.0.0
storage:
  files:
  - path: /etc/flatcar/enabled-sysext.conf
    contents:
      inline: |
        zfs
systemd:
  units:
  - name: nfs-server.service
    enabled: true
  - name: zpool-create.service
    enabled: true
    contents: |
      [Unit]
      ConditionFirstBoot=1
      Before=first-boot-complete.target
      Wants=first-boot-complete.target

      [Service]
      Type=oneshot
      ExecStart=zpool create tank /dev/disk/by-id/virtio-zfs
      ExecStart=zfs create -o sharenfs=on tank/nfs

      [Install]
      WantedBy=multi-user.target
`)

var nfsClientUserData = conf.Butane(`
variant: flatcar
version: 1.0.0
systemd:
  units:
  - name: nfs-client.target
    enabled: true
`)

func init() {
	register.Register(&register.Test{
		Name:        "sysext.zfs.reboot",
		Run:         checkSysextZfs,
		ClusterSize: 0,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3902},
	})

	register.Register(&register.Test{
		Name:        "sysext.zfs.docker",
		Run:         checkZfsDocker,
		ClusterSize: 0,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3902},
	})

	register.Register(&register.Test{
		Name:        "sysext.zfs.nfs",
		Run:         checkZfsNfs,
		ClusterSize: 0,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3902},
	})
}

func createZfsMachine(c cluster.TestCluster, userdata *conf.UserData) platform.Machine {
	var m platform.Machine
	var err error
	options := platform.MachineOptions{
		AdditionalDisks: []platform.Disk{
			{Size: "1G", DeviceOpts: []string{"serial=zfs"}},
		},
	}
	switch pc := c.Cluster.(type) {
	// These cases have to be separated because when put together to the same case statement
	// the golang compiler no longer checks that the individual types in the case have the
	// NewMachineWithOptions function, but rather whether platform.Cluster does which fails
	case *qemu.Cluster:
		m, err = pc.NewMachineWithOptions(userdata, options)
	case *unprivqemu.Cluster:
		m, err = pc.NewMachineWithOptions(userdata, options)
	default:
		c.Fatal("unknown cluster type")
	}
	if err != nil {
		c.Fatalf("creating a machine failed: %v", err)
	}
	return m
}

func checkSysextZfs(c cluster.TestCluster) {
	m := createZfsMachine(c, zfsUserData)
	c.AssertCmdOutputContains(m, "findmnt /tank", "zfs")
	c.AssertCmdOutputContains(m, "zpool list -H -o name", "tank")
	c.AssertCmdOutputContains(m, "zfs list -H -o name", "tank")
	c.MustSSH(m, "sudo chown core /tank/ && echo world >/tank/hello")
	err := m.Reboot()
	if err != nil {
		c.Fatalf("could not reboot: %v", err)
	}
	c.AssertCmdOutputContains(m, "lsmod", "zfs")
	c.AssertCmdOutputContains(m, "grep --with-filename . /tank/hello", "hello:world")
}

func checkZfsDocker(c cluster.TestCluster) {
	m := createZfsMachine(c, zfsDockerUserData)
	c.AssertCmdOutputContains(m, "docker info -f '{{.Driver}}'", "zfs")
	c.AssertCmdOutputContains(m, "docker run --rm ghcr.io/flatcar/busybox mount", "zfs")
}

func checkZfsNfs(c cluster.TestCluster) {
	m := createZfsMachine(c, zfsNfsUserData)
	c.MustSSH(m, "sudo chown core /tank/nfs")
	c.MustSSH(m, "echo world >/tank/nfs/hello")
	m2, err := c.NewMachine(nfsClientUserData)
	if err != nil {
		c.Fatalf("Cluster.NewMachine: %s", err)
	}
	c.MustSSH(m2, "sudo mkdir /mnt/nfs")
	c.MustSSH(m2, fmt.Sprintf("sudo mount -t nfs -o nfsvers=4 %s:/tank/nfs /mnt/nfs", m.PrivateIP()))
	c.AssertCmdOutputContains(m2, "cat /mnt/nfs/hello", "world")
}
