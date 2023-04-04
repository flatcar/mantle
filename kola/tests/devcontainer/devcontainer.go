// Copyright 2022 The Flatcar Maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package devcontainer

import (
	"fmt"

	"github.com/coreos/go-semver/semver"

	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/kola/tests/util"
	"github.com/flatcar/mantle/platform/conf"
)

// Both template parameters may contain @ARCH@ and @VERSION@
// placeholders, which will be substituted by real values at test run
// time.
type scriptTemplateParameters struct {
	BinhostURLTemplate string
}

type configTemplateParameters struct {
	DevContainerScriptBase64Contents string
	DownloadLibraryBase64Contents    string
	MainScriptBase64Contents         string
}

var (
	devContainerScript = util.TrimLeftSpace(`
#!/bin/bash

set -euo pipefail

set -x

source /usr/share/coreos/release

if [[ "${EXPECTED_VERSION}" != "${FLATCAR_RELEASE_VERSION}" ]]; then
        echo "Version mismatch, expected '${EXPECTED_VERSION}', got '${FLATCAR_RELEASE_VERSION}'."
        exit 1
fi

export PORTAGE_BINHOST="${PORTAGE_BINHOST}"
emerge-gitclone
emerge --getbinpkg --verbose coreos-sources
zcat /proc/config.gz >/usr/src/linux/.config
exec make -C /usr/src/linux "-j$(nproc)" modules_prepare V=1
`)

	scriptPrologTemplate = util.TrimLeftSpace(`
#!/bin/bash

set -x

set -euo pipefail

source /home/core/download-library.sh

download_dev_container_image flatcar_developer_container.bin

source /usr/share/coreos/release

ARCH="${FLATCAR_RELEASE_BOARD/-usr/}"
VERSION="${FLATCAR_RELEASE_VERSION}"

# PORTAGE_BINHOST and EXPECTED_VERSION are meant to be propagated to
# the dev container as environment variables.
PORTAGE_BINHOST=$(process_template '{{ .BinhostURLTemplate }}' "${ARCH}" "${VERSION}")
EXPECTED_VERSION="${FLATCAR_RELEASE_VERSION}"

# These directories (USR_SRC_DIR and VAR_TMP_DIR) are meant to be used
# by the dev container to store files generated during emerging kernel
# sources and making the modules_prepare job.
#
# Previously tmpfs was used, but under qemu we might not have enough
# memory.
workdir="${PWD}/dev-container-workdir-${RANDOM}"
USR_SRC_DIR="${workdir}/src"
VAR_TMP_DIR="${workdir}/tmp"
mkdir -p "${USR_SRC_DIR}" "${VAR_TMP_DIR}"
`)

	systemdNspawnScriptBody = util.TrimLeftSpace(`
sudo systemd-nspawn \
        --console=pipe \
        --setenv=PORTAGE_BINHOST="${PORTAGE_BINHOST}" \
        --setenv=EXPECTED_VERSION="${EXPECTED_VERSION}" \
        --bind-ro=/lib/modules \
        --bind-ro=/home/core/dev-container-script \
        --bind="${USR_SRC_DIR}:/usr/src" \
        --bind="${VAR_TMP_DIR}:/var/tmp" \
        --image=flatcar_developer_container.bin \
        --machine=flatcar-developer-container \
        /bin/bash /home/core/dev-container-script
`)

	dockerScriptBody = util.TrimLeftSpace(`
# TODO: It would much much better if we provided dev-container as a
# docker image on ghcr.io.

offset=$(parted flatcar_developer_container.bin unit b print 2>/dev/null | grep 'Start' --after-context=1 | tail --lines=1 | awk '{ print $2 }' | head --bytes=-2)
mkdir root
sudo mount -o loop,ro,offset="${offset}" flatcar_developer_container.bin root
sudo tar -C root -czf dev-container-image.tar.gz .
sudo umount root
rm -f flatcar_developer_container.bin
docker import dev-container-image.tar.gz dev-container:42
sudo rm -f dev-container-image.tar.gz
# We need to restore the SELinux context of the script, otherwise we
# will get permission denied errors when trying to invoke the script
# inside the dev container.
restorecon /home/core/dev-container-script

docker run \
        --log-driver=journald \
        --env PORTAGE_BINHOST="${PORTAGE_BINHOST}" \
        --env EXPECTED_VERSION="${EXPECTED_VERSION}" \
        --mount type=bind,source=/lib/modules,target=/lib/modules,readonly=true \
        --mount type=bind,source=/home/core/dev-container-script,target=/home/core/dev-container-script,readonly=true \
        --mount type=bind,source="${USR_SRC_DIR}",target=/usr/src \
        --mount type=bind,source="${VAR_TMP_DIR}",target=/var/tmp \
        dev-container:42 \
        /bin/bash /home/core/dev-container-script
`)

	configTemplate = util.TrimLeftSpace(`
storage:
  files:
    - path: /home/core/dev-container-script
      filesystem: root
      mode: 0755
      contents:
        remote:
          url: "data:text/plain;base64,{{ .DevContainerScriptBase64Contents }}"
      user:
        name: core
      group:
        name: core
    - path: /home/core/download-library.sh
      filesystem: root
      mode: 0644
      contents:
        remote:
          url: "data:text/plain;base64,{{ .DownloadLibraryBase64Contents }}"
      user:
        name: core
      group:
        name: core
    - path: /home/core/main-script
      filesystem: root
      mode: 0755
      contents:
        remote:
          url: "data:text/plain;base64,{{ .MainScriptBase64Contents }}"
      user:
        name: core
      group:
        name: core
`)
)

