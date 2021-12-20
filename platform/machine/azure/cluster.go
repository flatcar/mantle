// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/conf"
)

type cluster struct {
	*platform.BaseCluster
	flight         *flight
	sshKey         string
	ResourceGroup  string
	StorageAccount string
}

func (ac *cluster) vmname() string {
	b := make([]byte, 5)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", ac.Name()[0:13], b)
}

func (ac *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	conf, err := ac.RenderUserData(userdata, map[string]string{
		"$private_ipv4": "${COREOS_AZURE_IPV4_DYNAMIC}",
	})
	if err != nil {
		return nil, err
	}

	instance, err := ac.flight.api.CreateInstance(ac.vmname(), conf.String(), ac.sshKey, ac.ResourceGroup, ac.StorageAccount)
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
		if e := ac.flight.api.TerminateResourceGroup(ac.ResourceGroup); e != nil {
			plog.Errorf("Deleting resource group %v: %v", ac.ResourceGroup, e)
		}
	}
	ac.flight.DelCluster(ac)
}
