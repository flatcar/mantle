// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/platform/local"
	mutil "github.com/flatcar/mantle/util"
)

type (
	downloadLibraryParameters struct {
		ImageDirectoryURLTemplate string
	}
)

var (
	downloadLibraryTemplate = TrimLeftSpace(`
#!/bin/bash

# Takes a template, architecture and version and prints the template
# with all instances of @ARCH@ replaced with the passed architecture
# and all instances of @VERSION@ replaced with the passed version.
#
# Example:
# url=$(process_template 'https://example.com/@ARCH@/@VERSION@/image.bz2' 'amd64' '1.2.3')
function process_template {
        local template="${1}"; shift
        local arch="${1}"; shift
        local version="${1}"; shift
        local result="${template}"

        result="${result//@ARCH@/${arch}}"
        result="${result//@VERSION@/${version}}"

        echo "${result}"
}

# Downloads the developer container image and decompresses it. The
# result is stored in the passed path.
#
# Example:
# download_dev_container_image flatcar_devcontainer.bin
function download_dev_container_image {
        local output_bin="${1}"; shift

        local arch version image_url bzip2cat

        arch=$(source /usr/share/flatcar/release; echo "${FLATCAR_RELEASE_BOARD/-usr/}")
        version=$(source /usr/share/flatcar/release; echo "${FLATCAR_RELEASE_VERSION}")
        image_url=$(process_template '{{ .ImageDirectoryURLTemplate }}/flatcar_developer_container.bin.bz2' "${arch}" "${version}")

        echo "Fetching developer container from ${image_url}"
        # Stolen from copy_from_buildcache in ci_automation_common.sh. Not
        # using --output-dir option as this seems to be quite a new addition
        # and curl on older version of Flatcar does not understand it.
        curl --fail --silent --show-error --location --retry-delay 1 --retry 60 \
                --retry-connrefused --retry-max-time 60 --connect-timeout 20 \
                --remote-name "${image_url}"

        bzip2cat=bzcat
        if command -v lbzcat; then
                bzip2cat=lbzcat
        fi

        # The image file takes over 6Gb after normal unpacking, but a lot of
        # it is just zeros. Use cp --sparse=always to avoid unnecessary disk
        # space waste. Especially that we may not have 6Gb of disk space
        # available.
        cp --sparse=always <("${bzip2cat}" flatcar_developer_container.bin.bz2) "${output_bin}"
}
`)
)

// TrimLeftSpace trims leading whitespace. Useful for defining script
// as string variables, like:
//
// var (
//
//	script = util.TrimLeftSpace(`
//
// #!/bin/bash
// …
// `)
func TrimLeftSpace(contents string) string {
	return strings.TrimLeftFunc(contents, unicode.IsSpace)
}

// DevContainerDownloadLibrary generates a bash library that could be
// sourced on the machine to import two functions: process_template
// and download_dev_container_image.
//
// The source of the developer container image can be specified using
// --devcontainer-url or --devcontainer-file options.
func DevContainerDownloadLibrary() (string, error) {
	devcontainerURL := kola.DevcontainerURL
	if kola.DevcontainerFile != "" {
		// This URL is deterministic as it runs on the started machine.
		devcontainerURL = "http://localhost:8080"
	}
	libraryParameters := downloadLibraryParameters{
		ImageDirectoryURLTemplate: devcontainerURL,
	}
	return ExecNamedTemplate(downloadLibraryTemplate, "download library", libraryParameters)
}

// NewMachineWithLargeDisk creates a new machine on the passed qemu or
// qemu-unpriv cluster. The extraSize parameter is a string describing
// size, like "5G".
func NewMachineWithLargeDisk(c cluster.TestCluster, extraSize string, userData *conf.UserData) (platform.Machine, error) {
	options := platform.MachineOptions{
		ExtraPrimaryDiskSize: extraSize,
	}
	if pc, ok := c.Cluster.(platform.CreateWithOptions); ok {
		return pc.NewMachineWithOptions(userData, options)
	} else {
		return nil, fmt.Errorf("platform %s does not support creating machines with options", c.Cluster.Platform())
	}
}

// Serve is a function that could be used as a native function for
// running a simple HTTP server inside cluster machines.
func Serve() error {
	httpServer := local.SimpleHTTP{}
	return httpServer.Serve()
}

// ConfigureDevContainerHTTPServer sets up the local HTTP server to
// provide the compressed developer container image if such is
// available through --devcontainer-file.
func ConfigureDevContainerHTTPServer(c cluster.TestCluster, srv platform.Machine) error {
	if kola.DevcontainerFile == "" {
		// Not using local file as a source of developer
		// container, so no need to configure the local HTTP
		// server.
		return nil
	}
	// Manually copy Kolet on the host, as the initial size cluster may be 0.
	if err := kola.UploadKolet(c, strings.SplitN(kola.QEMUOptions.Board, "-", 2)[0]); err != nil {
		return fmt.Errorf("uploading kolet to machine: %w", err)
	}

	in, err := os.Open(kola.DevcontainerFile)
	if err != nil {
		return fmt.Errorf("opening dev container file: %w", err)
	}

	defer in.Close()

	if err := platform.InstallFile(in, srv, "/var/www/flatcar_developer_container.bin.bz2"); err != nil {
		return fmt.Errorf("copying dev container to HTTP server: %w", err)
	}

	if _, err := c.SSH(srv, fmt.Sprintf("sudo systemd-run --quiet ./kolet run %s Http", c.H.Name())); err != nil {
		return err
	}

	if err := mutil.WaitUntilReady(60*time.Second, 5*time.Second, func() (bool, error) {
		_, _, err := srv.SSH(fmt.Sprintf("curl %s:8080", srv.PrivateIP()))
		return err == nil, nil
	}); err != nil {
		return fmt.Errorf("timed out waiting for http server to become active")
	}
	return nil
}

func ToBase64(input string) string {
	return base64.StdEncoding.EncodeToString(([]byte)(input))
}
