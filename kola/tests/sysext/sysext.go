// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package sysext

import (
	"fmt"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/kola/tests/util"
	"github.com/flatcar/mantle/platform/conf"
)

type (
	configTemplateParameters struct {
		DownloadLibraryBase64Contents    string
		DevContainerScriptBase64Contents string
		MainScriptBase64Contents         string
		CheckScriptBase64Contents        string
	}
)

var (
	devContainerScript = util.TrimLeftSpace(`
#!/bin/bash

set -x

set -euo pipefail

version=$(source /etc/os-release; echo "${VERSION}")
version_id=$(source /etc/os-release; echo "${VERSION_ID}")
board=$(source /usr/share/flatcar/release; echo "${FLATCAR_RELEASE_BOARD}")

mkdir -p /work/sysext_rootfs/usr/share/flatcar-sysext-kola-test
echo "${version_id}" >/work/sysext_rootfs/usr/share/flatcar-sysext-kola-test/file
mkdir -p /work/sysext_rootfs/usr/lib/extension-release.d
sysext_arch=x86-64
if [[ "${board}" = 'arm64-usr' ]]; then sysext_arch=arm64; fi
metadata=(
    'ID=flatcar'
    "VERSION_ID=${version_id}"
    "ARCHITECTURE=${sysext_arch}"
)
metadata_file=/work/sysext_rootfs/usr/lib/extension-release.d/extension-release.oem-test
printf '%s\n' "${metadata[@]}" >"${metadata_file}"
mksquashfs /work/sysext_rootfs "/work/oem-test-${version}.raw" -all-root
`)

	mainScript = util.TrimLeftSpace(`
#!/bin/bash

set -x

set -euo pipefail

source /home/core/download-library.sh

download_dev_container_image flatcar_developer_container.bin

# This is where the built sysext will be stored
workdir="${PWD}/dev-container-workdir-${RANDOM}"
mkdir -p "${workdir}"

sudo systemd-nspawn \
        --console=pipe \
        --bind-ro=/home/core/dev-container-script.sh \
        --bind="${workdir}:/work" \
        --image=flatcar_developer_container.bin \
        --machine=flatcar-developer-container \
        /bin/bash /home/core/dev-container-script.sh

version=$(source /etc/os-release; echo "${VERSION}")
sysext_file="${workdir}/oem-test-${version}.raw"

if [[ ! -e "${sysext_file}" ]]; then
        echo "Expected ${sysext_file} to exist, contents of workdir:"
        ls -la "${workdir}"
        exit 1
fi

# Rebrand our image from "qemu" to "test".
if [[ ! -e /oem/oem-release ]]; then
        # This is the regular case, on the generic image used for kola
        # QEMU tests there is no OEM setup
        printf '%s\n' 'ID=test' 'VERSION_ID=1.0.0' 'NAME=testing stuff' | sudo tee /oem/oem-release >/dev/null
else
        # This only works when the OEM setup is optional, e.g., with
        # the QEMU OEM image
        sudo sed -i'' -e 's/^ID=.*/ID=test/' /oem/oem-release
fi
sudo mkdir -p /oem/sysext
sudo mv "${sysext_file}" /oem/sysext
sudo touch /oem/sysext/active-oem-test
# We keep /var/log to keep journald logs in case something goes wrong.
sudo flatcar-reset --keep-machine-id --keep-paths /var/log
`)

	checkScript = util.TrimLeftSpace(`
#!/bin/bash

set -x

set -euo pipefail

list_out=$(systemd-sysext list --json=pretty)
status_out=$(systemd-sysext status --json=pretty)
printf 'sysext list:\n%s\nsysext status:\n%s\n' "${list_out}" "${status_out}"

list_oem_test=$(jq '.[] | select(.name == "oem-test")' <<<"${list_out}")

if [[ -z "${list_oem_test}" ]]; then
        echo "oem-test image is not listed"
        exit 1
fi

oem_test_type=$(jq --raw-output '.type' <<<"${list_oem_test}")
if [[ "${oem_test_type}" != 'raw' ]]; then
        echo "oem test image type should be 'raw', is '${oem_test_type}'"
        exit 1
fi

oem_test_path=$(jq --raw-output '.path' <<<"${list_oem_test}")
if [[ "${oem_test_path}" != '/etc/extensions/oem-test.raw' ]]; then
        echo "oem test image path should be '/etc/extensions/oem-test.raw', is '${oem_test_path}'"
        exit 1
fi

status_usr=$(jq '.[] | select(.hierarchy == "/usr")' <<<"${status_out}")
if [[ -z "${status_usr}" ]]; then
        echo "no sysext hierarchy for /usr?"
        exit 1
fi

status_usr_extensions_oem_test=$(jq --raw-output '.extensions[] | select(. == "oem-test")' <<<"${status_usr}")
if [[ "${status_usr_extensions_oem_test}" != 'oem-test' ]]; then
        echo "oem-test sysext is not active on /usr"
        exit 1
fi

f=/usr/share/flatcar-sysext-kola-test/file
if [[ ! -e "${f}" ]]; then
        echo "Missing file from sysext"
        exit 1
fi
got=$(cat "${f}")
ex=$(source /etc/os-release; echo "${VERSION_ID}")
if [[ "${got}" != "${ex}" ]]; then
        echo "Bad content of sysext file (got '${got}', expected '${ex}')"
        exit 1
fi
`)

	butaneTemplate = util.TrimLeftSpace(`
variant: flatcar
version: 1.0.0
storage:
  files:
    - path: /home/core/download-library.sh
      mode: 0644
      contents:
        source: "data:text/plain;base64,{{ .DownloadLibraryBase64Contents }}"
      user:
        name: core
      group:
        name: core
    - path: /home/core/dev-container-script.sh
      mode: 0755
      contents:
        source: "data:text/plain;base64,{{ .DevContainerScriptBase64Contents }}"
      user:
        name: core
      group:
        name: core
    - path: /home/core/main-script.sh
      mode: 0755
      contents:
        source: "data:text/plain;base64,{{ .MainScriptBase64Contents }}"
      user:
        name: core
      group:
        name: core
    - path: /oem/check-script.sh
      # set overwrite to true as flatcar-reset will not remove it
      overwrite: true
      mode: 0755
      contents:
        source: "data:text/plain;base64,{{ .CheckScriptBase64Contents }}"
      user:
        name: core
      group:
        name: core
`)
)

