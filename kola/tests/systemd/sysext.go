// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package systemd

import (
	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform/conf"
)

func init() {
	register.Register(&register.Test{
		Name:        "systemd.sysext.simple",
		Run:         checkSysextSimple,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3185},
		UserData: conf.ContainerLinuxConfig(`storage:
  files:
    - path: /etc/extensions/test/usr/lib/extension-release.d/extension-release.test
      contents:
        inline: |
          ID=flatcar
          SYSEXT_LEVEL=1.0
    - path: /etc/extensions/test/usr/hello-sysext
      contents:
        inline: |
          sysext works`),
	})
	register.Register(&register.Test{
		Name:        "systemd.sysext.custom-docker",
		Run:         checkSysextCustomDocker,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3185},
		UserData: conf.ContainerLinuxConfig(`storage:
  files:
    - path: /etc/systemd/system-generators/torcx-generator
  directories:
    - path: /etc/extensions/docker-flatcar
    - path: /etc/extensions/containerd-flatcar`),
	})

}

func checkHelper(c cluster.TestCluster) {
	_ = c.MustSSH(c.Machines()[0], `grep -m 1 '^sysext works$' /usr/hello-sysext`)
	// "mountpoint /usr/share/oem" is too lose for our purposes, because we want to check if the mount point is accessible and "df" only shows these by default
	target := c.MustSSH(c.Machines()[0], `if [ -e /dev/disk/by-label/OEM ]; then df --output=target | grep /usr/share/oem; fi`)
	// check against multiple entries which is not wanted
	if string(target) != "/usr/share/oem" {
		c.Fatalf("should get /usr/share/oem, got %q", string(target))
	}
}

func checkSysextSimple(c cluster.TestCluster) {
	// First check directly after boot
	checkHelper(c)
	_ = c.MustSSH(c.Machines()[0], `sudo systemctl restart systemd-sysext`)
	// Second check after reloading the extensions (e.g., to add/remove/update them)
	checkHelper(c)
}

func checkSysextCustomDocker(c cluster.TestCluster) {
	cmdNotWorking := `if docker run --rm ghcr.io/kinvolk/busybox true; then exit 1; fi`
	cmdWorking := `docker run --rm ghcr.io/kinvolk/busybox echo Hello World`
	// First assert that Docker doesn't work because Torcx is disabled
	_ = c.MustSSH(c.Machines()[0], cmdNotWorking)
	// We build a custom sysext image locally because we don't host them somewhere yet
	_ = c.MustSSH(c.Machines()[0], `git clone https://github.com/flatcar/sysext-bakery.git && git -C sysext-bakery checkout 2d6ab6bdba96476138c7db841e9a4c4ed38bd3fc`)
	// Flatcar has no mksquashfs and btrfs is missing a bugfix but at least ext4 works
	// The first test is for a fixed Docker version, which with the time will get old and older but is still expected to work because users may also "freeze" their Docker version this way
	_ = c.MustSSH(c.Machines()[0], `ONLY_DOCKER=1 FORMAT=ext4 sysext-bakery/create_docker_sysext.sh 20.10.21 docker && ONLY_CONTAINERD=1 FORMAT=ext4 sysext-bakery/create_docker_sysext.sh 20.10.21 containerd && sudo mv docker.raw containerd.raw /etc/extensions/`)
	_ = c.MustSSH(c.Machines()[0], `sudo systemctl restart systemd-sysext`)
	// We should now be able to use Docker
	_ = c.MustSSH(c.Machines()[0], cmdWorking)
	// The next test is with a recent Docker version, here the one from the Flatcar image to couple it to something that doesn't change under our feet
	version := string(c.MustSSH(c.Machines()[0], `bzcat /usr/share/licenses/licenses.json.bz2 | grep -m 1 -o 'app-emulation/docker[^:]*' | cut -d - -f 3`))
	_ = c.MustSSH(c.Machines()[0], `ONLY_DOCKER=1 FORMAT=ext4 sysext-bakery/create_docker_sysext.sh `+version+` docker && ONLY_CONTAINERD=1 FORMAT=ext4 sysext-bakery/create_docker_sysext.sh `+version+` containerd && sudo mv docker.raw containerd.raw /etc/extensions/`)
	_ = c.MustSSH(c.Machines()[0], `sudo systemctl restart systemd-sysext && sudo systemctl restart docker containerd`)
	// We should now still be able to use Docker
	_ = c.MustSSH(c.Machines()[0], cmdWorking)
}
