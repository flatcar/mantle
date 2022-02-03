// Copyright 2016 CoreOS, Inc.
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

package ignition

import (
	"fmt"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
)

const (
	targetUUID   = "9aa5237a-ab6b-458b-a7e8-f25e2baef1a3"
	targetVfatID = "1A37-8FA3"
	swapFileName = "/var/vm/swapfile1"
	swapSize     = "1024M"
)

func init() {
	// Reformat the root as btrfs
	btrfsConfigV1 := conf.Ignition(`{
		               "ignitionVersion": 1,
		               "storage": {
		                   "filesystems": [
		                       {
		                           "device": "/dev/disk/by-partlabel/ROOT",
		                           "format": "btrfs",
		                           "create": {
		                               "force": true,
		                               "options": [
		                                   "--label=ROOT",
		                                   "--uuid=` + targetUUID + `"
		                               ]
		                           }
		                       }
		                   ]
		               }
		           }`)
	btrfsConfigV2 := conf.Ignition(`{
		               "ignition": {
		                   "version": "2.0.0"
		               },
		               "storage": {
		                   "filesystems": [
		                       {
		                           "mount": {
		                               "device": "/dev/disk/by-label/ROOT",
		                               "format": "btrfs",
		                               "create": {
		                                   "force": true,
		                                   "options": [
		                                       "--label=ROOT",
		                                       "--uuid=` + targetUUID + `"
		                                   ]
		                               }
		                           }
		                       }
		                   ]
		               }
		           }`)
	register.Register(&register.Test{
		Name:        "cl.ignition.v1.btrfsroot",
		Run:         btrfsRoot,
		ClusterSize: 1,
		UserData:    btrfsConfigV1,
		Distros:     []string{"cl"},
	})
	register.Register(&register.Test{
		Name:        "cl.ignition.v2.btrfsroot",
		Run:         btrfsRoot,
		ClusterSize: 1,
		UserData:    btrfsConfigV2,
		Distros:     []string{"cl"},
	})

	// Reformat the root as xfs
	xfsConfigV1 := conf.Ignition(`{
		             "ignitionVersion": 1,
		             "storage": {
		                 "filesystems": [
		                     {
		                         "device": "/dev/disk/by-partlabel/ROOT",
		                         "format": "xfs",
		                         "create": {
		                             "force": true,
		                             "options": [
		                                 "-L", "ROOT",
		                                 "-m", "uuid=` + targetUUID + `"
		                             ]
		                         }
		                     }
		                 ]
		             }
		         }`)
	xfsConfigV2 := conf.Ignition(`{
		             "ignition": {
		                 "version": "2.0.0"
		             },
		             "storage": {
		                 "filesystems": [
		                     {
		                         "mount": {
		                             "device": "/dev/disk/by-partlabel/ROOT",
		                             "format": "xfs",
		                             "create": {
		                                 "force": true,
		                                 "options": [
		                                     "-L", "ROOT",
		                                     "-m", "uuid=` + targetUUID + `"
		                                 ]
		                             }
		                         }
		                     }
		                 ]
		             }
		         }`)
	register.Register(&register.Test{
		Name:        "cl.ignition.v1.xfsroot",
		Run:         xfsRoot,
		ClusterSize: 1,
		UserData:    xfsConfigV1,
		Distros:     []string{"cl"},
	})
	register.Register(&register.Test{
		Name:        "cl.ignition.v2.xfsroot",
		Run:         xfsRoot,
		ClusterSize: 1,
		UserData:    xfsConfigV2,
		Distros:     []string{"cl"},
	})

	// Reformat the root as ext4
	ext4ConfigV1 := conf.Ignition(`{
		             "ignitionVersion": 1,
		             "storage": {
		                 "filesystems": [
		                     {
		                         "device": "/dev/disk/by-partlabel/ROOT",
		                         "format": "ext4",
		                         "create": {
		                             "force": true,
		                             "options": [
		                                 "-L", "ROOT",
		                                 "-U", "` + targetUUID + `"
		                             ]
		                         }
		                     }
		                 ]
		             }
		         }`)
	ext4ConfigV2 := conf.Ignition(`{
		             "ignition": {
		                 "version": "2.0.0"
		             },
		             "storage": {
		                 "filesystems": [
		                     {
		                         "mount": {
		                             "device": "/dev/disk/by-partlabel/ROOT",
		                             "format": "ext4",
		                             "create": {
		                                 "force": true,
		                                 "options": [
		                                     "-L", "ROOT",
		                                     "-U", "` + targetUUID + `"
		                                 ]
		                             }
		                         }
		                     }
		                 ]
		             }
		         }`)
	register.Register(&register.Test{
		Name:        "cl.ignition.v1.ext4root",
		Run:         ext4Root,
		ClusterSize: 1,
		UserData:    ext4ConfigV1,
		Distros:     []string{"cl"},
	})
	register.Register(&register.Test{
		Name:        "cl.ignition.v2.ext4root",
		Run:         ext4Root,
		ClusterSize: 1,
		UserData:    ext4ConfigV2,
		Distros:     []string{"cl"},
	})
	register.Register(&register.Test{
		Name:        "cl.ignition.v2_1.ext4checkexisting",
		Run:         ext4CheckExisting,
		ClusterSize: 1,
		Distros:     []string{"cl"},
	})

	vfatConfigV2_1 := conf.Ignition(`{
			             "ignition": {
			                 "version": "2.1.0"
			             },
			             "storage": {
			                 "filesystems": [
			                     {
			                         "mount": {
			                             "device": "/dev/disk/by-partlabel/USR-B",
			                             "format": "vfat",
			                             "wipeFilesystem": true,
			                             "label": "USR-B",
			                             "uuid": "` + targetVfatID + `"
			                         }
			                     }
			                 ]
			             }
			         }`)
	register.Register(&register.Test{
		Name:        "cl.ignition.v2_1.vfat",
		Run:         vfatUsrB,
		ClusterSize: 1,
		UserData:    vfatConfigV2_1,
		Distros:     []string{"cl"},
	})

	swapConfigV2_1 := conf.Ignition(`{
			             "ignition": {
			                 "version": "2.1.0"
			             },
			             "storage": {
			                 "filesystems": [
			                     {
			                         "mount": {
			                             "device": "/dev/disk/by-partlabel/USR-B",
			                             "format": "swap",
			                             "wipeFilesystem": true,
			                             "label": "USR-B",
			                             "uuid": "` + targetUUID + `"
			                         }
			                     }
			                 ]
			             }
			         }`)
	register.Register(&register.Test{
		Name:        "cl.ignition.v2_1.swap",
		Run:         swapUsrB,
		ClusterSize: 1,
		UserData:    swapConfigV2_1,
		Distros:     []string{"cl"},
	})

	swapActivation := conf.ContainerLinuxConfig(fmt.Sprintf(`storage:
  files:
  - path: /etc/sysctl.d/80-swappiness.conf
    filesystem: root
    mode: 0644
    contents:
      inline: "vm.swappiness=10"

systemd:
  units:
    - name: var-vm-swapfile1.swap
      enabled: true
      contents: |
        [Unit]
        Description=Turn on swap
        Requires=create-swapfile.service
        After=create-swapfile.service

        [Swap]
        What=%[1]s

        [Install]
        WantedBy=multi-user.target
    - name: create-swapfile.service
      contents: |
        [Unit]
        Description=Create a swapfile
        RequiresMountsFor=/var

        [Service]
        Type=oneshot
        ExecStart=/usr/bin/mkdir -p /var/vm
        ExecStart=/usr/bin/fallocate -l %[2]s %[1]s
        ExecStart=/usr/bin/chmod 600 %[1]s
        ExecStart=/usr/sbin/mkswap %[1]s
        RemainAfterExit=true`, swapFileName, swapSize))

	register.Register(&register.Test{
		Name:        "cl.swap_activation",
		Run:         testSwapActivation,
		ClusterSize: 1,
		UserData:    swapActivation,
		Distros:     []string{"cl"},
		MinVersion:  semver.Version{Major: 3033},
	})
}

