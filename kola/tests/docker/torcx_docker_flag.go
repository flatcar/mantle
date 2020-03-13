// Copyright 2017 CoreOS, Inc.
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

package docker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/coreos/mantle/kola/cluster"
	"github.com/coreos/mantle/kola/register"
	"github.com/coreos/mantle/platform"
	"github.com/coreos/mantle/platform/conf"
)

func init() {
	register.Register(&register.Test{
		Run:         dockerTorcxFlagFile,
		ClusterSize: 1,
		Name:        "docker.torcx-flag-file",
		UserData: conf.ContainerLinuxConfig(`
storage:
  files:
    - filesystem: root
      path: /etc/flatcar/docker-1.12
      contents:
        inline: yes
      mode: 0644
`),
		Distros: []string{"cl"},
	})
	register.Register(&register.Test{
		Run:         dockerTorcxFlagFileCloudConfig,
		ClusterSize: 1,
		Name:        "docker.torcx-flag-file.cloud-config",
		UserData: conf.CloudConfig(`
#cloud-config
write_files:
  - path: "/etc/flatcar/docker-1.12"
    content: yes
`),
		Distros:          []string{"cl"},
		ExcludePlatforms: []string{"qemu-unpriv"},
	})
}

func dockerTorcxFlagFile(c cluster.TestCluster) {
	m := c.Machines()[0]

	// Skip the test in case of Edge, e.g. "xxxx.99.z"
	ver := strings.Split(string(c.MustSSH(m, "grep ^VERSION_ID= /etc/os-release")), "=")[1]
	semver, err := parseCLVersion(ver)
	if err != nil {
		c.Fatalf("cannot parse Flatcar version: %v", err)
	}
	if semver.Minor == int64(99) {
		c.Skipf("skipping tests for Edge %s", semver.String())
	}

	// flag=yes
	checkTorcxDockerVersions(c, m, `^1\.12$`, `^1\.12\.`)

	// flag=no
	c.MustSSH(m, "echo no | sudo tee /etc/flatcar/docker-1.12")
	if err := m.Reboot(); err != nil {
		c.Fatalf("could not reboot: %v", err)
	}
	c.MustSSH(m, `sudo rm -rf /var/lib/docker`)
	checkTorcxDockerVersions(c, m, `^1[7-9]\.`, `^1[7-9]\.`)
}

func dockerTorcxFlagFileCloudConfig(c cluster.TestCluster) {
	m := c.Machines()[0]

	// Skip the test in case of Edge, e.g. "xxxx.99.z"
	ver := strings.Split(string(c.MustSSH(m, "grep ^VERSION_ID= /etc/os-release")), "=")[1]
	semver, err := parseCLVersion(ver)
	if err != nil {
		c.Fatalf("cannot parse Flatcar version: %v", err)
	}
	if semver.Minor == int64(99) {
		c.Skipf("skipping tests for Edge %s", semver.String())
	}

	// cloudinit runs after torcx
	if err := m.Reboot(); err != nil {
		c.Fatalf("couldn't reboot: %v", err)
	}

	// flag=yes
	checkTorcxDockerVersions(c, m, `^1\.12$`, `^1\.12\.`)
}

func checkTorcxDockerVersions(c cluster.TestCluster, m platform.Machine, expectedRefRE, expectedVerRE string) {
	ref := getTorcxDockerReference(c, m)
	if !regexp.MustCompile(expectedRefRE).MatchString(ref) {
		c.Errorf("reference %s did not match %q", ref, expectedRefRE)
	}

	ver := getDockerServerVersion(c, m)
	if !regexp.MustCompile(expectedVerRE).MatchString(ver) {
		c.Errorf("version %s did not match %q", ver, expectedVerRE)
	}
}

func parseCLVersion(input string) (*semver.Version, error) {
	version, err := semver.NewVersion(input)
	if err != nil {
		return nil, fmt.Errorf("parsing os-release semver: %v", err)
	}

	return version, nil
}
