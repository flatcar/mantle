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
	"net/http"
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
		// Skip AVC checks, we will do our own only on the
		// last boot logs, as the older logs may come from an
		// old version of Flatcar that still has some AVC
		// messages.
		Flags: []register.Flag{register.NoSELinuxAVCChecks},
	})
	register.Register(&register.Test{
		Name:        "cl.update.docker-btrfs-compat",
		Run:         btrfs_compat,
		ClusterSize: 1,
		NativeFuncs: map[string]func() error{
			"Omaha": Serve,
		},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv"},
		// This test verifies preservation of storage driver "btrfs" for docker.
		// Docker releases before v23 defaulted to btrfs if the docker can only run if the update payload to test is given.
		// The image passed must also be a release lower than or equal to 3760.0.0
		// because newer versions ship docker 24.
		EndVersion: semver.Version{Major: 3760},
		SkipFunc: func(version semver.Version, channel, arch, platform string) bool {
			return kola.UpdatePayloadFile == ""
		},
		// Skip AVC checks, we will do our own only on the
		// last boot logs, as the older logs may come from an
		// old version of Flatcar that still has some AVC
		// messages.
		Flags:   []register.Flag{register.NoSELinuxAVCChecks},
		Distros: []string{"cl"},
	})
	register.Register(&register.Test{
		Name:        "cl.update.oem",
		Run:         oemPayload,
		ClusterSize: 1,
		NativeFuncs: map[string]func() error{
			"Omaha": Serve,
		},
		Distros: []string{"cl"},
		// This test uses its own OEM files and shouldn't run on other platforms
		Platforms: []string{"qemu", "qemu-unpriv"},
		SkipFunc: func(version semver.Version, channel, arch, platform string) bool {
			// This test can only run if the update payload to test is given.
			// The image passed must also be an old release that does not have
			// the OEM sysext setup because we want to test the migration path
			// (see scripts/ci-automation/vendor-testing/qemu_update.sh)
			return kola.UpdatePayloadFile == ""
		},
		// This test is expected to run on a very old version as start image
		UserData: conf.ContainerLinuxConfig(`storage:
  filesystems:
    - name: oem
      mount:
        device: "/dev/disk/by-label/OEM"
        format: "ext4"
  files:
    - path: /oem-release
      filesystem: oem
      contents:
        inline: |
          ID=azure
    - path: /python/shouldbedeleted
      filesystem: oem
      contents:
        inline: |
          should be deleted because its part of the Azure OEM cleanup paths
    - path: /etc/systemd/system/waagent.service
      contents:
        inline: |
          [Service]
          ExecStart=/bin/echo "should be deleted because its part of the Azure OEM cleanup paths"
systemd:
  units:
    - name: chronyd.service
      mask: true
`),
		// Skip AVC checks, we will do our own only on the
		// last boot logs, as the older logs may come from an
		// old version of Flatcar that still has some AVC
		// messages.
		Flags: []register.Flag{register.NoSELinuxAVCChecks},
	})
	register.Register(&register.Test{
		Name:        "cl.sysext.boot.old",
		Run:         sysextBootLogicOld,
		ClusterSize: 0,
		Distros:     []string{"cl"},
		// This test is uses its own OEM files and shouldn't run on other platforms
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3481},
		EndVersion: semver.Version{Major: 3603},
	})
	register.Register(&register.Test{
		Name:        "cl.sysext.boot",
		Run:         sysextBootLogicNew,
		ClusterSize: 0,
		Distros:     []string{"cl"},
		// This test is uses its own OEM files and shouldn't run on other platforms
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3603},
	})
	register.Register(&register.Test{
		Name:        "cl.sysext.fallbackdownload",
		Run:         sysextFallbackDownload,
		ClusterSize: 0,
		Distros:     []string{"cl"},
		// This test is uses its own OEM files and shouldn't run on other platforms
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3620},
	})
	register.Register(&register.Test{
		Name:        "cl.update.payload-boot-part-too-small",
		Run:         payloadBootPartTooSmall,
		ClusterSize: 1,
		NativeFuncs: map[string]func() error{
			"Omaha": Serve,
		},
		Distros: []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv"},
		SkipFunc: func(version semver.Version, channel, arch, platform string) bool {
			return kola.UpdatePayloadFile == ""
		},
		Flags: []register.Flag{register.NoSELinuxAVCChecks},
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

func checkNoAVCMessages(c cluster.TestCluster, m platform.Machine) {
	version := c.MustSSH(m, `set -euo pipefail; grep -m 1 "^VERSION=" /usr/lib/os-release | cut -d = -f 2`)
	if len(version) == 0 {
		c.Fatalf("got an empty version from os-release")
	}

	sv, err := semver.NewVersion(string(version))
	if err != nil {
		c.Fatalf("failed to parse os-release version: %v", err)
	}

	if sv.LessThan(semver.Version{Major: kola.AVCChecksMajorVersion}) {
		// skip AVC checks altogether - too old Flatcar version
		return
	}

	// end with "true" to return 0 in case grep selects no lines and returns 1
	out, err := c.SSH(m, `journalctl -b | grep -ie 'avc:[[:space:]]*denied'; true`)
	if err != nil {
		c.Fatalf("failed to get AVC messages from last boot in journal from machine %s: %v", m.ID(), err)
	}
	if len(out) > 0 {
		c.Fatalf("found AVC messages in last boot logs on machine %s", m.ID())
	}
}

func payloadPrepareMachine(conf *conf.UserData, c cluster.TestCluster) (string, platform.Machine) {
	addr := configureOmahaServer(c, c.Machines()[0])

	// create the actual test machine, the machine
	// that is created by the test registration is
	// used to host the omaha server
	m, err := c.NewMachine(conf)
	if err != nil {
		c.Fatalf("creating test machine: %v", err)
	}

	// Machines are intentionally configured post-boot
	// via SSH to allow for testing versions which predate
	// Ignition
	configureMachineForUpdate(c, m, addr)
	tutil.AssertBootedUsr(c, m, "USR-A")

	return addr, m
}

func payloadPerformUpdate(addr string, m platform.Machine, c cluster.TestCluster) {
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

func payload(c cluster.TestCluster) {
	addr, m := payloadPrepareMachine(nil, c)
	payloadPerformUpdate(addr, m, c)
	checkNoAVCMessages(c, m)
}

func payloadBootPartTooSmall(c cluster.TestCluster) {
	addr, m := payloadPrepareMachine(nil, c)
	// Fill as much as we can (9999 MB)
	c.MustSSH(m, `sudo dd if=/dev/zero of=/boot/increase_boot_part_usage bs=1M count=9999 || true`)
	configureMachineForUpdate(c, m, addr)
	updateMachineBootPartTooSmall(c, m)
	tutil.AssertBootedUsr(c, m, "USR-A")

	// reboot to make sure that the machine can boot with /boot partition 100% full
	c.Logf("Rebooting test machine")
	if err := m.Reboot(); err != nil {
		c.Fatalf("reboot failed: %v", err)
	}
	tutil.AssertBootedUsr(c, m, "USR-A")
}

func updateMachineBootPartTooSmall(c cluster.TestCluster, m platform.Machine) {
	c.Logf("Triggering update_engine")

	out, stderr, err := m.SSH("update_engine_client -check_for_update")
	if err != nil {
		c.Fatalf("Executing update_engine_client failed: %v: %v: %s", out, err, stderr)
	}

	err = util.WaitUntilReady(60*time.Second, 10*time.Second, func() (bool, error) {
		envs, stderr, err := m.SSH("update_engine_client -status 2>/dev/null")
		if err != nil {
			return false, fmt.Errorf("checking update_engine_client status failed: %v: %s", err, stderr)
		}

		return splitNewlineEnv(string(envs))["CURRENT_OP"] == "UPDATE_STATUS_IDLE", nil
	})
	if err != nil {
		c.Fatalf("Update did not fail: %v", err)
	}

	failure_message := string(c.MustSSH(m, `journalctl -xeu update-engine.service | grep -i 'Failed to copy kernel from /boot/flatcar/vmlinuz-a to /boot/flatcar/vmlinuz-b'`))
	if failure_message == "" {
		c.Fatalf("Failure message not found in the update-engine.service: 'Failed to copy kernel from /boot/flatcar/vmlinuz-a to /boot/flatcar/vmlinuz-b'")
	}
}

func btrfs_compat(c cluster.TestCluster) {
	conf := conf.ContainerLinuxConfig(`
systemd:
  units:
    - name: format-var-lib-docker.service
      enabled: true
      contents: |
        [Unit]
        Before=docker.service var-lib-docker.mount
        ConditionPathExists=!/var/lib/docker.btrfs
        [Service]
        Type=oneshot
        ExecStart=/usr/bin/truncate --size=25G /var/lib/docker.btrfs
        ExecStart=/usr/sbin/mkfs.btrfs /var/lib/docker.btrfs
        [Install]
        WantedBy=multi-user.target
    - name: var-lib-docker.mount
      enabled: true
      contents: |
        [Unit]
        Before=docker.service
        After=format-var-lib-docker.service
        Requires=format-var-lib-docker.service
        [Install]
        RequiredBy=docker.service
        [Mount]
        What=/var/lib/docker.btrfs
        Where=/var/lib/docker
        Type=btrfs
        Options=loop,discard`)

	addr, m := payloadPrepareMachine(conf, c)

	// We need to populate the docker storage.
	// If empty, docker 23 and above will silently switch to the 'overlay2' driver.
	// If populated, docker SHOULD preserve the btrfs driver. That's what we test for later.
	c.MustSSH(m, `docker info | grep 'Storage Driver: btrfs' || { echo "ERROR: expected BTRFS storage driver"; docker info; exit 1; }`)
	c.MustSSH(m, `docker run -i --name docker_btrfs_driver_test alpine ls /`)

	payloadPerformUpdate(addr, m, c)

	c.MustSSH(m, `docker info | grep 'Storage Driver: btrfs' || { echo "ERROR: expected BTRFS storage driver"; docker info; exit 1; }`)

	c.MustSSH(m, `docker image ls | grep alpine || { echo "ERROR: Container image 'alpine' disappeared after update"; docker image ls; exit 1; } `)
	c.MustSSH(m, `docker ps --all | grep docker_btrfs_driver_test || { echo "ERROR: Container 'docker_btrfs_driver_test' disappeared after update"; docker ps --all; exit 1; } `)
	checkNoAVCMessages(c, m)
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

func oemPayload(c cluster.TestCluster) {
	// This test first consumes a payload locally given to kola
	// and then also uses this payload again with flatcar-update,
	// where the OEM extension will be fallback downloaded from
	// bincache during the postinst hook, and then after the reboot
	// there is a final update to itself with flatcar-update where
	// the OEM extension is passed expliclitly and the machine
	// reboots once more to migrate to the sysext OEM setup.
	m := c.Machines()[0]
	// The instance should now host its own update with a kolet
	addr := configureOmahaServer(c, m)
	configureMachineForUpdate(c, m, addr)
	tutil.AssertBootedUsr(c, m, "USR-A")
	// Updating will fail if there is no payload on bincache,
	// so don't expect this to work for local or GitHub Action builds
	updateMachine(c, m)
	tutil.AssertBootedUsr(c, m, "USR-B")
	tutil.InvalidateUsrPartition(c, m, "USR-A")
	// Check that the instance is not yet migrated
	_ = c.MustSSH(m, `test -e /oem/python/shouldbedeleted && test -e /etc/systemd/system/waagent.service`)
	_ = c.MustSSH(m, `test ! -e /oem/sysext/active-oem-azure`)
	version := string(c.MustSSH(m, `set -euo pipefail; grep -m 1 "^VERSION=" /usr/lib/os-release | cut -d = -f 2`))
	if version == "" {
		c.Fatalf("Assertion for version string failed")
	}
	_ = c.MustSSH(m, `test -e /oem/sysext/oem-azure-`+version+`.raw`)
	arch := strings.SplitN(kola.QEMUOptions.Board, "-", 2)[0]
	_ = c.MustSSH(m, `curl -fsSLO --retry-delay 1 --retry 60 --retry-connrefused --retry-max-time 60 --connect-timeout 20 https://bincache.flatcar-linux.net/images/`+arch+`/`+version+`/flatcar_test_update-oem-azure.gz`)
	_ = c.MustSSH(m, `curl -fsSLO --retry-delay 1 --retry 60 --retry-connrefused --retry-max-time 60 --connect-timeout 20 https://bincache.flatcar-linux.net/images/`+arch+`/`+version+`/flatcar_test_update-flatcar-tools.gz`)
	_ = c.MustSSH(m, `sudo flatcar-update --to-version `+version+` --to-payload /updates/update.gz --extension ./flatcar_test_update-oem-azure.gz --extension ./flatcar_test_update-flatcar-tools.gz --disable-afterwards --force-dev-key`)

	checkNoAVCMessages(c, m)

	c.Logf("Rebooting test machine after flatcar-update run (2nd reboot)")
	if err := m.Reboot(); err != nil {
		c.Fatalf("reboot failed: %v", err)
	}
	tutil.AssertBootedUsr(c, m, "USR-A")
	// Check that the instance has migrated
	_ = c.MustSSH(m, `test ! -e /oem/python/shouldbedeleted && test ! -e /etc/systemd/system/waagent.service`)
	_ = c.MustSSH(m, `test -e /oem/sysext/active-oem-azure`)
	_ = c.MustSSH(m, `systemd-sysext status --json=pretty | jq --raw-output '.[] | select(.hierarchy == "/usr") | .extensions[]' | grep -q oem-azure`)
	checkNoAVCMessages(c, m)
}

func sysextBootLogicOld(c cluster.TestCluster) {
	sysextBootLogic(c, "/usr/share/oem")
}

func sysextBootLogicNew(c cluster.TestCluster) {
	sysextBootLogic(c, "/oem")
}

func sysextBootLogic(c cluster.TestCluster, oemMountpoint string) {
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
sudo mkdir -p /etc/flatcar/sysext /etc/flatcar/oem-sysext %[2]s/sysext /etc/extensions
echo ID=test | sudo tee %[2]s/oem-release
echo myext | sudo tee /etc/flatcar/enabled-sysext.conf
sudo touch %[2]s/sysext/active-oem-test /etc/flatcar/oem-sysext/oem-test-%[1]s.raw /etc/flatcar/oem-sysext/oem-test-1.2.3.raw /etc/flatcar/sysext/flatcar-myext-%[1]s.raw /etc/flatcar/sysext/flatcar-myext-1.2.3.raw
sudo ln -fs /etc/flatcar/oem-sysext/oem-test-1.2.3.raw /etc/extensions/oem-test.raw
sudo ln -fs /etc/flatcar/sysext/flatcar-myext-1.2.3.raw /etc/extensions/flatcar-myext.raw
`, version, oemMountpoint))
	if err := noIgnition.Reboot(); err != nil {
		c.Fatalf("couldn't reboot: %v", err)
	}
	// Check that the right symlinks are set up for case "a)" and prepare the next boot
	_ = c.MustSSH(noIgnition, fmt.Sprintf(`set -euxo pipefail
[ "$(readlink -f /etc/extensions/oem-test.raw)" = "%[2]s/sysext/oem-test-%[1]s.raw" ] || { echo "OEM symlink wrong"; exit 1 ; }
[ "$(readlink -f /etc/extensions/flatcar-myext.raw)" = "/etc/flatcar/sysext/flatcar-myext-%[1]s.raw" ] || { echo "Extension symlink wrong"; exit 1; }
sudo mv %[2]s/sysext/oem-test-%[1]s.raw /etc/flatcar/oem-sysext/
sudo mv /etc/flatcar/oem-sysext/oem-test-1.2.3.raw %[2]s/sysext/
sudo ln -fs %[2]s/sysext/oem-test-1.2.3.raw /etc/extensions/oem-test.raw
sudo ln -fs /etc/flatcar/sysext/flatcar-myext-1.2.3.raw /etc/extensions/flatcar-myext.raw
`, version, oemMountpoint))
	if err := noIgnition.Reboot(); err != nil {
		c.Fatalf("couldn't reboot: %v", err)
	}
	// Check that the boot logic set up the right sysext symlinks for case "b)"
	testCmds := fmt.Sprintf(`set -euxo pipefail
[ "$(readlink -f /etc/extensions/oem-test.raw)" = "%[2]s/sysext/oem-test-%[1]s.raw" ] || { echo "OEM symlink wrong"; exit 1 ; }
[ "$(readlink -f /etc/extensions/flatcar-myext.raw)" = "/etc/flatcar/sysext/flatcar-myext-%[1]s.raw" ] || { echo "Extension symlink wrong"; exit 1; }
`, version, oemMountpoint)
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

func sysextFallbackDownload(c cluster.TestCluster) {
	// The first test case is to not use Ignition which means that the
	// set of systemd units in the initrd is different and we also
	// don't have Ignition mount the OEM partition
	m, err := c.NewMachine(nil)
	if err != nil {
		c.Fatalf("creating test machine: %v", err)
	}

	// Check that we don't have an OEM sysext image
	_ = c.MustSSH(m, `test ! -e /etc/extensions/oem-qemu.raw`)

	version := string(c.MustSSH(m, `set -euo pipefail; grep -m 1 "^VERSION=" /usr/lib/os-release | cut -d = -f 2`))
	if version == "" {
		c.Fatalf("Assertion for version string failed")
	}

	arch := strings.SplitN(kola.QEMUOptions.Board, "-", 2)[0]

	client := &http.Client{
		Timeout: time.Second * 60,
	}
	// Check that we are on a dev build. Overwriting the pub key with a bind mount won't work for the initrd.
	keySum := string(c.MustSSH(m, `md5sum /usr/share/update_engine/update-payload-key.pub.pem | cut -d " " -f 1`))
	if keySum != "7192addf4a7f890c0057d21653eff2ea" {
		c.Skip("Test skipped, only dev builds are supported")
	}
	// For simplicity, only support bincache, the test will be skipped on GitHub PRs and release builds.
	// The URL comes from bootengine:dracut/99setup-root/initrd-setup-root-after-ignition where it would be flatcar_test_update-oem-qemu.gz
	// but here we instead test for version.txt to still run and fail the test if that file is missing for unknown reasons
	reply, err := client.Head(fmt.Sprintf("https://bincache.flatcar-linux.net/images/%s/%s/version.txt", arch, version))
	if err != nil || reply.StatusCode != 200 {
		c.Skip("pre-check failed (URL not found?)")
	}

	_ = c.MustSSH(m, `sudo mkdir -p /oem/sysext && sudo touch /oem/sysext/active-oem-qemu && echo ID=qemu | sudo tee /oem/oem-release > /dev/null`)

	c.Logf("Rebooting test machine")

	if err = m.Reboot(); err != nil {
		c.Fatalf("reboot failed: %v", err)
	}

	// test -e resolves the symlink and checks that the target also exists
	_ = c.MustSSH(m, `test -e /etc/extensions/oem-qemu.raw`)
	m.Destroy()
}
