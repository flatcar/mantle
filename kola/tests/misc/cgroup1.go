// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package misc

import (
	"fmt"
	"strings"

	"github.com/coreos/go-semver/semver"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/kola/tests/coretest"
	"github.com/flatcar-linux/mantle/platform/conf"
)

func init() {
	config := conf.Ignition(`{
"ignition": { "version": "2.0.0" },
  "storage": {
    "files": [{
      "filesystem": "root",
      "path": "/etc/flatcar-cgroupv1",
      "contents": { "source": "data:," },
      "mode": 420
    }]
  }
}`)

	register.Register(&register.Test{
		Run:         CgroupV1Test,
		ClusterSize: 1,
		Name:        "cl.cgroupv1",
		UserData:    config,
		NativeFuncs: map[string]func() error{
			"CgroupMounts": TestCgroup1Mounts,
		},
		Distros:    []string{"cl"},
		MinVersion: semver.Version{Major: 3033},
	})
}

func CgroupV1Test(c cluster.TestCluster) {
	tests := c.ListNativeFunctions()
	for _, name := range tests {
		c.RunNative(name, c.Machines()[0])
	}
}

func TestCgroup1Mounts() error {
	mounts, err := coretest.GetMountTable()
	if err != nil {
		return err
	}
	// check that we are no hybrid
	for _, mount := range mounts {
		if mount.FsType == "cgroup2" {
			return fmt.Errorf("cgroup2 is mounted: %v", mount)
		}
	}
	controllers := map[string]bool{
		"blkio":        false,
		"cpu":          false,
		"cpuacct":      false,
		"cpuset":       false,
		"devices":      false,
		"freezer":      false,
		"hugetlb":      false,
		"memory":       false,
		"name=systemd": false,
		"net_cls":      false,
		"net_prio":     false,
		"perf_event":   false,
		"pids":         false,
	}
	// check that we have all legacy controllers
	for _, mount := range mounts {
		if mount.FsType == "cgroup" {
			controllerFound := false
			for _, opt := range mount.Options {
				if _, ok := controllers[opt]; ok {
					controllers[opt] = true
					controllerFound = true
				}
			}
			if !controllerFound {
				return fmt.Errorf("unexpected controller: %v", mount)
			}
		}
	}
	missingControllers := make([]string, 0)
	for k, v := range controllers {
		if !v {
			missingControllers = append(missingControllers, k)
		}
	}
	if len(missingControllers) > 0 {
		return fmt.Errorf("cgroup controllers missing: %s", strings.Join(missingControllers, ","))
	}
	return nil
}
