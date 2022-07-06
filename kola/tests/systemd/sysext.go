// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package systemd

import (
	"github.com/coreos/go-semver/semver"
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
)

func init() {
	register.Register(&register.Test{
		Name:        "systemd.sysext.simple",
		Run:         checkSysextSimple,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3185},
		UserData: conf.ContainerLinuxConfig(`storage:
  files:
    - path: /etc/extensions/test/usr/lib/extension-release.d/extension-release.test
      contents:
        inline: |
          ID=flatcar
          SYSEXT_LEVEL=1.0
    - path: /etc/extensions/test/usr/hello-sysext
      contents:
        inline: |
          sysext works`),
	})
}

func checkHelper(c cluster.TestCluster) {
	_ = c.MustSSH(c.Machines()[0], `grep -m 1 '^sysext works$' /usr/hello-sysext`)
	// "mountpoint /usr/share/oem" is too lose for our purposes, because we want to check if the mount point is accessible and "df" only shows these by default
	target := c.MustSSH(c.Machines()[0], `if [ -e /dev/disk/by-label/OEM ]; then df --output=target | grep /usr/share/oem; fi`)
	// check against multiple entries which is not wanted
	if string(target) != "/usr/share/oem" {
		c.Fatalf("should get /usr/share/oem, got %q", string(target))
	}
}

func checkSysextSimple(c cluster.TestCluster) {
	// First check directly after boot
	checkHelper(c)
	_ = c.MustSSH(c.Machines()[0], `sudo systemctl restart systemd-sysext`)
	// Second check after reloading the extensions (e.g., to add/remove/update them)
	checkHelper(c)
}
