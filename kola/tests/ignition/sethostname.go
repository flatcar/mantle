// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package ignition

import (
	"strings"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
)

func init() {
	// Set the hostname
	configV1 := conf.Ignition(`{
		          "ignitionVersion": 1,
		          "storage": {
		              "filesystems": [
		                  {
		                      "device": "/dev/disk/by-partlabel/ROOT",
		                      "format": "ext4",
		                      "files": [
		                          {
		                              "path": "/etc/hostname",
		                              "mode": 420,
		                              "contents": "core1"
		                          }
		                      ]
		                  }
		              ]
		          }
		      }`)
	configV2 := conf.Ignition(`{
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
		                          "source": "data:,core1"
		                      }
		                  }
		              ]
		          }
		      }`)
	configV3 := conf.Ignition(`{
		          "ignition": {
		              "version": "3.0.0"
		          },
		          "storage": {
		              "files": [
		                  {
		                      "path": "/etc/hostname",
		                      "mode": 420,
							  "overwrite": true,
		                      "contents": {
		                          "source": "data:,core1"
		                      }
		                  }
		              ]
		          }
		      }`)

	// These tests are disabled on Azure because the hostname
	// is required by the API and is overwritten via waagent.service
	// after the machine has booted.
	register.Register(&register.Test{
		Name:             "cl.ignition.v1.sethostname",
		Run:              setHostname,
		ClusterSize:      1,
		UserData:         configV1,
		Distros:          []string{"cl"},
		ExcludePlatforms: []string{"azure"},
	})
	register.Register(&register.Test{
		Name:             "coreos.ignition.sethostname",
		Run:              setHostname,
		ClusterSize:      1,
		UserData:         configV2,
		UserDataV3:       configV3,
		Distros:          []string{"cl", "fcos", "rhcos"},
		ExcludePlatforms: []string{"azure"},
	})
}

func setHostname(c cluster.TestCluster) {
	m := c.Machines()[0]

	out := c.MustSSH(m, "hostnamectl")

	if !strings.Contains(string(out), "Static hostname: core1") {
		c.Fatalf("hostname wasn't set correctly:\n%s", out)
	}
}
