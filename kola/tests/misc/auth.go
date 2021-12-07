// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package misc

import (
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
)

func init() {
	register.Register(&register.Test{
		Run:         AuthVerify,
		ClusterSize: 1,
		Name:        "coreos.auth.verify",
		Distros:     []string{"cl", "fcos", "rhcos"},
	})
}

// Basic authentication tests.

// AuthVerify asserts that invalid passwords do not grant access to the system
func AuthVerify(c cluster.TestCluster) {
	m := c.Machines()[0]

	client, err := m.PasswordSSHClient("core", "asdf")
	if err == nil {
		client.Close()
		c.Fatalf("Successfully authenticated despite invalid password auth")
	}
}
