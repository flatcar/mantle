// Copyright 2018 CoreOS, Inc.
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

package update

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-omaha/omaha"
	"github.com/coreos/go-semver/semver"

	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	tutil "github.com/flatcar/mantle/kola/tests/util"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/platform/local"
	"github.com/flatcar/mantle/util"
)

func init() {
	register.Register(&register.Test{
		Name:        "cl.update.payload",
		Run:         payload,
		ClusterSize: 1,
		NativeFuncs: map[string]func() error{
			"Omaha": Serve,
		},
		Distros: []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv"},
		SkipFunc: func(version semver.Version, channel, arch, platform string) bool {
			// This test can only run if the update payload to test is given.
			// The image passed must also be an old release to ensure that we
			// don't have incomaptible changes
			// (see scripts/ci-automation/vendor-testing/qemu_update.sh)
			return kola.UpdatePayloadFile == ""
		},
	})
	register.Register(&register.Test{
		Name:        "cl.sysext.boot",
		Run:         sysextBootLogic,
		ClusterSize: 0,
		Distros:     []string{"cl"},
		// This test is uses its own OEM files and shouldn't run on other platforms
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3481},
	})
}

func Serve() error {
	omahaserver, err := omaha.NewTrivialServer(":34567")
	if err != nil {
		return fmt.Errorf("creating trivial omaha server: %v\n", err)
	}

	omahawrapper := local.OmahaWrapper{TrivialServer: omahaserver}

	if err = omahawrapper.AddPackage("/updates/update.gz", "update.gz"); err != nil {
		return fmt.Errorf("bad payload: %v", err)
	}

	return omahawrapper.Serve()
}

func payload(c cluster.TestCluster) {
	addr := configureOmahaServer(c, c.Machines()[0])

	// create the actual test machine, the machine
	// that is created by the test registration is
	// used to host the omaha server
	m, err := c.NewMachine(nil)
	if err != nil {
		c.Fatalf("creating test machine: %v", err)
	}

	// Machines are intentionally configured post-boot
	// via SSH to allow for testing versions which predate
	// Ignition
	configureMachineForUpdate(c, m, addr)

	tutil.AssertBootedUsr(c, m, "USR-A")

	updateMachine(c, m)

	tutil.AssertBootedUsr(c, m, "USR-B")

	tutil.InvalidateUsrPartition(c, m, "USR-A")

	/*
		in the case we previously downloaded and installed an **official** release, the
		/usr/share/update_engine/update-payload-key.pub.pem will be changed to the official one.
		In consequence, update-engine will fail to verify the update payload since this
		one appears to be signed, in a test context, with a dev-key (generated from
		the SDK.)
		We configure again to inject the dev-pub-key to correctly verify the downloaded payload
	*/
	configureMachineForUpdate(c, m, addr)

	updateMachine(c, m)

	tutil.AssertBootedUsr(c, m, "USR-A")
}

func configureOmahaServer(c cluster.TestCluster, srv platform.Machine) string {
	in, err := os.Open(kola.UpdatePayloadFile)
	if err != nil {
		c.Fatalf("opening update payload: %v", err)
	}
	defer in.Close()
	if err := platform.InstallFile(in, srv, "/updates/update.gz"); err != nil {
		c.Fatalf("copying update payload to omaha server: %v", err)
	}

	c.MustSSH(srv, fmt.Sprintf("sudo systemd-run --quiet ./kolet run %s Omaha", c.H.Name()))

	err = util.WaitUntilReady(60*time.Second, 5*time.Second, func() (bool, error) {
		_, _, err := srv.SSH(fmt.Sprintf("curl %s:34567", srv.PrivateIP()))
		return err == nil, nil
	})
	if err != nil {
		c.Fatal("timed out waiting for omaha server to become active")
	}

	return fmt.Sprintf("%s:34567", srv.PrivateIP())
}

