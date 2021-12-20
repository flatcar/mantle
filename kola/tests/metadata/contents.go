// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package metadata

import (
	"strings"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
)

func init() {
	enableMetadataService := conf.Ignition(`{
	    "ignitionVersion": 1,
	    "systemd": {
		"units": [
		    {
			"name": "coreos-metadata.service",
			"enable": true
		    },
		    {
			"name": "metadata.target",
			"enable": true,
			"contents": "[Install]\nWantedBy=multi-user.target"
		    }
		]
	    }
	}`)

	register.Register(&register.Test{
		Name:        "cl.metadata.aws",
		Run:         verifyAWS,
		ClusterSize: 1,
		Platforms:   []string{"aws"},
		UserData:    enableMetadataService,
		Distros:     []string{"cl"},
	})

	register.Register(&register.Test{
		Name:        "cl.metadata.azure",
		Run:         verifyAzure,
		ClusterSize: 1,
		Platforms:   []string{"azure"},
		UserData:    enableMetadataService,
		Distros:     []string{"cl"},
	})

	register.Register(&register.Test{
		Name:        "cl.metadata.packet",
		Run:         verifyPacket,
		ClusterSize: 1,
		Platforms:   []string{"packet"},
		UserData:    enableMetadataService,
		Distros:     []string{"cl"},
	})
}

func verifyAWS(c cluster.TestCluster) {
	verify(c, "COREOS_EC2_IPV4_LOCAL", "COREOS_EC2_IPV4_PUBLIC", "COREOS_EC2_HOSTNAME")
}

func verifyAzure(c cluster.TestCluster) {
	verify(c, "COREOS_AZURE_IPV4_DYNAMIC")
	// kola tests do not spawn machines behind a load balancer on Azure
	// which is required for COREOS_AZURE_IPV4_VIRTUAL to be present
}

func verifyPacket(c cluster.TestCluster) {
	verify(c, "COREOS_PACKET_HOSTNAME", "COREOS_PACKET_PHONE_HOME_URL", "COREOS_PACKET_IPV4_PUBLIC_0", "COREOS_PACKET_IPV4_PRIVATE_0", "COREOS_PACKET_IPV6_PUBLIC_0")
}

func verify(c cluster.TestCluster, keys ...string) {
	m := c.Machines()[0]

	out := c.MustSSH(m, "cat /run/metadata/coreos")

	for _, key := range keys {
		if !strings.Contains(string(out), key) {
			c.Errorf("%q wasn't found in %q", key, string(out))
		}
	}
}