func init() {
	register.Register(&register.Test{
		Name:        "sysext.simple.old",
		Run:         checkSysextSimpleOld,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3185},
		EndVersion: semver.Version{Major: 3603},
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
		Name:        "sysext.simple",
		Run:         checkSysextSimpleNew,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms:  []string{"qemu", "qemu-unpriv", "azure"},
		MinVersion: semver.Version{Major: 3603},
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
		Name:        "sysext.custom-docker.torcx",
		Run:         checkSysextCustomDocker,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3185},
		// Torcx was retired after release 3760.
		EndVersion: semver.Version{Major: 3760},
		UserData: conf.ContainerLinuxConfig(`storage:
  files:
    - path: /etc/systemd/system-generators/torcx-generator
  directories:
    - path: /etc/extensions/docker-flatcar
    - path: /etc/extensions/containerd-flatcar`),
	})
	register.Register(&register.Test{
		Name:        "sysext.custom-docker.sysext",
		Run:         checkSysextCustomDocker,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv", "azure"},
		// Sysext docker was introduced after release 3760.
		// NOTE that 3761 is a developer version which was never released.
		// However, the next largest Alpha major release shipped sysext.
		MinVersion: semver.Version{Major: 3761},
		UserData: conf.Butane(`
variant: flatcar
version: 1.0.0
storage:
  links:
  - path: /etc/extensions/docker-flatcar.raw
    target: /dev/null
    hard: false
    overwrite: true
  - path: /etc/extensions/containerd-flatcar.raw
    target: /dev/null
    hard: false
    overwrite: true
`),
	})
	register.Register(&register.Test{
		Name:        "sysext.custom-oem",
		Run:         checkSysextCustomOEM,
		ClusterSize: 0,
		Distros:     []string{"cl"},
		// This test is uses its own OEM files and shouldn't run on other platforms
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3603},
		NativeFuncs: map[string]func() error{
			"Http": util.Serve,
		},
	})
}

func checkHelper(c cluster.TestCluster, oemMountpoint string) {
	_ = c.MustSSH(c.Machines()[0], `grep -m 1 '^sysext works$' /usr/hello-sysext`)
	// "mountpoint /oem" (or "/usr/share/oem") is too loose for
	// our purposes, because we want to check if the mount point
	// is accessible and "df" only shows these by default
	target := c.MustSSH(c.Machines()[0], fmt.Sprintf(`if [ -e /dev/disk/by-label/OEM ]; then df --output=target | grep %s; fi`, oemMountpoint))
	// check against multiple entries which is not wanted
	if string(target) != oemMountpoint {
		c.Fatalf("should get %q, got %q", oemMountpoint, string(target))
	}
}

func checkSysextSimpleOld(c cluster.TestCluster) {
	checkSysextSimple(c, "/usr/share/oem")
}

func checkSysextSimpleNew(c cluster.TestCluster) {
	checkSysextSimple(c, "/oem")
}

func checkSysextSimple(c cluster.TestCluster, oemMountpoint string) {
	// First check directly after boot
	checkHelper(c, oemMountpoint)
	_ = c.MustSSH(c.Machines()[0], `sudo systemctl restart systemd-sysext`)
	// Second check after reloading the extensions (e.g., to add/remove/update them)
	checkHelper(c, oemMountpoint)
}

