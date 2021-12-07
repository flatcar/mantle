// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package systemd

import (
	"fmt"
	"strings"
	"time"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
	"github.com/flatcar-linux/mantle/util"
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
		// qemu-unpriv machines cannot communicate
		ExcludePlatforms: []string{"azure", "qemu-unpriv"},
	})
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