func init() {
	register.Register(&register.Test{
		Name:        "devcontainer.systemd-nspawn",
		Run:         withSystemdNspawn,
		ClusterSize: 0,
		// This test is normally not related to the cloud environment
		Platforms:  []string{"qemu", "qemu-unpriv"},
		Distros:    []string{"cl"},
		MinVersion: semver.Version{Major: 2592},
		NativeFuncs: map[string]func() error{
			"Http": util.Serve,
		},
	})
	register.Register(&register.Test{
		Name:        "devcontainer.docker",
		Run:         withDocker,
		ClusterSize: 0,
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv"},
		Distros:   []string{"cl"},
		// TODO: Revisit this flag when updating SELinux policies.
		Flags: []register.Flag{register.NoEnableSelinux},
		NativeFuncs: map[string]func() error{
			"Http": util.Serve,
		},
	})
}

func withSystemdNspawn(c cluster.TestCluster) {
	runDevContainerTest(c, systemdNspawnScriptBody)
}

func withDocker(c cluster.TestCluster) {
	runDevContainerTest(c, dockerScriptBody)
}

func runDevContainerTest(c cluster.TestCluster, scriptBody string) {
	downloadLibrary, err := util.DevContainerDownloadLibrary()
	if err != nil {
		c.Fatalf("creating a dev container download script failed: %v", err)
	}

	scriptParameters := scriptTemplateParameters{
		BinhostURLTemplate: kola.DevcontainerBinhostURL,
	}

	userdata, err := prepareUserData(scriptParameters, scriptBody, downloadLibrary)
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

	if _, err := c.SSH(machine, "/home/core/main-script"); err != nil {
		c.Fatalf("main script failed: %v", err)
	}
}

func prepareUserData(scriptParameters scriptTemplateParameters, scriptBody, downloadLibrary string) (*conf.UserData, error) {
	prolog, err := util.ExecNamedTemplate(scriptPrologTemplate, "script prolog", scriptParameters)
	if err != nil {
		return nil, err
	}
	mainScript := fmt.Sprintf("%s%s", prolog, scriptBody)
	mainScriptBase64 := util.ToBase64(mainScript)
	downloadLibraryBase64 := util.ToBase64(downloadLibrary)
	devContainerScriptBase64 := util.ToBase64(devContainerScript)
	configParameters := configTemplateParameters{
		DevContainerScriptBase64Contents: devContainerScriptBase64,
		DownloadLibraryBase64Contents:    downloadLibraryBase64,
		MainScriptBase64Contents:         mainScriptBase64,
	}
	config, err := util.ExecNamedTemplate(configTemplate, "cloud config", configParameters)
	if err != nil {
		return nil, err
	}
	return conf.ContainerLinuxConfig(config), nil
}
