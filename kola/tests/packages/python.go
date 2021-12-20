// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package packages

import (
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
)

// init runs when the package is imported and takes care of registering tests
func init() {
	register.Register(&register.Test{
		Run:         noPythonTest,
		ClusterSize: 1,
		Name:        `fcos.python`,
		Distros:     []string{"fcos"},
	})
}

// Test: Verify python is not installed
func noPythonTest(c cluster.TestCluster) {
	m := c.Machines()[0]

	out, err := c.SSH(m, `rpm -q python2`)
	if err == nil {
		c.Fatalf("%s should not be installed", out)
	}

	out, err = c.SSH(m, `rpm -q python3`)
	if err == nil {
		c.Fatalf("%s should not be installed", out)
	}
}
