// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package systemd

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/platform/local"
	"github.com/flatcar/mantle/platform/machine/qemu"
	"github.com/flatcar/mantle/platform/machine/unprivqemu"
	"github.com/flatcar/mantle/util"
)

// BEGIN FOR UTILS
func trimLeftSpace(contents string) string {
	return strings.TrimLeftFunc(contents, unicode.IsSpace)
}

// END FOR UTILS

type (
	downloadScriptParameters struct {
		ImageDirectoryURLTemplate string
	}

	configTemplateParameters struct {
		DownloadScriptBase64Contents     string
		DevContainerScriptBase64Contents string
		MainScriptBase64Contents         string
		CheckScriptBase64Contents        string
	}
)

var (
	// BEGIN FOR UTILS
	downloadScriptTemplate = trimLeftSpace(`
#!/bin/bash

set -x

set -euo pipefail

output_bin="${1}"

function process_template() {
        local template="${1}"; shift
        local arch="${1}"; shift
        local version="${1}"; shift
        local result="${template}"

        result="${result//@ARCH@/${arch}}"
        result="${result//@VERSION@/${version}}"

        echo "${result}"
}

source /usr/share/flatcar/release

ARCH="${FLATCAR_RELEASE_BOARD/-usr/}"
VERSION="${FLATCAR_RELEASE_VERSION}"
IMAGE_URL=$(process_template '{{ .ImageDirectoryURLTemplate }}/flatcar_developer_container.bin.bz2' "${ARCH}" "${VERSION}")

echo "Fetching developer container from ${IMAGE_URL}"
# Stolen from copy_from_buildcache in ci_automation_common.sh. Not
# using --output-dir option as this seems to be quite a new addition
# and curl on older version of Flatcar does not understand it.
curl --fail --silent --show-error --location --retry-delay 1 --retry 60 \
        --retry-connrefused --retry-max-time 60 --connect-timeout 20 \
        --remote-name "${IMAGE_URL}"

bzip2cat=bzcat
if command -v lbzcat; then
        bzip2cat=lbzcat
fi

# The image file takes over 6Gb after normal unpacking, but a lot of
# it is just zeros. Use cp --sparse=always to avoid unnecessary disk
# space waste. Especially that we may not have 6Gb of disk space
# available.
cp --sparse=always <("${bzip2cat}" flatcar_developer_container.bin.bz2) "${output_bin}"
`)
	// END FOR UTILS

	devContainerScript = trimLeftSpace(`
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

	mainScript = trimLeftSpace(`
#!/bin/bash

set -x

set -euo pipefail

/home/core/download-script.sh flatcar_developer_container.bin

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

	checkScript = trimLeftSpace(`
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

	butaneTemplate = trimLeftSpace(`
variant: flatcar
version: 1.0.0
storage:
  files:
    - path: /home/core/download-script.sh
      mode: 0755
      contents:
        source: "data:text/plain;base64,{{ .DownloadScriptBase64Contents }}"
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
	register.Register(&register.Test{
		Name:        "systemd.sysext.custom-oem",
		Run:         checkSysextCustomOEM,
		ClusterSize: 0,
		Distros:     []string{"cl"},
		// This test is uses its own OEM files and shouldn't run on other platforms
		Platforms:  []string{"qemu", "qemu-unpriv"},
		MinVersion: semver.Version{Major: 3605},
		NativeFuncs: map[string]func() error{
			"Http": Serve,
		},
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
	_ = c.MustSSH(c.Machines()[0], `git clone https://github.com/flatcar/sysext-bakery.git && git -C sysext-bakery checkout e68d2fe25c8412f4774477d1d75c40f615145c46`)
	// Flatcar has no mksquashfs and btrfs is missing a bugfix but at least ext4 works
	// The first test is for a fixed Docker version, which with the time will get old and older but is still expected to work because users may also "freeze" their Docker version this way
	_ = c.MustSSH(c.Machines()[0], fmt.Sprintf(`ARCH=%[1]s ONLY_DOCKER=1 FORMAT=ext4 sysext-bakery/create_docker_sysext.sh 20.10.21 docker && ARCH=%[1]s ONLY_CONTAINERD=1 FORMAT=ext4 sysext-bakery/create_docker_sysext.sh 20.10.21 containerd && sudo mv docker.raw containerd.raw /etc/extensions/`, arch))
	_ = c.MustSSH(c.Machines()[0], `sudo systemctl restart systemd-sysext`)
	// We should now be able to use Docker
	_ = c.MustSSH(c.Machines()[0], cmdWorking)
	// The next test is with a recent Docker version, here the one from the Flatcar image to couple it to something that doesn't change under our feet
	version := string(c.MustSSH(c.Machines()[0], `bzcat /usr/share/licenses/licenses.json.bz2 | grep -m 1 -o 'app-emulation/docker[^:]*' | cut -d - -f 3`))
	_ = c.MustSSH(c.Machines()[0], fmt.Sprintf(`ONLY_DOCKER=1 FORMAT=ext4 ARCH=%[2]s sysext-bakery/create_docker_sysext.sh %[1]s docker && ONLY_CONTAINERD=1 FORMAT=ext4 ARCH=%[2]s sysext-bakery/create_docker_sysext.sh %[1]s containerd && sudo mv docker.raw containerd.raw /etc/extensions/`, version, arch))
	_ = c.MustSSH(c.Machines()[0], `sudo systemctl restart systemd-sysext && sudo systemctl restart docker containerd`)
	// We should now still be able to use Docker
	_ = c.MustSSH(c.Machines()[0], cmdWorking)
}

func checkSysextCustomOEM(c cluster.TestCluster) {
	// BEGIN COPIED STUFF
	devcontainerURL := kola.DevcontainerURL
	if kola.DevcontainerFile != "" {
		// This URL is deterministic as it runs on the started machine.
		devcontainerURL = "http://localhost:8080"
	}
	// END COPIED STUFF

	userdata, err := prepareUserData(devcontainerURL)
	if err != nil {
		c.Fatalf("preparing user data failed: %v", err)
	}
	// BEGIN COPIED STUFF
	machine, err := newMachineWithLargeDisk(c, userdata)
	if err != nil {
		c.Fatalf("creating a machine failed: %v", err)
	}

	if kola.DevcontainerFile != "" {
		configureHTTPServer(c, machine)
	}
	// END COPIED STUFF

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

func prepareUserData(devContainerURL string) (*conf.UserData, error) {
	scriptParameters := downloadScriptParameters{
		ImageDirectoryURLTemplate: devContainerURL,
	}
	downloadScript, err := executeTemplate(downloadScriptTemplate, "download script", scriptParameters)
	if err != nil {
		return nil, err
	}
	downloadScriptBase64 := base64.StdEncoding.EncodeToString(([]byte)(downloadScript))
	mainScriptBase64 := base64.StdEncoding.EncodeToString(([]byte)(mainScript))
	devContainerScriptBase64 := base64.StdEncoding.EncodeToString(([]byte)(devContainerScript))
	checkScriptBase64 := base64.StdEncoding.EncodeToString(([]byte)(checkScript))
	configParameters := configTemplateParameters{
		DownloadScriptBase64Contents:     downloadScriptBase64,
		DevContainerScriptBase64Contents: devContainerScriptBase64,
		MainScriptBase64Contents:         mainScriptBase64,
		CheckScriptBase64Contents:        checkScriptBase64,
	}
	config, err := executeTemplate(butaneTemplate, "butane config", configParameters)
	if err != nil {
		return nil, err
	}
	return conf.Butane(config), nil
}

// BEGIN COPIED STUFF
func executeTemplate(contents, name string, parameters any) (string, error) {
	tmpl, err := template.New(name).Parse(contents)
	if err != nil {
		return "", fmt.Errorf("parsing %s as a template failed: %w", name, err)
	}
	buf := bytes.Buffer{}
	if err := tmpl.Execute(&buf, parameters); err != nil {
		return "", fmt.Errorf("executing %s template failed: %w", name, err)
	}
	return buf.String(), nil
}

func newMachineWithLargeDisk(c cluster.TestCluster, userData *conf.UserData) (platform.Machine, error) {
	options := platform.MachineOptions{
		ExtraPrimaryDiskSize: "5G",
	}
	switch pc := c.Cluster.(type) {
	case *qemu.Cluster:
		return pc.NewMachineWithOptions(userData, options)
	case *unprivqemu.Cluster:
		return pc.NewMachineWithOptions(userData, options)
	}
	return nil, errors.New("unknown cluster type, this test should only be running on qemu or qemu-unpriv platforms")
}

func Serve() error {
	httpServer := local.SimpleHTTP{}
	return httpServer.Serve()
}

func configureHTTPServer(c cluster.TestCluster, srv platform.Machine) {
	// manually copy Kolet on the host, as the initial size cluster is 0.
	kola.ScpKolet(c, strings.SplitN(kola.QEMUOptions.Board, "-", 2)[0])

	in, err := os.Open(kola.DevcontainerFile)
	if err != nil {
		c.Fatalf("opening dev container file: %v", err)
	}

	defer in.Close()

	if err := platform.InstallFile(in, srv, "/var/www/flatcar_developer_container.bin.bz2"); err != nil {
		c.Fatalf("copying dev container to HTTP server: %v", err)
	}

	c.MustSSH(srv, fmt.Sprintf("sudo systemd-run --quiet ./kolet run %s Http", c.H.Name()))

	if err := util.WaitUntilReady(60*time.Second, 5*time.Second, func() (bool, error) {
		_, _, err := srv.SSH(fmt.Sprintf("curl %s:8080", srv.PrivateIP()))
		return err == nil, nil
	}); err != nil {
		c.Fatal("timed out waiting for http server to become active")
	}
}

// END COPIED STUFF
