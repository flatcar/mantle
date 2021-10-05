// Copyright 2015 CoreOS, Inc.
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

package misc

import (
	"bytes"
	"fmt"
	"time"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/util"
)

var timesyncdMsgs = [][]byte{
	[]byte(`Status: "Synchronized to time server 10.0.0.1:123 (10.0.0.1)."`),                    // systemd < 241
	[]byte(`Status: "Synchronized to time server for the first time 10.0.0.1:123 (10.0.0.1)."`), // systemd >= 241
	[]byte(`Status: "Initial synchronization to time server 10.0.0.1:123 (10.0.0.1)."`),         // systemd >= 243
}

func init() {
	register.Register(&register.Test{
		Run:              NTP,
		ClusterSize:      0,
		Name:             "linux.ntp",
		Platforms:        []string{"qemu"},
		ExcludePlatforms: []string{"qemu-unpriv"},
		Distros:          []string{"cl"},
	})
}

// Test that timesyncd starts using the local NTP server
func NTP(c cluster.TestCluster) {
	m, err := c.NewMachine(nil)
	if err != nil {
		c.Fatalf("Cluster.NewMachine: %s", err)
	}

	out := c.MustSSH(m, "networkctl status eth0")
	if !bytes.Contains(out, []byte("NTP: 10.0.0.1")) {
		c.Fatalf("Bad network config:\n%s", out)
	}

	checkTimeSyncdMsgs := func(in []byte) bool {
		for _, m := range timesyncdMsgs {
			if bytes.Contains(in, m) {
				return true
			}
		}
		return false
	}

	checker := func() error {
		out, err = c.SSH(m, "systemctl status systemd-timesyncd.service")
		if err != nil {
			return fmt.Errorf("systemctl: %v", err)
		}

		if !checkTimeSyncdMsgs(out) {
			return fmt.Errorf("unexpected systemd-timesyncd status: %q", out)
		}

		return nil
	}

	if err = util.Retry(60, 1*time.Second, checker); err != nil {
		c.Fatal(err)
	}
}