var ext4NoClobberV2_1 = conf.Ignition(`{
		            "ignition": {
		                "version": "2.1.0"
		            },
		            "storage": {
		                "filesystems": [
		                    {
		                        "mount": {
		                            "device": "/dev/disk/by-partlabel/ROOT",
		                            "format": "ext4",
		                            "label": "ROOT"
		                        }
		                    }
		                ]
		            }
		        }`)

func btrfsRoot(c cluster.TestCluster) {
	testRoot(c, "btrfs")
}

func xfsRoot(c cluster.TestCluster) {
	testRoot(c, "xfs")
}

func ext4Root(c cluster.TestCluster) {
	testRoot(c, "ext4")
}

func vfatUsrB(c cluster.TestCluster) {
	testFormatted(c, "vfat", "USR-B")
}

func swapUsrB(c cluster.TestCluster) {
	testFormatted(c, "swap", "USR-B")
}

func testFormatted(c cluster.TestCluster, fs, label string) {
	m := c.Machines()[0]

	out := c.MustSSH(m, "sudo blkid -s UUID -o value /dev/disk/by-label/"+label)
	target := targetUUID
	if fs == "vfat" {
		target = targetVfatID
	}
	if strings.TrimRight(string(out), "\n") != target {
		c.Fatalf("filesystem wasn't correctly formatted:\n%s", out)
	}

	out = c.MustSSH(m, "sudo blkid -s TYPE -o value /dev/disk/by-label/"+label)
	if strings.TrimRight(string(out), "\n") != fs {
		c.Fatalf("filesystem has incorrect type:\n%s", out)
	}
}

