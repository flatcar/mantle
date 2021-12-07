// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package ignition

import (
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
)

// These tests require the kola key to be passed to the instance via cloud
// provider metadata since it will not be injected into the config. Platforms
// where the cloud provider metadata system is not available have been excluded.
func init() {
	// Tests for https://github.com/coreos/bugs/issues/1184
	register.Register(&register.Test{
		Name:             "cl.ignition.misc.empty",
		Run:              empty,
		ClusterSize:      1,
		ExcludePlatforms: []string{"qemu", "esx"},
		Distros:          []string{"cl"},
		UserData:         conf.Empty(),
	})
	// Tests for https://github.com/coreos/bugs/issues/1981
	register.Register(&register.Test{
		Name:             "cl.ignition.v1.noop",
		Run:              empty,
		ClusterSize:      1,
		ExcludePlatforms: []string{"qemu", "esx", "openstack"},
		Distros:          []string{"cl"},
		Flags:            []register.Flag{register.NoSSHKeyInUserData},
		UserData:         conf.Ignition(`{"ignitionVersion": 1}`),
	})
	register.Register(&register.Test{
		Name:             "cl.ignition.v2.noop",
		Run:              empty,
		ClusterSize:      1,
		ExcludePlatforms: []string{"qemu", "esx", "openstack"},
		Distros:          []string{"cl"},
		Flags:            []register.Flag{register.NoSSHKeyInUserData},
		UserData:         conf.Ignition(`{"ignition":{"version":"2.0.0"}}`),
	})
}

func empty(_ cluster.TestCluster) {
}
