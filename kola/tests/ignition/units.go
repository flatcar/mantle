// Copyright The Mantle Authors
// Copyright 2020 Red Hat
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
		Name:        "cl.ignition.instantiated.enable-unit",
		Run:         enableSystemdInstantiatedService,
		ClusterSize: 1,
		UserData: conf.Ignition(`{
    "ignition": {"version": "3.0.0"},
    "systemd": {
        "units": [{
			"name": "echo@.service",
			"contents": "[Unit]\nDescription=f\n[Service]\nType=oneshot\nExecStart=/bin/echo %i\nRemainAfterExit=yes\n[Install]\nWantedBy=multi-user.target\n"
		  },
		  {
		  "name": "echo@.timer",
		  "contents": "[Unit]\nDescription=echo timer template\n[Timer]\nOnUnitInactiveSec=10s\n[Install]\nWantedBy=timers.target"
		},
		{
		  "enabled": true,
		  "name": "echo@bar.service"
		},
		{
		  "enabled": true,
		  "name": "echo@foo.service"
		},
		{
			"enabled": true,
			"name": "echo@foo.timer"
		}]
    }
}`),
		Distros:    []string{"cl"},
		MinVersion: semver.Version{Major: 3185},
	})
}

func enableSystemdInstantiatedService(c cluster.TestCluster) {
	m := c.Machines()[0]
	// MustSSH function will throw an error if the exit code
	// of the command is anything other than 0.
	_ = c.MustSSH(m, "systemctl -q is-active echo@foo.service")
	_ = c.MustSSH(m, "systemctl -q is-active echo@bar.service")
	_ = c.MustSSH(m, "systemctl -q is-enabled echo@foo.service")
	_ = c.MustSSH(m, "systemctl -q is-enabled echo@bar.service")
	_ = c.MustSSH(m, "systemctl -q is-active echo@foo.timer")
	_ = c.MustSSH(m, "systemctl -q is-enabled echo@foo.timer")
}
