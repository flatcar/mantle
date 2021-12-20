// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package ignition

import (
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
)

func init() {
	// verify that SSH key injection works correctly through Ignition,
	// without injecting via platform metadata
	register.Register(&register.Test{
		Name:             "cl.ignition.v1.ssh.key",
		Run:              empty,
		ClusterSize:      1,
		ExcludePlatforms: []string{"qemu"}, // redundant on qemu
		Flags:            []register.Flag{register.NoSSHKeyInMetadata},
		UserData:         conf.Ignition(`{"ignitionVersion": 1}`),
		Distros:          []string{"cl"},
	})
	register.Register(&register.Test{
		Name:             "coreos.ignition.ssh.key",
		Run:              empty,
		ClusterSize:      1,
		ExcludePlatforms: []string{"qemu"}, // redundant on qemu
		Flags:            []register.Flag{register.NoSSHKeyInMetadata},
		UserData:         conf.Ignition(`{"ignition":{"version":"2.0.0"}}`),
		UserDataV3:       conf.Ignition(`{"ignition":{"version":"3.0.0"}}`),
		Distros:          []string{"cl", "fcos", "rhcos"},
	})
}
