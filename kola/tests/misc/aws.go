// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package misc

import (
	"fmt"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
)

func init() {
	register.Register(&register.Test{
		Name:        "coreos.misc.aws.diskfriendlyname",
		Platforms:   []string{"aws"},
		Run:         awsVerifyDiskFriendlyName,
		ClusterSize: 1,
		// Previously broken on NVMe devices, see
		// https://github.com/coreos/bugs/issues/2399
		MinVersion: semver.Version{Major: 1828},
		Distros:    []string{"cl", "rhcos"},
	})
}

// Check invariants on AWS instances.

func awsVerifyDiskFriendlyName(c cluster.TestCluster) {
	friendlyName := "/dev/xvda"
	c.MustSSH(c.Machines()[0], fmt.Sprintf("stat %s", friendlyName))
}
