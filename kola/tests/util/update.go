// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"strings"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/platform"
)

func AssertBootedUsr(c cluster.TestCluster, m platform.Machine, usr string) {
	usrdev := GetUsrDeviceNode(c, m)
	target := c.MustSSH(m, "readlink -f /dev/disk/by-partlabel/"+usr)
	if usrdev != string(target) {
		c.Fatalf("Expected /usr to be %v (%s) but it is %v", usr, target, usrdev)
	}
}

func GetUsrDeviceNode(c cluster.TestCluster, m platform.Machine) string {
	// find /usr dev
	usrdev := c.MustSSH(m, "findmnt -no SOURCE /usr")

	// XXX: if the /usr dev is /dev/mapper/usr, we're on a verity enabled
	// image, so use dmsetup to find the real device.
	if strings.TrimSpace(string(usrdev)) == "/dev/mapper/usr" {
		usrdev = c.MustSSH(m, "echo -n /dev/$(sudo dmsetup info --noheadings -Co blkdevs_used usr)")
	}

	return string(usrdev)
}

func InvalidateUsrPartition(c cluster.TestCluster, m platform.Machine, partition string) {
	if out, stderr, err := m.SSH(fmt.Sprintf("sudo flatcar-setgoodroot && sudo wipefs /dev/disk/by-partlabel/%s", partition)); err != nil {
		c.Fatalf("invalidating %s failed: %s: %v: %s", partition, out, err, stderr)
	}
}
