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


package sysext

import (
	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform/conf"
)

func init() {
	register.Register(&register.Test{
		Run:         containerdDisable,
		ClusterSize: 1,
		Platforms: []string{"qemu", "qemu-unpriv"},
		Name:      "sysext.disable-containerd",
        // Only releases after 3745 ship sysext
		MinVersion: semver.Version{Major: 3746},
		UserData: conf.Butane(`
variant: flatcar
version: 1.0.0
storage:
  links:
  - path: /etc/extensions/containerd-flatcar.raw
    target: /dev/null
    hard: false
    overwrite: true
`),
		Distros: []string{"cl"},
	})
}

func containerdDisable(c cluster.TestCluster) {
	m := c.Machines()[0]
	output := c.MustSSH(m,
            `test ! -f /usr/bin/containerd || echo "ERROR"`)
	if string(output) == "ERROR" {
		c.Errorf("/usr/bin/containerd exists even when sysext is disabled!")
	}
}
