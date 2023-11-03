// Copyright 2017 CoreOS, Inc.
// Copyright 2023 The Flatcar Maintainers.
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

// This test originated as torcx test and is kept to shield against
// similar activation issues with sysext.

// The test ensures that, given respective user configuration, Docker is started
// at boot (instead just socket-activated). This is necessary for auto-restarting
// containers at boot that have been running at shutodwn. That's done by docker,
// so docker must be running.

package docker

import (
	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform/conf"
)

func init() {
	// Regression test for https://github.com/coreos/bugs/issues/2079
	register.Register(&register.Test{
		Run:         dockerEnable,
		ClusterSize: 1,
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv"},
		Name:      "docker.enable-service.torcx",
		// Torcx was retired after release 3760.
		EndVersion: semver.Version{Major: 3760},
		UserData: conf.ContainerLinuxConfig(`
systemd:
  units:
  - name: docker.service
    enabled: true
`),
		Distros: []string{"cl"},
	})

	register.Register(&register.Test{
		Run:         dockerEnable,
		ClusterSize: 1,
		// This test is normally not related to the cloud environment
		Platforms:  []string{"qemu", "qemu-unpriv"},
		Name:       "docker.enable-service.sysext",
		MinVersion: semver.Version{Major: 3746},
		UserData: conf.Butane(`
variant: flatcar
version: 1.0.0
systemd:
  units:
  - name: docker.service
    enabled: true
storage:
  links:
  - path: /etc/systemd/system/multi-user.target.wants/docker.service
    target: /usr/lib/systemd/system/docker.service
    hard: false
    overwrite: true
`),
		// TODO FIXME: Convert this to a multi-user.target.upholds/docker.service symlink
		// after we switch to systemd-254.
		Distros: []string{"cl"},
	})
}

func dockerEnable(c cluster.TestCluster) {
	m := c.Machines()[0]
	c.AssertCmdOutputContains(m, "systemctl is-enabled docker", "enabled")
}
