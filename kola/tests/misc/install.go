// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package misc

import (
	"bytes"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
)

func init() {
	register.Register(&register.Test{
		Run:         InstallCloudConfig,
		ClusterSize: 1,
		Name:        "cl.install.cloudinit",
		UserData: conf.Ignition(`{
  "ignition": { "version": "2.0.0" },
  "storage": {
    "files": [{
      "filesystem": "root",
      "path": "/var/lib/flatcar-install/user_data",
      "contents": { "source": "data:,%23cloud-config%0Ahostname:%20%22cloud-config-worked%22" },
      "mode": 420
    }]
  }
}`),
		Distros:          []string{"cl"},
		ExcludePlatforms: []string{"azure"},
	})
}

// Simulate coreos-install features

// Verify that the coreos-install cloud-config path is used
func InstallCloudConfig(c cluster.TestCluster) {
	m := c.Machines()[0]

	// Verify the host name was set from the cloud-config file
	if output, err := c.SSH(m, "hostname"); err != nil || !bytes.Equal(output, []byte("cloud-config-worked")) {
		c.Fatalf("hostname: %q: %v", output, err)
	}
}
