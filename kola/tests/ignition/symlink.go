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
		Name:        "cl.ignition.symlink",
		Run:         writeAbsoluteSymlink,
		ClusterSize: 1,
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv"},
		UserData: conf.Ignition(`{
		  "ignition": {
		      "version": "3.0.0"
		  },
		  "storage": {
		      "links": [
		          {
		              "group": {
		                  "name": "core"
		              },
		              "overwrite": true,
		              "path": "/etc/localtime",
		              "user": {
		                  "name": "core"
		              },
		              "hard": false,
		              "target": "/usr/share/zoneinfo/Europe/Zurich"
		          }
		      ]
		  }
	      }`),
		MinVersion: semver.Version{Major: 3185},
	})
}

func writeAbsoluteSymlink(c cluster.TestCluster) {
	m := c.Machines()[0]

	c.AssertCmdOutputContains(m, "readlink /etc/localtime", "/usr/share/zoneinfo/Europe/Zurich")
}
