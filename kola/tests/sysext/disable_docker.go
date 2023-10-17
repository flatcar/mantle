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
		Run:         dockerDisable,
		ClusterSize: 1,
		Platforms:   []string{"qemu", "qemu-unpriv"},
		Name:        "sysext.disable-docker",
		// Only releases after 3745 ship sysext
		MinVersion: semver.Version{Major: 3746},
		UserData: conf.Butane(`
variant: flatcar
version: 1.0.0
storage:
  links:
  - path: /etc/extensions/docker-flatcar.raw
    target: /dev/null
    hard: false
    overwrite: true
`),
		Distros: []string{"cl"},
	})
}

func dockerDisable(c cluster.TestCluster) {
	m := c.Machines()[0]
	output := c.MustSSH(m,
		`test -f /usr/bin/docker && echo "ERROR" || true`)
	if string(output) == "ERROR" {
		c.Errorf("/usr/bin/docker exists even when sysext is disabled!")
	}
}
