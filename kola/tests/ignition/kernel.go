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
		Name:        "cl.ignition.kargs",
		Run:         check,
		ClusterSize: 1,
		UserData: conf.Butane(`---
variant: flatcar
version: 1.0.0
kernel_arguments:
  should_exist:
    - quiet`),
		MinVersion: semver.Version{Major: 3185},
		// The additional reboot causes a large waiting time
		// and it's enough to test this on QEMU and other clouds
		// to check that the grub.cfg rewriting causes no problems
		ExcludePlatforms: []string{"equinixmetal"},
	})
}

func check(c cluster.TestCluster) {
	m := c.Machines()[0]

	c.AssertCmdOutputContains(m, "cat /proc/cmdline", " quiet") // assuming space for word separation
}
