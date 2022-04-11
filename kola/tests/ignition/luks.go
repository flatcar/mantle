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
		Name:        "cl.ignition.luks",
		Run:         luksTest,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 3185},
		UserData: conf.Ignition(`{
		  "ignition": {"version": "3.2.0"},
		  "storage": {
		    "luks": [{
		      "name": "data",
		      "device": "/dev/disk/by-partlabel/USR-B"
		    }],
		    "filesystems": [{
		      "path": "/var/lib/data",
		      "device": "/dev/disk/by-id/dm-name-data",
		      "format": "ext4",
		      "label": "DATA"
		    }]
		  },
		  "systemd": {
		    "units": [{
		      "name": "var-lib-data.mount",
		      "enabled": true,
		      "contents": "[Mount]\nWhat=/dev/disk/by-label/DATA\nWhere=/var/lib/data\nType=ext4\n\n[Install]\nWantedBy=local-fs.target"
		    }]
		  }
	  	}`),
	})
}

func luksTest(c cluster.TestCluster) {
	m := c.Machines()[0]

	c.MustSSH(m, "sudo cryptsetup isLuks /dev/disk/by-partlabel/USR-B")
	c.MustSSH(m, "systemctl is-active var-lib-data.mount")
}
