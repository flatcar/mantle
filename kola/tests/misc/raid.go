// Copyright 2017 CoreOS, Inc.
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
	"fmt"

	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/kola/tests/util"
	tutil "github.com/flatcar/mantle/kola/tests/util"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
)

const (
	IgnitionConfigRootRaid = `variant: flatcar
version: 1.0.0
storage:
  disks:
    - device: /dev/disk/by-id/coreos-boot-disk
      partitions:
        - number: 9
          wipe_partition_entry: true
        - number: 9
          label: root1
          size_mib: 256
          type_guid: be9067b9-ea49-4f15-b4f6-f36f8c9e1818
        - number: 10
          label: root2
          size_mib: 256
          type_guid: be9067b9-ea49-4f15-b4f6-f36f8c9e1818
  raid:
    - name: rootarray
      level: "{{ .RaidLevel }}"
      devices:
        - /dev/disk/by-partlabel/root1
        - /dev/disk/by-partlabel/root2
  filesystems:
    - path: /
      device: /dev/md/rootarray
      format: ext4
      label: ROOT
      wipe_filesystem: true
`

	IgnitionConfigDataRaid = `{
  "ignition": {
    "config": {},
    "security": {
      "tls": {}
    },
    "timeouts": {},
    "version": "2.3.0"
  },
  "networkd": {},
  "storage": {
    "disks": [
      {
        "device": "/dev/disk/by-partlabel/OEM-CONFIG"
      },
      {
        "device": "/dev/disk/by-partlabel/USR-B"
      }
    ],
    "filesystems": [
      {
        "name": "DATA",
        "mount": {
          "device": "/dev/md/DATA",
          "format": "ext4",
          "label": "DATA"
        }
      }
    ],
    "raid": [
      {
        "devices": [
          "/dev/disk/by-partlabel/OEM-CONFIG",
          "/dev/disk/by-partlabel/USR-B"
        ],
        "level": "{{ .RaidLevel }}",
        "name": "DATA"
      }
    ]
  },
  "systemd": {
    "units": [
      {
        "name": "var-lib-data.mount",
        "enabled": true,
        "contents": "[Mount]\nWhat=/dev/md/DATA\nWhere=/var/lib/data\nType=ext4\n\n[Install]\nWantedBy=local-fs.target"
      }
    ]
  }
}
`
)

var (
	raidTypes = map[string]interface{}{
		"raid0": struct{}{},
		"raid1": struct{}{},
	}
)

type raidConfig struct {
	RaidLevel string
}

func init() {
	for raidLevel, _ := range raidTypes {
		level := raidLevel

		// root partition
		templRoot, err := util.ExecTemplate(IgnitionConfigRootRaid, raidConfig{
			RaidLevel: level,
		})
		if err != nil {
			fmt.Printf("fail to execute template for %s: %v\n", level, err)
			return
		}
		userDataRoot := conf.Butane(templRoot)

		runRootOnRaid := func(c cluster.TestCluster) {
			RootOnRaid(c)
		}

		register.Register(&register.Test{
			Run:         runRootOnRaid,
			ClusterSize: 1,
			UserData:    userDataRoot,
			Name:        fmt.Sprintf("cl.disk.%s.root", raidLevel),
			Distros:     []string{"cl"},
		})

		// data partition
		templData, err := util.ExecTemplate(IgnitionConfigDataRaid, raidConfig{
			RaidLevel: level,
		})
		if err != nil {
			fmt.Printf("fail to execute template for %s: %v\n", level, err)
			return
		}
		userDataData := conf.Ignition(templData)

		runDataOnRaid := func(c cluster.TestCluster) {
			DataOnRaid(c, userDataData)
		}

		register.Register(&register.Test{
			Run:         runDataOnRaid,
			ClusterSize: 1,
			Name:        fmt.Sprintf("cl.disk.%s.data", raidLevel),
			UserData:    userDataData,
			Distros:     []string{"cl"},
			// This test is normally not related to the cloud environment
			Platforms: []string{"qemu", "qemu-unpriv"},
		})
	}
}

func RootOnRaid(c cluster.TestCluster) {
	m := c.Machines()[0]

	checkIfMountpointIsRaid(c, m, "/")

	// reboot it to make sure it comes up again
	err := m.Reboot()
	if err != nil {
		c.Fatalf("could not reboot machine: %v", err)
	}

	checkIfMountpointIsRaid(c, m, "/")
}

func DataOnRaid(c cluster.TestCluster, userData *conf.UserData) {
	m := c.Machines()[0]

	checkIfMountpointIsRaid(c, m, "/var/lib/data")

	// reboot it to make sure it comes up again
	err := m.Reboot()
	if err != nil {
		c.Fatalf("could not reboot machine: %v", err)
	}

	checkIfMountpointIsRaid(c, m, "/var/lib/data")
}

func checkIfMountpointIsRaid(c cluster.TestCluster, m platform.Machine, mountpoint string) {
	tutil.CheckMountpoint(c, m, mountpoint, func(b tutil.Blockdevice) bool { return isValidRaidType(b.Type) })
}

// isValidRaidType checks if the given type string is one of the possible
// RAID types supported by the testsuite. For example, raid0 or raid1.
func isValidRaidType(rType string) bool {
	if _, ok := raidTypes[rType]; ok {
		return true
	}
	return false
}
