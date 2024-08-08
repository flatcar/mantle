// Copyright 2018 CoreOS, Inc.
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
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/azure"
	"github.com/flatcar/mantle/platform/conf"
)

type cluster struct {
	*platform.BaseCluster
	flight            *flight
	sshKey            string
	ResourceGroup     string
	Network           azure.Network
	ManagedIdentityID string // Add managed identity ID field to cluster struct
}

func (ac *cluster) vmname() string {
	b := make([]byte, 5)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", ac.Name()[0:13], b)
}

func (ac *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	return ac.NewMachineWithOptions(userdata, platform.MachineOptions{})
}

func (ac *cluster) NewMachineWithOptions(userdata *conf.UserData, options platform.MachineOptions) (platform.Machine, error) {
	conf, err := ac.RenderUserData(userdata, map[string]string{
		"$private_ipv4": "${COREOS_AZURE_IPV4_DYNAMIC}",
	})
	if err != nil {
		return nil, err
	}

	var instanceOptions azure.InstanceOptions

	// If ExtraPrimaryDiskSize is set, configure the custom disk size
	if options.ExtraPrimaryDiskSize != "" {
		diskSize, err := platform.ParseDiskSize(options.ExtraPrimaryDiskSize)
		if err != nil {
			return nil, err
		}
		// Convert to int32 as that's what Azure API expects
		instanceOptions.DiskSizeGB = int32(diskSize / (1024 * 1024 * 1024))
	}

	// Create the instance with the specified options
	instance, err := ac.flight.Api.CreateInstance(
		ac.vmname(),
		ac.sshKey,
		ac.ResourceGroup,
		conf,
		ac.Network,
		ac.ManagedIdentityID,
		instanceOptions,
	)

	if err != nil {
		return nil, err
	}

	mach := &machine{
		cluster: ac,
		mach:    instance,
	}

	mach.dir = filepath.Join(ac.RuntimeConf().OutputDir, mach.ID())
	if err := os.Mkdir(mach.dir, 0777); err != nil {
		mach.Destroy()
		return nil, err
	}

	confPath := filepath.Join(mach.dir, "user-data")
	if err := conf.WriteFile(confPath); err != nil {
		mach.Destroy()
		return nil, err
	}

	// We want to get to this point on CreateError (eg. OS Provisioning Timeout)
	// so that the serial console output is captured for debugging.
	if instance.CreateError != nil {
		mach.Destroy()
		return nil, instance.CreateError
	}

	if mach.journal, err = platform.NewJournal(mach.dir); err != nil {
		mach.Destroy()
		return nil, err
	}

	if err := platform.StartMachine(mach, mach.journal); err != nil {
		mach.Destroy()
		return nil, err
	}

	ac.AddMach(mach)

	return mach, nil
}

// Destroy deletes the Resource Group if it was created for this cluster, but it doesn't
// delete the Resource Group if the cluster runs in the Flight's image Resource Group
func (ac *cluster) Destroy() {
	ac.BaseCluster.Destroy()
	if ac.ResourceGroup != ac.flight.ImageResourceGroup {
		if e := ac.flight.Api.TerminateResourceGroup(ac.ResourceGroup); e != nil {
			plog.Errorf("Deleting resource group %v: %v", ac.ResourceGroup, e)
		}
	}
	ac.flight.DelCluster(ac)
}
