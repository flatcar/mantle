// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package misc

import (
	"strings"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
)

func init() {
	register.Register(&register.Test{
		Run:         dnfInstall,
		ClusterSize: 1,
		Name:        "cl.toolbox.dnf-install",
		Distros:     []string{"cl"},
	})
}

// regression test for https://github.com/coreos/bugs/issues/1676
func dnfInstall(c cluster.TestCluster) {
	m := c.Machines()[0]

	output := c.MustSSH(m, `toolbox sh -c 'dnf install -y tcpdump; tcpdump --version >/dev/null && echo PASS' 2>/dev/null`)

	if !strings.Contains(string(output), "PASS") {
		c.Fatalf("Expected 'pass' in output; got %v", string(output))
	}
}