func configureMachineForUpdate(c cluster.TestCluster, m platform.Machine, addr string) {
	// update atomicly so nothing reading update.conf fails
	c.MustSSH(m, fmt.Sprintf(`sudo bash -c "cat >/etc/coreos/update.conf.new <<EOF
GROUP=developer
SERVER=http://%s/v1/update
EOF"`, addr))
	c.MustSSH(m, "sudo mv /etc/coreos/update.conf{.new,}")

	// dev key
	key := `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzFS5uVJ+pgibcFLD3kbY
k02Edj0HXq31ZT/Bva1sLp3Ysv+QTv/ezjf0gGFfASdgpz6G+zTipS9AIrQr0yFR
+tdp1ZsHLGxVwvUoXFftdapqlyj8uQcWjjbN7qJsZu0Ett/qo93hQ5nHW7Sv5dRm
/ZsDFqk2Uvyaoef4bF9r03wYpZq7K3oALZ2smETv+A5600mj1Xg5M52QFU67UHls
EFkZphrGjiqiCdp9AAbAvE7a5rFcJf86YR73QX08K8BX7OMzkn3DsqdnWvLB3l3W
6kvIuP+75SrMNeYAcU8PI1+bzLcAG3VN3jA78zeKALgynUNH50mxuiiU3DO4DZ+p
5QIDAQAB
-----END PUBLIC KEY-----`

	if kola.ForceFlatcarKey {
		// prod key
		// https://github.com/flatcar/coreos-overlay/blob/flatcar-master/coreos-base/coreos-au-key/files/official-v2.pub.pem
		key = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAw/NZ5Tvc93KynOLPDOxa
hyAGRKB2NvgF9l2A61SsFw5CuZc/k02u1/BvFehK4XL/eOo90Dt8A2l28D/YKs7g
2IPUSAnA9hc5OKBbpHsDzisxlAh7kg4FpeeJJWJMzO8NDCG5NZVqXEpGjCmX0qSh
5MLiTDr9dU2YhLo93/92dKnTvsLjUVv5wnuF55Lt2wJv4CbxVn4hHwotGfSomTBO
+7o6hE3VIIo1C6lkP+FAqMyWKA9s6U0x4tGxCXszW3hPWOANLIT4m0e55ayxiy5A
ESEVW/xx6Rul75u925m21AqA6wwaEB6ZPKTnUiWoNKNv1xi8LPIz12+0nuE6iT1K
jQIDAQAB
-----END PUBLIC KEY-----`
	}

	// inject key
	c.MustSSH(m, fmt.Sprintf(`sudo bash -c "cat >/etc/coreos/update-payload-key.pub.pem <<EOF
%s
EOF"`, key))

	c.MustSSH(m, "sudo mount --bind /etc/coreos/update-payload-key.pub.pem /usr/share/update_engine/update-payload-key.pub.pem")

	// disable reboot so the test has explicit control
	c.MustSSH(m, "sudo systemctl mask --now locksmithd.service")
	c.MustSSH(m, "sudo systemctl reset-failed locksmithd.service")

	c.MustSSH(m, "sudo systemctl restart update-engine.service")
}

func updateMachine(c cluster.TestCluster, m platform.Machine) {
	c.Logf("Triggering update_engine")

	out, stderr, err := m.SSH("update_engine_client -check_for_update")
	if err != nil {
		c.Fatalf("Executing update_engine_client failed: %v: %v: %s", out, err, stderr)
	}

	err = util.WaitUntilReady(600*time.Second, 10*time.Second, func() (bool, error) {
		envs, stderr, err := m.SSH("update_engine_client -status 2>/dev/null")
		if err != nil {
			return false, fmt.Errorf("checking status failed: %v: %s", err, stderr)
		}

		return splitNewlineEnv(string(envs))["CURRENT_OP"] == "UPDATE_STATUS_UPDATED_NEED_REBOOT", nil
	})
	if err != nil {
		c.Fatalf("waiting for UPDATE_STATUS_UPDATED_NEED_REBOOT: %v", err)
	}

	c.Logf("Rebooting test machine")

	if err = m.Reboot(); err != nil {
		c.Fatalf("reboot failed: %v", err)
	}
}

// splits newline-delimited KEY=VAL pairs into a map
func splitNewlineEnv(envs string) map[string]string {
	m := make(map[string]string)
	sc := bufio.NewScanner(strings.NewReader(envs))
	for sc.Scan() {
		spl := strings.SplitN(sc.Text(), "=", 2)
		m[spl[0]] = spl[1]
	}
	return m
}

func sysextBootLogic(c cluster.TestCluster) {
	// The first test case is to not use Ignition which means that the
	// set of systemd units in the initrd is different and we also
	// don't have Ignition mount the OEM partition
	noIgnition, err := c.NewMachine(nil)
	if err != nil {
		c.Fatalf("creating test machine: %v", err)
	}
	version := string(c.MustSSH(noIgnition, `set -euo pipefail; grep -m 1 "^VERSION=" /usr/lib/os-release | cut -d = -f 2`))
	if version == "" {
		c.Fatalf("Assertion for version string failed")
	}
	// We disable systemd-sysext because the sysext files are empty and will fail to load.
	// We test the following cases that differ from the test case covered by Ignition.
	// We set up symlinks to emulate that a previous sysext was active and we store the new sysext
	// in the rootfs instead of the OEM partition. The previous sysext is either
	// a) stored on the rootfs and will stay there and the new one is is moved to the OEM partition
	// b) stored on the OEM partition and gets moved to the rootfs and the new one is moved to the OEM partition
	_ = c.MustSSH(noIgnition, fmt.Sprintf(`set -euxo pipefail
sudo systemctl mask --now systemd-sysext ensure-sysext
sudo mkdir -p /etc/flatcar/sysext /etc/flatcar/oem-sysext /usr/share/oem/sysext /etc/extensions
echo ID=test | sudo tee /usr/share/oem/oem-release
echo myext | sudo tee /etc/flatcar/enabled-sysext.conf
sudo touch /usr/share/oem/sysext/active-oem-test /etc/flatcar/oem-sysext/oem-test-%[1]s.raw /etc/flatcar/oem-sysext/oem-test-1.2.3.raw /etc/flatcar/sysext/flatcar-myext-%[1]s.raw /etc/flatcar/sysext/flatcar-myext-1.2.3.raw
sudo ln -fs /etc/flatcar/oem-sysext/oem-test-1.2.3.raw /etc/extensions/oem-test.raw
sudo ln -fs /etc/flatcar/sysext/flatcar-myext-1.2.3.raw /etc/extensions/flatcar-myext.raw
`, version))
	if err := noIgnition.Reboot(); err != nil {
		c.Fatalf("couldn't reboot: %v", err)
	}
	// Check that the right symlinks are set up for case "a)" and prepare the next boot
	_ = c.MustSSH(noIgnition, fmt.Sprintf(`set -euxo pipefail
[ "$(readlink -f /etc/extensions/oem-test.raw)" = "/usr/share/oem/sysext/oem-test-%[1]s.raw" ] || { echo "OEM symlink wrong"; exit 1 ; }
[ "$(readlink -f /etc/extensions/flatcar-myext.raw)" = "/etc/flatcar/sysext/flatcar-myext-%[1]s.raw" ] || { echo "Extension symlink wrong"; exit 1; }
sudo mv /usr/share/oem/sysext/oem-test-%[1]s.raw /etc/flatcar/oem-sysext/
sudo mv /etc/flatcar/oem-sysext/oem-test-1.2.3.raw /usr/share/oem/sysext/
sudo ln -fs /usr/share/oem/sysext/oem-test-1.2.3.raw /etc/extensions/oem-test.raw
sudo ln -fs /etc/flatcar/sysext/flatcar-myext-1.2.3.raw /etc/extensions/flatcar-myext.raw
`, version))
	if err := noIgnition.Reboot(); err != nil {
		c.Fatalf("couldn't reboot: %v", err)
	}
	// Check that the boot logic set up the right sysext symlinks for case "b)"
	testCmds := fmt.Sprintf(`set -euxo pipefail
[ "$(readlink -f /etc/extensions/oem-test.raw)" = "/usr/share/oem/sysext/oem-test-%[1]s.raw" ] || { echo "OEM symlink wrong"; exit 1 ; }
[ "$(readlink -f /etc/extensions/flatcar-myext.raw)" = "/etc/flatcar/sysext/flatcar-myext-%[1]s.raw" ] || { echo "Extension symlink wrong"; exit 1; }
`, version)
	_ = c.MustSSH(noIgnition, testCmds+`[ -e "/etc/flatcar/oem-sysext/oem-test-1.2.3.raw" ] || { echo "Old sysext didn't get moved to rootfs"; exit 1; }`)
	noIgnition.Destroy()
	// The second test case is to use Ignition and Ignition will also
	// mount the OEM partition in the initrd and use a different systemd
	// target unit to pull initrd-setup-root-after-ignition in.
	// The covered case here for the logic is
	// c) where no previous sysext image is used and the new one already
	// is on the OEM partition
	// There is no need to cover this case in the manual setup nor
	// adding more Ignition tests for cases a) and b) because the
	// logic that is hit is the same.
	conf := conf.ContainerLinuxConfig(fmt.Sprintf(`storage:
  filesystems:
     - name: oem
       mount:
         device: "/dev/disk/by-label/OEM"
         format: "btrfs"
  files:
    - path: /oem-release
      filesystem: oem
      contents:
        inline: |
          ID=test
    - path: /sysext/active-oem-test
      filesystem: oem
    - path: /sysext/oem-test-%[1]s.raw
      filesystem: oem
    - path: /etc/flatcar/enabled-sysext.conf
      contents:
        inline: |
          myext
    - path: /etc/flatcar/sysext/flatcar-myext-%[1]s.raw
systemd:
  units:
    - name: systemd-sysext.service
      mask: true
    - name: ensure-sysext.service
      mask: true
`, version))
	withIgnition, err := c.NewMachine(conf)
	if err != nil {
		c.Fatalf("creating test machine: %v", err)
	}
	_ = c.MustSSH(withIgnition, testCmds)
	withIgnition.Destroy()
}
