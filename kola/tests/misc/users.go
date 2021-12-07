// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package misc

import (
	"strings"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
)

func init() {
	register.Register(&register.Test{
		Run:              CheckUserShells,
		ClusterSize:      1,
		ExcludePlatforms: []string{"gce"},
		Name:             "cl.users.shells",
		Distros:          []string{"cl"},
	})
}

func CheckUserShells(c cluster.TestCluster) {
	m := c.Machines()[0]
	var badusers []string

	ValidUsers := map[string]string{
		"root":     "/bin/bash",
		"sync":     "/bin/sync",
		"shutdown": "/sbin/shutdown",
		"halt":     "/sbin/halt",
		"core":     "/bin/bash",
	}

	output := c.MustSSH(m, "getent passwd")

	users := strings.Split(string(output), "\n")

	for _, user := range users {
		userdata := strings.Split(user, ":")
		if len(userdata) != 7 {
			badusers = append(badusers, user)
			continue
		}

		username := userdata[0]
		shell := userdata[6]
		if shell == "/bin/sh" {
			// gentent returns one entry for root with /bin/sh instead of /bin/bash
			// but /bin/sh is anyway a symlink to /bin/bash
			shell = "/bin/bash"
		}
		if shell != ValidUsers[username] && shell != "/sbin/nologin" {
			badusers = append(badusers, user)
		}
	}

	if len(badusers) != 0 {
		c.Fatalf("Invalid users: %v", badusers)
	}
}