func checkSysextCustomDocker(c cluster.TestCluster) {
	arch := strings.SplitN(kola.QEMUOptions.Board, "-", 2)[0]
	if arch == "arm64" {
		arch = "aarch64"
	} else {
		arch = "x86_64"
	}

	cmdNotWorking := `if docker run --rm ghcr.io/flatcar/busybox true; then exit 1; fi`
	cmdWorking := `docker run --rm ghcr.io/flatcar/busybox echo Hello World`
	// First assert that Docker doesn't work because Torcx is disabled
	_ = c.MustSSH(c.Machines()[0], cmdNotWorking)
	// We build a custom sysext image locally because we don't host them somewhere yet
	_ = c.MustSSH(c.Machines()[0], `git clone https://github.com/flatcar/sysext-bakery.git && git -C sysext-bakery checkout 9850ffd5b2353f45a9b3bf4fb84f8138a149e3e7`)
	// Flatcar has no mksquashfs and btrfs is missing a bugfix but at least ext4 works
	// The first test is for a fixed Docker version, which with the time will get old and older but is still expected to work because users may also "freeze" their Docker version this way
	_ = c.MustSSH(c.Machines()[0], fmt.Sprintf(`ARCH=%[1]s ONLY_DOCKER=1 FORMAT=ext4 sysext-bakery/create_docker_sysext.sh 20.10.21 docker && ARCH=%[1]s ONLY_CONTAINERD=1 FORMAT=ext4 sysext-bakery/create_docker_sysext.sh 20.10.21 containerd && sudo mv docker.raw containerd.raw /etc/extensions/`, arch))
	_ = c.MustSSH(c.Machines()[0], `sudo systemctl restart systemd-sysext`)
	// We should now be able to use Docker
	_ = c.MustSSH(c.Machines()[0], cmdWorking)
	// The next test is with a recent Docker version, here the one from the Flatcar image to couple it to something that doesn't change under our feet
	version := string(c.MustSSH(c.Machines()[0], `bzcat /usr/share/licenses/licenses.json.bz2 | grep -m 1 -o 'app-\(containers\|emulation\)/docker-[0-9][^:]*' | cut -d - -f 3`))
	_ = c.MustSSH(c.Machines()[0], fmt.Sprintf(`ONLY_DOCKER=1 FORMAT=ext4 ARCH=%[2]s sysext-bakery/create_docker_sysext.sh %[1]s docker && ONLY_CONTAINERD=1 FORMAT=ext4 ARCH=%[2]s sysext-bakery/create_docker_sysext.sh %[1]s containerd && sudo mv docker.raw containerd.raw /etc/extensions/`, version, arch))
	_ = c.MustSSH(c.Machines()[0], `sudo systemctl restart systemd-sysext && sudo systemctl restart docker containerd`)
	// We should now still be able to use Docker
	_ = c.MustSSH(c.Machines()[0], cmdWorking)
}

func checkSysextCustomOEM(c cluster.TestCluster) {
	downloadLibrary, err := util.DevContainerDownloadLibrary()
	if err != nil {
		c.Fatalf("creating a dev container download script failed: %v", err)
	}

	userdata, err := prepareUserData(downloadLibrary)
	if err != nil {
		c.Fatalf("preparing user data failed: %v", err)
	}
	machine, err := util.NewMachineWithLargeDisk(c, "5G", userdata)
	if err != nil {
		c.Fatalf("creating a machine failed: %v", err)
	}
	err = util.ConfigureDevContainerHTTPServer(c, machine)
	if err != nil {
		c.Fatalf("configuring local HTTP server for dev container image failed: %v", err)
	}

	if _, err := c.SSH(machine, "/home/core/main-script.sh"); err != nil {
		c.Fatalf("main script failed: %v", err)
	}
	if err := machine.Reboot(); err != nil {
		c.Fatalf("could not reboot: %v", err)
	}
	if _, err := c.SSH(machine, "/oem/check-script.sh"); err != nil {
		c.Fatalf("check script failed: %v", err)
	}
}

func prepareUserData(downloadLibrary string) (*conf.UserData, error) {
	downloadLibraryBase64 := util.ToBase64(downloadLibrary)
	mainScriptBase64 := util.ToBase64(mainScript)
	devContainerScriptBase64 := util.ToBase64(devContainerScript)
	checkScriptBase64 := util.ToBase64(checkScript)
	configParameters := configTemplateParameters{
		DownloadLibraryBase64Contents:    downloadLibraryBase64,
		DevContainerScriptBase64Contents: devContainerScriptBase64,
		MainScriptBase64Contents:         mainScriptBase64,
		CheckScriptBase64Contents:        checkScriptBase64,
	}
	config, err := util.ExecNamedTemplate(butaneTemplate, "butane config", configParameters)
	if err != nil {
		return nil, err
	}
	return conf.Butane(config), nil
}
