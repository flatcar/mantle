// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package misc

import (
	"strings"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
)

func init() {
	register.Register(&register.Test{
		Run:         CloudInitBasic,
		ClusterSize: 1,
		Name:        "cl.cloudinit.basic",
		UserData: conf.CloudConfig(`#cloud-config
hostname: "core1"
write_files:
  - path: "/foo"
    content: bar`),
		Distros:          []string{"cl"},
		ExcludePlatforms: []string{"qemu-unpriv"},
	})
	register.Register(&register.Test{
		Run:         CloudInitScript,
		ClusterSize: 1,
		Name:        "cl.cloudinit.script",
		UserData: conf.Script(`#!/bin/bash
echo bar > /foo
mkdir -p ~core/.ssh
cat <<EOF >> ~core/.ssh/authorized_keys
@SSH_KEYS@
EOF
chown -R core.core ~core/.ssh
chmod 700 ~core/.ssh
chmod 600 ~core/.ssh/authorized_keys`),
		Distros:          []string{"cl"},
		ExcludePlatforms: []string{"qemu-unpriv"},
	})
}

func CloudInitBasic(c cluster.TestCluster) {
	m := c.Machines()[0]

	out := c.MustSSH(m, "cat /foo")
	if string(out) != "bar" {
		c.Fatalf("cloud-config produced unexpected value %q", out)
	}

	out = c.MustSSH(m, "hostnamectl")
	if !strings.Contains(string(out), "Static hostname: core1") {
		c.Fatalf("hostname wasn't set correctly:\n%s", out)
	}
}

func CloudInitScript(c cluster.TestCluster) {
	m := c.Machines()[0]

	out := c.MustSSH(m, "cat /foo")
	if string(out) != "bar" {
		c.Fatalf("userdata script produced unexpected value %q", out)
	}
}
