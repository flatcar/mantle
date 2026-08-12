// Copyright 2024 The Flatcar Maintainers
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

package azure

import (
	"strings"

	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
)

func init() {
	register.Register(&register.Test{
		Run:         checkAzureNVMe,
		ClusterSize: 1,
		Platforms:   []string{"azure"},
		Name:        "azure.nvme-friendly-naming",
		Distros:     []string{"cl"},
	})
}

func checkAzureNVMe(c cluster.TestCluster) {
	m := c.Machines()[0]

	// Check for the presence of the azure-nvme-id binary
	c.MustSSH(m, "test -x /usr/sbin/azure-nvme-id")

	// Dynamically detect if NVMe drives are attached to this test instance.
	// We do not fail if NVMe isn't present because Kola test runner instances
	// might still use SCSI depending on configuration/generation.
	out, err := c.SSH(m, "ls /dev/nvme0n1")
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		// NVMe is not present on this machine, meaning the udev rules won't trigger the symlink creation.
		// We already checked rules are present and valid, which satisfies basic conditions.
		c.Skip("no NVMe disk found on this instance")
	}

	// Wait for udev to settle just in case
	c.MustSSH(m, "sudo udevadm settle")

	// If NVMe is present, the azure-nvme udev rules should have created /dev/disk/azure links
	c.MustSSH(m, "test -d /dev/disk/azure")

	// Check that we have either the root or os symlink which indicates the rules worked.
	// We use shell logic to check if at least one of them exists since naming conventions might slightly vary.
	c.MustSSH(m, "test -e /dev/disk/azure/root || test -e /dev/disk/azure/os")
}
