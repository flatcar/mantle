// Copyright 2023 The Flatcar Maintainers.
// SPDX-License-Identifier: Apache-2.0

package sysext

import (
	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform/conf"
)

func init() {
	register.Register(&register.Test{
		Run:         containerdDisable,
		ClusterSize: 1,
		Platforms:   []string{"qemu", "qemu-unpriv", "azure"},
		Name:        "sysext.disable-containerd",
		// Sysext containerd was introduced after release 3760.
		// NOTE that 3761 is a developer version which was never released.
		// However, the next largest Alpha major release shipped sysext.
		MinVersion: semver.Version{Major: 3761},
		// We also disable our vendor docker sysext since it depends on the containerd sysext.
		UserData: conf.Butane(`
variant: flatcar
version: 1.0.0
storage:
  links:
  - path: /etc/extensions/containerd-flatcar.raw
    target: /dev/null
    hard: false
    overwrite: true
  - path: /etc/extensions/docker-flatcar.raw
    target: /dev/null
    hard: false
    overwrite: true
`),
		Distros: []string{"cl"},
	})
}

func containerdDisable(c cluster.TestCluster) {
	m := c.Machines()[0]
	output := c.MustSSH(m,
		`test -f /usr/bin/containerd && echo "ERROR" || true`)
	if string(output) == "ERROR" {
		c.Errorf("/usr/bin/containerd exists even when sysext is disabled!")
	}
}
