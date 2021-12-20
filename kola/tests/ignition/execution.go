// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

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
