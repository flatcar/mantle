package util

import (
	"encoding/json"

	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/platform"
)

type Blockdevice struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	Mountpoint *string `json:"mountpoint"`
	// Mountpoints holds all mountpoints relevant for the device
	// it aims to replace `Mountpoint` from util-linux-2.37.
	Mountpoints []string      `json:"mountpoints"`
	Children    []Blockdevice `json:"children"`
}

type lsblkOutput struct {
	Blockdevices []Blockdevice `json:"blockdevices"`
}

// CheckMountpoint checks if a given machine has a device mounted at the given mountpoint that satisfies the given predicate.
// If not, the test is failed.
func CheckMountpoint(c cluster.TestCluster, m platform.Machine, mountpoint string, predicate func(Blockdevice) bool) {
	c.MustSSH(m, "lsblk -o NAME,LABEL")
	output := c.MustSSH(m, "lsblk --json")

	l := lsblkOutput{}
	err := json.Unmarshal(output, &l)
	if err != nil {
		c.Fatalf("couldn't unmarshal lsblk output: %v", err)
	}

	foundMountpoint := checkMountpointWalker(c, l.Blockdevices, mountpoint, predicate)
	if !foundMountpoint {
		c.Fatalf("didn't find mountpoint in lsblk output")
	}
}

// checkMountpointWalker will iterate over bs and recurse into its children, looking for a device mounted at `mountpoint`
// that satisfies the given predicate. true is returned if and only if such a device is found.
func checkMountpointWalker(c cluster.TestCluster, bs []Blockdevice, mountpoint string, predicate func(Blockdevice) bool) bool {
	for _, b := range bs {
		// >= util-linux-2.37
		for _, mnt := range b.Mountpoints {
			if mnt == mountpoint && predicate(b) {
				return true
			}
		}

		if b.Mountpoint != nil && *b.Mountpoint == mountpoint {
			if !predicate(b) {
				c.Fatalf("found device mounted at %q (%q), but failed to meet condition (had type %q)", mountpoint, b.Name, b.Type)
			}
			return true
		}
		foundMountpoint := checkMountpointWalker(c, b.Children, mountpoint, predicate)
		if foundMountpoint {
			return true
		}
	}
	return false
}
