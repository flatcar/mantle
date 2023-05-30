// Copyright (c) Microsoft.
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
	"fmt"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform/conf"
)

const (
	oemPlaceholder   string = "@OEM@"
	oldOEMMountpoint string = "/usr/share/oem"
	newOEMMountpoint string = "/oem"

	regularButaneConfigTemplate string = `---
variant: flatcar
version: 1.0.0
storage:
  filesystems:
     - device: "/dev/disk/by-label/OEM"
       format: "btrfs"
       path: @OEM@
  files:
    - path: @OEM@/grub.cfg
      mode: 0644
      overwrite: true
      contents:
        inline: |
          set linux_append="flatcar.autologin"
          # Needed if --qemu-skip-mangle is not set
          set linux_console="console=ttyS0,115200"
`
	indirectButaneConfigTemplate string = `---
variant: flatcar
version: 1.0.0
storage:
  files:
    - path: @OEM@/grub.cfg
      mode: 0644
      overwrite: true
      contents:
        inline: |
          set linux_append="flatcar.autologin"
          # Needed if --qemu-skip-mangle is not set
          set linux_console="console=ttyS0,115200"
`
)

func withOldOEMMountpoint(template string) string {
	return strings.Replace(template, oemPlaceholder, oldOEMMountpoint, -1)
}

func withNewOEMMountpoint(template string) string {
	return strings.Replace(template, oemPlaceholder, newOEMMountpoint, -1)
}

func init() {
	register.Register(&register.Test{
		Name:        "cl.ignition.oem.regular",
		Run:         reusePartitionOld,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test overwrites the grub.cfg which does not work on cloud environments after reboot
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3549},
		UserData:   conf.Butane(withOldOEMMountpoint(regularButaneConfigTemplate)),
	})
	register.Register(&register.Test{
		Name:        "cl.ignition.oem.regular.new",
		Run:         reusePartitionNew,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test overwrites the grub.cfg which does not work on cloud environments after reboot
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3603},
		UserData:   conf.Butane(withNewOEMMountpoint(regularButaneConfigTemplate)),
	})
	register.Register(&register.Test{
		// Check new behavior from https://github.com/flatcar/bootengine/pull/58
		// to not have to specify the OEM filesystem in Ignition
		Name:        "cl.ignition.oem.indirect",
		Run:         reusePartitionOld,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test overwrites the grub.cfg which does not work on cloud environments after reboot
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3550},
		UserData:   conf.Butane(withOldOEMMountpoint(indirectButaneConfigTemplate)),
	})
	register.Register(&register.Test{
		// Check new behavior from https://github.com/flatcar/bootengine/pull/58
		// to not have to specify the OEM filesystem in Ignition
		Name:        "cl.ignition.oem.indirect.new",
		Run:         reusePartitionNew,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test overwrites the grub.cfg which does not work on cloud environments after reboot
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3603},
		UserData:   conf.Butane(withNewOEMMountpoint(indirectButaneConfigTemplate)),
	})
	register.Register(&register.Test{
		Name:        "cl.ignition.oem.reuse",
		Run:         reusePartitionOld,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test overwrites the grub.cfg which does not work on cloud environments after reboot
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 2983},
		// Using CLC format here also covers the ign-converter case
		UserData: conf.ContainerLinuxConfig(`storage:
  filesystems:
     - name: oem
       mount:
         device: "/dev/disk/by-label/OEM"
         format: "ext4"
  files:
    - path: /grub.cfg
      filesystem: oem
      mode: 0644
      contents:
        inline: |
          set linux_append="flatcar.autologin"`),
	})
	register.Register(&register.Test{
		Name:        "cl.ignition.oem.wipe",
		Run:         wipeOEM,
		MinVersion:  semver.Version{Major: 2983},
		ClusterSize: 1,
		// `wiping` the OEM file system does not allow the instance to boot on platforms
		// different from QEMU.
		// More details: https://github.com/flatcar/Flatcar/issues/514.
		Platforms: []string{"qemu", "qemu-unpriv"},
		// Using CLC format here also covers the ign-converter case
		UserData: conf.ContainerLinuxConfig(`storage:
  filesystems:
     - name: oem
       mount:
         device: "/dev/disk/by-label/OEM"
         format: "ext4"
         wipe_filesystem: true
         label: "OEM"
  files:
    - path: /grub.cfg
      filesystem: oem
      mode: 0644
      contents:
        inline: |
          set linux_append="flatcar.autologin"`),
	})
}

func reusePartitionOld(c cluster.TestCluster) {
	reusePartition(c, oldOEMMountpoint)
}

func reusePartitionNew(c cluster.TestCluster) {
	reusePartition(c, newOEMMountpoint)
}

// reusePartition asserts that even if the config uses a different fs format, we keep using `btrfs`.
func reusePartition(c cluster.TestCluster, oemMountpoint string) {
	grub := c.MustSSH(c.Machines()[0], fmt.Sprintf(`grep -m 1 '^set linux_append="flatcar.autologin"$' %s/grub.cfg`, oemMountpoint))
	if string(grub) != `set linux_append="flatcar.autologin"` {
		c.Fatalf("did not find written grub entry: %s", string(grub))
	}

	out := c.MustSSH(c.Machines()[0], `lsblk --output FSTYPE,LABEL,MOUNTPOINT --json | jq -r '.blockdevices | .[] | select(.label=="OEM") | .fstype'`)

	if string(out) != "btrfs" {
		debug := c.MustSSH(c.Machines()[0], `lsblk --output FSTYPE,LABEL,MOUNTPOINT --json; echo ; lsblk`)
		c.Fatalf("should get btrfs, got: %s\ndebug info: %s", string(out), string(debug))
	}

	// Test for https://github.com/flatcar/Flatcar/issues/979
	// Not sure why exactly the second mount is able to reproduce this, I suppose
	// it is the coredump service DBus hang around initrd-setup-root that contributes
	// to exposing a different execution trace of the initrd service race.
	_ = c.MustSSH(c.Machines()[0], `sudo touch /boot/flatcar/first_boot`)
	err := c.Machines()[0].Reboot()
	if err != nil {
		c.Fatalf("Couldn't reboot machine: %v", err)
	}
	_ = c.MustSSH(c.Machines()[0], `true`)
}

// wipeOEM asserts that if the config uses a different fs format with a wipe of the fs we effectively wipe the fs.
func wipeOEM(c cluster.TestCluster) {
	out := c.MustSSH(c.Machines()[0], `lsblk --output FSTYPE,LABEL,MOUNTPOINT --json | jq -r '.blockdevices | .[] | select(.label=="OEM") | .fstype'`)

	if string(out) != "ext4" {
		c.Fatalf("should get ext4, got: %s", string(out))
	}
}
