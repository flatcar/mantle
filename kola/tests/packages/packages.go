// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package packages

import (
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
)

func init() {
	register.Register(&register.Test{
		Run:         packageTests,
		ClusterSize: 1,
		Name:        "packages",
		Distros:     []string{"cl"},
	})
}

func packageTests(c cluster.TestCluster) {
	c.Run("sys-cluster/ipvsadm", ipvsadm)
}
