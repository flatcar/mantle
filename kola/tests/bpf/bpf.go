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
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/util"
)

// cmdPrefix is a temporary hack to pull `bcc` tools into Flatcar
const cmdPrefix = "docker run -d --name %s -v /lib/modules:/lib/modules -v /sys/kernel/debug:/sys/kernel/debug -v /sys/fs/cgroup:/sys/fs/cgroup -v /sys/fs/bpf:/sys/fs/bpf --privileged --net host --pid host quay.io/iovisor/bcc %s"

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "kola/tests/bpf")
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
		// exclude `arm64` while `quay.io/iovisor/bcc` does not have `arm64` support.
		Architectures: []string{"amd64"},
	})
}

func execsnoopTest(c cluster.TestCluster) {
	c.Run("filter name and args", func(c cluster.TestCluster) {
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
		if err := util.Retry(5, 2*time.Second, func() error {
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
		logs, err := c.SSH(m, fmt.Sprintf("sudo cat $(docker inspect --format='{{.LogPath}}' %s)", containerName))
		if err != nil {
			c.Fatalf("unable to run SSH command: %v", err)
		}

		l := Log{}
		dockerLogs := bytes.Split(logs, []byte("\n"))
		if len(dockerLogs) != 3 {
			// we have the headers of the table
			// then 2 lines for docker ps and the torcx call
			c.Fatalf("docker logs should hold 3 values. Got: %d", len(dockerLogs))
		}

		execFound := false
		for _, log := range dockerLogs {
			if len(log) == 0 {
				continue
			}

			if err := json.Unmarshal(log, &l); err != nil {
				c.Fatalf("unable to unmarshal log: %v", err)
			}
			plog.Infof("handling log %v", l)

			if l.Stream == "stderr" {
				c.Fatal("stream should not log to 'stderr'")
			}

			if strings.Contains(l.Log, "docker info") || strings.Contains(l.Log, "docker top") || strings.Contains(l.Log, "docker logs") {
				c.Fatal("log should not contain 'docker info' or 'docker top' or 'docker logs'")
			}

			if strings.Contains(l.Log, "docker ps") {
				execFound = true
			}
		}

		if !execFound {
			c.Fatal("did not get 'docker ps' in the logs")
		}
	})
}
