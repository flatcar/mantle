// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package bpf

import (
	"fmt"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform/conf"
)

var igVer = "v0.50.0"

func init() {
	register.Register(&register.Test{
		Run:        igTest,
		Name:       `bpf.ig`,
		Distros:    []string{"cl"},
		MinVersion: semver.Version{Major: 4081},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv", "azure"},
		// Required while SELinux policy is not correctly updated to support
		// `bpf` and `perfmon` permissions.
		Flags: []register.Flag{register.NoEnableSelinux},
	})
}

func igTest(c cluster.TestCluster) {
	arch := strings.SplitN(kola.QEMUOptions.Board, "-", 2)[0]
	if arch == "amd64" {
		arch = "x86-64"
	}

	c.Run("ig", func(c cluster.TestCluster) {
		node, err := c.NewMachine(conf.Butane(fmt.Sprintf(`
            variant: flatcar
            version: 1.0.0
            storage:
              files:
                - path: /etc/extensions/ig.raw
                  mode: 0644
                  contents:
                    source: https://extensions.flatcar.org/extensions/ig-%s-%s.raw
        `, igVer, arch)))
		if err != nil {
			c.Fatalf("creating node: %v", err)
		}

		// The following test that ig tracing and filtering works when applied
		// to the host. ig runs in the foreground, but it can take a few seconds
		// to be ready, even after prefetching the gadget. To avoid flakiness,
		// ig is put in the background and grep is used to wait for the
		// "running" debug message. coproc doesn't handle stderr, so stderr is
		// redirected to stdout and the real stdout is redirected to a file for
		// later analysis. The timeout against grep ensures that we don't wait
		// for "running" forever. The gadget is prefetched with --help so that
		// the download does not count against the timeout. The trap prevents ig
		// from keeping the script alive when an unexpected error occurs.

		if _, err := c.SSH(node, fmt.Sprintf(`
			set -ex
			sudo ig run trace_exec:%[1]s --help
			coproc IG { sudo ig run trace_exec:%[1]s --host --filter 'proc.comm=docker,args~ps' --output json --verbose 2>&1 > ig.json; }
			trap 'kill %%%%' ERR
			timeout 30 grep -F -m1 'running...' <&${IG[0]}
			docker info
			docker ps
			docker images
			kill %%%%
			wait
			jq -s -e '.[] | select(.args == "/usr/bin/docker\u00a0ps")' ig.json
			jq -s -e 'isempty(.[] | select(.args == "/usr/bin/docker\u00a0info"))' ig.json
			jq -s -e 'isempty(.[] | select(.args == "/usr/bin/docker\u00a0images"))' ig.json
		`, igVer)); err != nil {
			c.Fatalf("ig run trace_exec did not behave as expected: %v", err)
		}

		if _, err := c.SSH(node, fmt.Sprintf(`
			set -ex
			sudo ig run trace_dns:%[1]s --help
			coproc IG { sudo ig run trace_dns:%[1]s --host --filter 'name=flatcar.org.' --output json --verbose 2>&1 > ig.json; }
			trap 'kill %%%%' ERR
			timeout 30 grep -F -m1 'running...' <&${IG[0]}
			dig kinvolk.io
			dig flatcar.org
			dig stable.release.flatcar-linux.net
			kill %%%%
			wait
			jq -s -e '.[] | select(.name == "flatcar.org.")' ig.json
			jq -s -e 'isempty(.[] | select(.name == "kinvolk.io."))' ig.json
			jq -s -e 'isempty(.[] | select(.name == "stable.release.flatcar-linux.net."))' ig.json
		`, igVer)); err != nil {
			c.Fatalf("ig run trace_dns did not behave as expected: %v", err)
		}
	})
}
