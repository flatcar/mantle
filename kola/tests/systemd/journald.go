// Copyright 2015 CoreOS, Inc.
// Copyright 2023 the Flatcar Maintainers.
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

package systemd

import (
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/util"
)

var (
	gatewayconf = conf.Ignition(`{
			   "ignition": {
			       "version": "2.0.0"
			   },
			   "storage": {
			       "files": [
				   {
				       "filesystem": "root",
				       "path": "/etc/hostname",
				       "mode": 420,
				       "contents": {
					   "source": "data:,gateway"
				       }
				   }
			       ]
			   },
			   "systemd": {
			       "units": [
				   {
				       "name": "systemd-journal-gatewayd.socket",
				       "enable": true
				   }
			       ]
			   }
		       }`)
)

func init() {
	register.Register(&register.Test{
		Run:         journalRemote,
		ClusterSize: 0,
		Name:        "systemd.journal.remote",
		Distros:     []string{"cl"},

		// Disabled on Azure because setting hostname
		// is required at the instance creation level
		ExcludePlatforms: []string{"azure"},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv"},
	})
	register.Register(&register.Test{
		Run:     journalUser,
		Name:    "systemd.journal.user",
		Distros: []string{"cl"},

		// This test is normally not related to the cloud environment
		Platforms:   []string{"qemu", "qemu-unpriv"},
		DefaultUser: "flatcar",
		ClusterSize: 1,
		MinVersion:  semver.Version{Major: 3549},
		UserData: conf.Butane(`variant: flatcar
version: 1.0.0
passwd:
  users:
    - name: flatcar
      groups:
        - systemd-journal
storage:
  directories:
    - path: /etc/systemd/user/default.target.wants
      mode: 0755
  files:
    - path: /var/lib/systemd/linger/flatcar
      mode: 0644
    - path: /etc/systemd/user/hello.service
      mode: 0644
      contents:
        inline: |
          [Unit]
          Description=A hello world unit!

          [Service]
          Type=oneshot
          ExecStart=/usr/bin/echo "Foo !"

          [Install]
          WantedBy=default.target
  links:
    - path: /etc/systemd/user/default.target.wants/hello.service
      target: /etc/systemd/user/hello.service
      hard: false`),
	})
}

func journalUser(c cluster.TestCluster) {
	if err := util.Retry(10, 2*time.Second, func() error {
		cmd := "journalctl --user"
		log, err := c.SSH(c.Machines()[0], cmd)
		if err != nil {
			return fmt.Errorf("Did not get expexted log output from '%s': %v", cmd, err)
		}

		if len(log) == 0 {
			return fmt.Errorf("Waiting for log output...")
		}

		if strings.Contains(string(log), "Foo") {
			return nil
		}

		return fmt.Errorf("Waiting for log output containing 'Foo'...")
	}); err != nil {
		c.Fatalf("Unable to find 'Foo' in user journal: %v", err)
	}
}

// JournalRemote tests that systemd-journal-remote can read log entries from
// a systemd-journal-gatewayd server.
func journalRemote(c cluster.TestCluster) {
	// start gatewayd and log a message
	gateway, err := c.NewMachine(gatewayconf)
	if err != nil {
		c.Fatalf("Cluster.NewMachine: %s", err)
	}

	// log a unique message on gatewayd machine
	msg := "supercalifragilisticexpialidocious"
	c.MustSSH(gateway, "logger "+msg)

	// spawn a machine to read from gatewayd
	collector, err := c.NewMachine(nil)
	if err != nil {
		c.Fatalf("Cluster.NewMachine: %s", err)
	}

	// collect logs from gatewayd machine
	cmd := fmt.Sprintf("sudo systemd-run --unit systemd-journal-remote-client /usr/lib/systemd/systemd-journal-remote --url http://%s:19531", gateway.PrivateIP())
	c.MustSSH(collector, cmd)

	// find the message on the collector
	journalReader := func() error {
		cmd = fmt.Sprintf("sudo journalctl _HOSTNAME=gateway -t core --file /var/log/journal/remote/remote-%s.journal", gateway.PrivateIP())
		out, err := c.SSH(collector, cmd)
		if err != nil {
			return fmt.Errorf("journalctl: %v: %v", out, err)
		}

		if !strings.Contains(string(out), msg) {
			return fmt.Errorf("journal missing entry: expected %q got %q", msg, out)
		}

		return nil
	}

	if err := util.Retry(5, 2*time.Second, journalReader); err != nil {
		c.Fatal(err)
	}
}
