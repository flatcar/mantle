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

package ignition

import (
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
)

func init() {
	register.Register(&register.Test{
		Name:        "cl.ignition.v1.once",
		Run:         runsOnce,
		ClusterSize: 1,
		UserData: conf.Ignition(`{
                             "ignitionVersion": 1,
                             "storage": {
                               "filesystems": [
                                 {
                                   "device": "/dev/disk/by-partlabel/ROOT",
                                   "format": "ext4",
                                   "files": [
                                     {
                                       "path": "/etc/ignition-ran",
                                       "contents": "Ignition ran.",
                                       "mode": 420
                                     }
                                   ]
                                 }
                               ]
                             }
                           }`),
		Distros: []string{"cl"},
		// We can run the coreos.ignition.once test on all cloud environments instead
		Platforms: []string{"qemu", "qemu-unpriv"},
	})
	register.Register(&register.Test{
		Name:        "coreos.ignition.once",
		Run:         runsOnce,
		ClusterSize: 1,
		UserData: conf.Ignition(`{
                             "ignition": { "version": "2.0.0" },
                             "storage": {
                               "files": [
                                 {
                                   "filesystem": "root",
                                   "path": "/etc/ignition-ran",
                                   "contents": {
                                     "source": "data:,Ignition%20ran."
                                   },
                                   "mode": 420
                                 }
                               ]
                             }
                           }`),
		UserDataV3: conf.Ignition(`{
                             "ignition": { "version": "3.0.0" },
                             "storage": {
                               "files": [
                                 {
                                   "path": "/etc/ignition-ran",
                                   "contents": {
                                     "source": "data:,Ignition%20ran."
                                   },
                                   "mode": 420
                                 }
                               ]
                             }
                           }`),
		Distros: []string{"cl", "fcos", "rhcos"},
	})
	register.Register(&register.Test{
		Name:        "cl.ignition.translation",
		Run:         testTranslation,
		ClusterSize: 1,
		// Important: the link's path shares a prefix with the
		// file's path and should pass the ign-converter sanity
		// check for not using a link dir in the file path
		UserData: conf.Ignition(`{
                             "ignition": { "version": "2.3.0" },
                             "networkd": {
                               "units": [
                                 {
                                   "name": "00-dummy.netdev",
                                   "contents": "[NetDev]\nName=kola\nKind=dummy"
                                 },
                                 {
                                   "name": "00-dummy.network",
                                   "contents": "[Match]\nType=!vlan bond bridge\nName=kola\n\n[Network]\nAddress=10.0.2.1/24"
                                 }
                               ]
                             },
                             "storage": {
                               "files": [
                                 {
                                   "filesystem": "root",
                                   "path": "/testdir/helloworld",
                                   "contents": {
                                     "source": "data:,"
                                   },
                                   "mode": 420
                                 }
                               ],
                               "links": [
                                 {
                                   "filesystem": "root",
                                   "path": "/testdir/hello",
                                   "target": "/testdir/helloworld"
                                 }
                               ]
                             }
                           }`),
		// This basic tests does not need to waste time on
		// other platforms
		Platforms: []string{"qemu", "qemu-unpriv"},
	})
}

func runsOnce(c cluster.TestCluster) {
	m := c.Machines()[0]

	// remove file created by Ignition; fail if it doesn't exist
	c.MustSSH(m, "sudo rm /etc/ignition-ran")

	err := m.Reboot()
	if err != nil {
		c.Fatalf("Couldn't reboot machine: %v", err)
	}

	// make sure file hasn't been recreated
	c.MustSSH(m, "test ! -e /etc/ignition-ran")
}

func testTranslation(c cluster.TestCluster) {
	m := c.Machines()[0]

	// fail if link is broken or does not exist
	c.MustSSH(m, "ls /testdir/hello")

	// assert that the networkd configuration has correctly been translated to files and applied.
	c.AssertCmdOutputContains(m, `ip --json address show kola | jq -r '.[] | .addr_info | .[] | select( .family == "inet") | .local'`, "10.0.2.1")
	c.AssertCmdOutputContains(m, `cat /etc/systemd/network/00-dummy.network`, "!vlan bond bridge")
}