func testRoot(c cluster.TestCluster, fs string) {
	m := c.Machines()[0]

	testFormatted(c, fs, "ROOT")

	out := c.MustSSH(m, "findmnt --noheadings --output FSTYPE --target /")
	if string(out) != fs {
		c.Fatalf("root wasn't correctly reformatted:\n%s", out)
	}
}

func ext4CheckExisting(c cluster.TestCluster) {
	m1 := c.Machines()[0]

	// Get root filesystem UUID
	out := c.MustSSH(m1, "sudo blkid /dev/disk/by-partlabel/ROOT -s UUID -o value")
	uuid1 := strings.TrimRight(string(out), "\n")

	// Start a machine with config that conditionally recreates the FS
	m2, err := c.NewMachine(ext4NoClobberV2_1)
	if err != nil {
		c.Fatalf("Couldn't start machine: %v", err)
	}

	// Verify UUID hasn't changed
	out = c.MustSSH(m2, "sudo blkid /dev/disk/by-partlabel/ROOT -s UUID -o value")
	uuid2 := strings.TrimRight(string(out), "\n")
	if uuid1 != uuid2 {
		c.Fatalf("Filesystem was reformatted: %s != %s", uuid1, uuid2)
	}
}

func testSwapActivation(c cluster.TestCluster) {
	m := c.Machines()[0]

	out := c.MustSSH(m, "swapon --show=NAME,SIZE --noheadings")
	swap := string(out)

	if swap == "" {
		c.Fatalf("swap is not enabled while it should be")
	}

	columns := strings.Split(swap, " ")
	if len(columns) < 2 {
		c.Fatalf("swap output should be at least: name size")
	}

	name := columns[0]
	size := columns[1]

	if name != swapFileName {
		c.Fatalf("swap's name should be: %s, got %s", swapFileName, swap)
	}

	if size != swapSize {
		c.Fatalf("swap's size should be: %s, got %s", swapSize, size)
	}
}
