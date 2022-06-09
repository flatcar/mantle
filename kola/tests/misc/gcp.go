// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package misc

import (
	"github.com/coreos/go-semver/semver"
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
)

func init() {
	register.Register(&register.Test{
		Name:        "cl.misc.gce.oem",
		ClusterSize: 1,
		Platforms:   []string{"gce"},
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 2801},
		Run:         gceVerifyOEMService,
	})
}

func gceVerifyOEMService(c cluster.TestCluster) {
	machine := c.Machines()[0]
	// verify that the oem-gce service is running
	c.MustSSH(machine, "systemctl is-active oem-gce.service")
	nrestarts := c.MustSSH(machine, "systemctl show oem-gce.service -P NRestarts")
	if string(nrestarts) != "0" {
		c.Fatalf("oem-gce service restarted too many times: %s", nrestarts)
	}
	// check that interface is configured and named correctly
	c.MustSSH(machine, "networkctl status eth0")
}
