// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package bpf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/util"
)

// cmdPrefix is a temporary hack to pull `bcc` tools into Flatcar
const cmdPrefix = "docker run -d --name %s -v /lib/modules:/lib/modules -v /sys/kernel/debug:/sys/kernel/debug -v /sys/fs/cgroup:/sys/fs/cgroup -v /sys/fs/bpf:/sys/fs/bpf --privileged --net host --pid host ghcr.io/flatcar/bcc %s"

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "kola/tests/bpf")
)

// Log defines the standard log format
// from Docker
// https://docs.docker.com/config/containers/logging/json-file/
type Log struct {
	Log    string `json:"log"`
	Time   string `json:"time"`
	Stream string `json:"stream"`
}

func init() {
	register.Register(&register.Test{
		Run:         execsnoopTest,
		ClusterSize: 1,
		Name:        `bpf.execsnoop`,
		Distros:     []string{"cl"},
		// required while SELinux policy is not correcly updated to support
		// `bpf` and `perfmon` permission.
		Flags: []register.Flag{register.NoEnableSelinux},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv", "azure"},
	})
}

func execsnoopTest(c cluster.TestCluster) {
	m := c.Machines()[0]
	containerName := "execsnoop"

	// filter commands with `docker ps`
	plog.Infof("running %s container", containerName)
	cmd := fmt.Sprintf(cmdPrefix, containerName, "/usr/share/bcc/tools/execsnoop -n docker -l ps")
	if _, err := c.SSH(m, cmd); err != nil {
		c.Fatalf("unable to run SSH command '%s': %v", cmd, err)
	}

	// wait for the container and the `execsnoop` command to be correctly started before
	// generating traffic.
	if err := util.Retry(10, 2*time.Second, func() error {

		// Run 'docker ps' to trigger log output. Execsnoop won't print anything, not even the header,
		// before it's been triggered for the first time.
		_ = c.MustSSH(m, "docker ps")

		// we first assert that the container is running and then the process too.
		// it's not possible to use `docker top...` command because it's the execsnoop itself who takes some time to start.
		logs, err := c.SSH(m, fmt.Sprintf("sudo cat $(docker inspect --format='{{.LogPath}}' %s)", containerName))
		if err != nil {
			return fmt.Errorf("getting running process: %w", err)
		}

		if len(logs) > 0 {
			return nil
		}

		return fmt.Errorf("no logs, the service has not started yet properly")
	}); err != nil {
		c.Fatalf("unable to get container ready: %v", err)
	}

	// generate some "traffic"
	_ = c.MustSSH(m, "docker info")
	_ = c.MustSSH(m, fmt.Sprintf("docker logs %s", containerName))
	_ = c.MustSSH(m, fmt.Sprintf("docker top %s", containerName))

	plog.Infof("getting logs from %s container", containerName)
	if err := util.Retry(10, 2*time.Second, func() error {
		logs, err := c.SSH(m, fmt.Sprintf("sudo cat $(docker inspect --format='{{.LogPath}}' %s)", containerName))
		if err != nil {
			c.Fatalf("unable to run SSH command: %v", err)
		}
		dockerLogs := bytes.Split(logs, []byte("\n"))

		// we have the headers of the table
		// then 2 lines for docker ps and the torcx call if torcx is used
		if len(dockerLogs) < 2 {
			return fmt.Errorf("Waiting for execsnoop log entries")
		}

		l := Log{}
		for _, log := range dockerLogs {

			if err := json.Unmarshal(log, &l); err != nil {
				return fmt.Errorf("Transient error unmarshalling docker log: %v", err)
			}
			if l.Stream == "stderr" {
				return fmt.Errorf("Transient error: stream should not log to 'stderr'")
			}
			if strings.Contains(l.Log, "docker info") || strings.Contains(l.Log, "docker top") || strings.Contains(l.Log, "docker logs") {
				c.Fatal("log should not contain 'docker info' or 'docker top' or 'docker logs'")
			}

			if strings.Contains(l.Log, "docker ps") {
				return nil
			}
		}
		return fmt.Errorf("Waiting for execsnoop log entries")
	}); err != nil {
		c.Fatalf("Unable to find 'docker ps' log lines in execsnoop logs: %v", err)
	}
}
