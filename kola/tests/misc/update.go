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

package misc

import (
	"fmt"
	"regexp"
	"time"

	"golang.org/x/net/context"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/kola/tests/util"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
)

var (
	// prevents a race where update-engine sets the boot partition back to
	// USR-A after the test sets it to USR-B
	disableUpdateEngine = conf.ContainerLinuxConfig(`systemd:
  units:
    - name: update-engine.service
      mask: true
    - name: locksmithd.service
      mask: true`)
)

func init() {
	register.Register(&register.Test{
		Run:         RebootIntoUSRB,
		ClusterSize: 1,
		Name:        "cl.update.reboot",
		UserData:    disableUpdateEngine,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv"},
	})
	register.Register(&register.Test{
		Run:         RecoverBadVerity,
		ClusterSize: 1,
		Name:        "cl.update.badverity",
		Flags:       []register.Flag{register.NoEmergencyShellCheck, register.NoKernelPanicCheck},
		UserData:    disableUpdateEngine,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv"},
	})
	register.Register(&register.Test{
		Run:         RecoverBadUsr,
		ClusterSize: 1,
		Name:        "coreos.update.badusr",
		Flags:       []register.Flag{register.NoEmergencyShellCheck},
		UserData:    disableUpdateEngine,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv"},
		// From this version, signed load scripts are used, we lack the GPG key
		// to sign a new one, and the kernel is supposed to be loaded from /usr.
		EndVersion: semver.Version{Major: 4344},
	})
}

// Simulate update scenarios

// Check that we can reprioritize and boot into USR-B. This largely
// validates the other tests in this file.
func RebootIntoUSRB(c cluster.TestCluster) {
	m := c.Machines()[0]

	util.AssertBootedUsr(c, m, "USR-A")

	// copy USR-A to USR-B
	c.MustSSH(m, "sudo cp /dev/disk/by-partlabel/USR-A /dev/disk/by-partlabel/USR-B")

	// copy kernel or load + verity files
	if _, err := c.SSH(m, "sudo cp /boot/flatcar/vmlinuz-a /boot/flatcar/vmlinuz-b"); err != nil {
		c.MustSSH(m, "sudo cp /boot/flatcar/load-a /boot/flatcar/load-b")
		c.MustSSH(m, "sudo cp /boot/flatcar/load-a.sig /boot/flatcar/load-b.sig")
		c.MustSSH(m, "sudo cp /boot/flatcar/verity-a.sig /boot/flatcar/verity-b.sig")
		c.MustSSH(m, "sudo cp /boot/flatcar/verity-a.sig.sig /boot/flatcar/verity-b.sig.sig")
	}

	prioritizeUsr(c, m, "USR-B")
	if err := m.Reboot(); err != nil {
		c.Fatalf("couldn't reboot: %v", err)
	}
	util.AssertBootedUsr(c, m, "USR-B")
}

// Verify that we reboot into the old image after invalidating the verity root
// hash. Annoyingly for newer versions, this will cause GRUB's load script GPG
// verification to fail rather than the kernel's root hash verification, but the
// end result should be the same. The security is hard to break. :)
func RecoverBadVerity(c cluster.TestCluster) {
	m := c.Machines()[0]

	skipUnlessVerity(c, m)

	util.AssertBootedUsr(c, m, "USR-A")

	// copy USR-A to USR-B
	c.MustSSH(m, "sudo cp /dev/disk/by-partlabel/USR-A /dev/disk/by-partlabel/USR-B")

	// copy kernel or load + verity files and then invalidate the verity hash
	if _, err := c.SSH(m, "sudo cp /boot/flatcar/vmlinuz-a /boot/flatcar/vmlinuz-b"); err == nil {
		c.MustSSH(m, fmt.Sprintf("sudo dd of=/boot/flatcar/vmlinuz-b bs=1 seek=%d count=64 conv=notrunc status=none <<<0000000000000000000000000000000000000000000000000000000000000000", getKernelVerityHashOffset(c)))
	} else {
		c.MustSSH(m, "sudo cp /boot/flatcar/load-a.sig /boot/flatcar/load-b.sig")
		c.MustSSH(m, "sudo cp /boot/flatcar/verity-a.sig /boot/flatcar/verity-b.sig")
		c.MustSSH(m, "sudo cp /boot/flatcar/verity-a.sig.sig /boot/flatcar/verity-b.sig.sig")
		c.MustSSH(m, "sed -r 's/usrhash=[^ ]+/usrhash=0000000000000000000000000000000000000000000000000000000000000000/' /boot/flatcar/load-a | sudo tee /boot/flatcar/load-b")
	}

	prioritizeUsr(c, m, "USR-B")
	// rebootWithEmergencyShellTimeout also covers the kernel panic timeout of 1 minute before reboot
	rebootWithEmergencyShellTimeout(c, m)
	util.AssertBootedUsr(c, m, "USR-A")
}

// Verify that we reboot into the old image when the new image is an
// unreasonable filesystem (an empty one) that passes verity.
func RecoverBadUsr(c cluster.TestCluster) {
	m := c.Machines()[0]

	util.AssertBootedUsr(c, m, "USR-A")

	// create filesystem for USR-B
	c.MustSSH(m, "sudo mkfs.ext4 -q -b 4096 /dev/disk/by-partlabel/USR-B 25600")

	// create verity metadata for USR-B
	output := c.MustSSH(m, "sudo veritysetup format --hash=sha256 "+
		"--data-block-size 4096 --hash-block-size 4096 --data-blocks 25600 --hash-offset 104857600 "+
		"/dev/disk/by-partlabel/USR-B /dev/disk/by-partlabel/USR-B")

	// extract root hash for USR-B
	match := regexp.MustCompile("\nRoot hash:\\s+([0-9a-f]+)").FindSubmatch(output)
	if match == nil {
		c.Fatalf("Couldn't obtain new root hash; output %s", output)
	}
	verityHash := match[1]

	// copy kernel
	c.MustSSH(m, "sudo cp /boot/flatcar/vmlinuz-a /boot/flatcar/vmlinuz-b")

	// update verity hash on B kernel
	c.MustSSH(m, fmt.Sprintf("sudo dd of=/boot/flatcar/vmlinuz-b bs=1 seek=%d count=64 conv=notrunc status=none <<<%s", getKernelVerityHashOffset(c), verityHash))

	prioritizeUsr(c, m, "USR-B")
	rebootWithEmergencyShellTimeout(c, m)
	util.AssertBootedUsr(c, m, "USR-A")
}

func prioritizeUsr(c cluster.TestCluster, m platform.Machine, usr string) {
	c.MustSSH(m, "sudo cgpt repair /dev/disk/by-partlabel/"+usr)
	c.MustSSH(m, "sudo cgpt add -S0 -T1 /dev/disk/by-partlabel/"+usr)
	c.MustSSH(m, "sudo cgpt prioritize /dev/disk/by-partlabel/"+usr)
}

// reboot, waiting extra-long for the 5-minute emergency shell timeout
func rebootWithEmergencyShellTimeout(c cluster.TestCluster, m platform.Machine) {
	// reboot; wait extra 5 minutes; check machine
	// this defeats some of the machinery in m.Reboot()
	if err := platform.StartReboot(m); err != nil {
		c.Fatal(err)
	}
	time.Sleep(5 * time.Minute)
	if err := platform.CheckMachine(context.TODO(), m); err != nil {
		c.Fatal(err)
	}
}
