// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ignition

import (
	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform/conf"
)

// These tests require the kola key to be passed to the instance via cloud
// provider metadata since it will not be injected into the config. Platforms
// where the cloud provider metadata system is not available have been excluded.
func init() {
	// Tests for https://github.com/coreos/bugs/issues/1184
	register.Register(&register.Test{
		Name:        "cl.ignition.misc.empty",
		Run:         empty,
		ClusterSize: 1,
		// brightbox does not support yet adding SSH keys to the metadata service.
		// akamai does not provide SSH keys as metadata (https://github.com/coreos/afterburn/issues/1111)
		ExcludePlatforms: []string{"qemu", "esx", "brightbox", "akamai"},
		Distros:          []string{"cl"},
		// The userdata injection of disabling the update server won't work
		// for an empty config, we still take care of doing later it via SSH
		Flags:    []register.Flag{register.NoDisableUpdates, register.NoSSHKeyInUserData},
		UserData: conf.Empty(),
		// Should run on all cloud environments
	})
	// Tests for https://github.com/coreos/bugs/issues/1981
	register.Register(&register.Test{
		Name:             "cl.ignition.v1.noop",
		Run:              empty,
		ClusterSize:      1,
		ExcludePlatforms: []string{"qemu", "esx", "openstack", "brightbox", "akamai"},
		Distros:          []string{"cl"},
		Flags:            []register.Flag{register.NoSSHKeyInUserData},
		UserData:         conf.Ignition(`{"ignitionVersion": 1}`),
		// Should run on all cloud environments
	})
	register.Register(&register.Test{
		Name:             "cl.ignition.v2.noop",
		Run:              empty,
		ClusterSize:      1,
		ExcludePlatforms: []string{"qemu", "esx", "brightbox", "akamai"},
		Distros:          []string{"cl"},
		Flags:            []register.Flag{register.NoSSHKeyInUserData},
		UserData:         conf.Ignition(`{"ignition":{"version":"2.0.0"}}`),
		MinVersion:       semver.Version{Major: 3227},
		// Should run on all cloud environments
	})
}

func empty(c cluster.TestCluster) {
	m := c.Machines()[0]
	_ = c.MustSSH(m, "echo SERVER=disabled | sudo tee /etc/flatcar/update.conf")
}
