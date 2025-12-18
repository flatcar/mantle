// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0
package ignition

import (
	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform/conf"
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
	})
}

func check(c cluster.TestCluster) {
	m := c.Machines()[0]

	c.AssertCmdOutputContains(m, "cat /proc/cmdline", " quiet") // assuming space for word separation
}
