// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"regexp"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/conf"
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
		Distros:         []string{"cl"},
		ExcludeChannels: []string{"alpha", "beta", "edge", "stable"},
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
		ExcludeChannels:  []string{"alpha", "beta", "edge", "stable"},
	})
}

func dockerTorcxFlagFile(c cluster.TestCluster) {
	m := c.Machines()[0]

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
