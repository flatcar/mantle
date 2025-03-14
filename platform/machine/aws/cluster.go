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

package aws

import (
	"os"
	"path/filepath"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
)

type cluster struct {
	*platform.BaseCluster
	flight *flight
}

func (ac *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	return ac.NewMachineWithOptions(userdata, platform.MachineOptions{})
}

func (ac *cluster) NewMachineWithOptions(userdata *conf.UserData, options platform.MachineOptions) (platform.Machine, error) {
	conf, err := ac.RenderUserData(userdata, map[string]string{
		"$public_ipv4":  "${COREOS_EC2_IPV4_PUBLIC}",
		"$private_ipv4": "${COREOS_EC2_IPV4_LOCAL}",
	})
	if err != nil {
		return nil, err
	}

	var keyname string
	if !ac.RuntimeConf().NoSSHKeyInMetadata {
		keyname = ac.flight.Name()
	}

	// Only pass the extra disk size if it's set
	var rootDiskSize *int64
	if options.ExtraPrimaryDiskSize != "" {
		diskSize, err := platform.ParseDiskSize(options.ExtraPrimaryDiskSize)
		if err != nil {
			return nil, err
		}
		// expect disk size in GiB
		diskSizeSigned := int64(diskSize / (1024 * 1024 * 1024))
		rootDiskSize = &diskSizeSigned
	}

	instances, err := ac.flight.api.CreateInstances(ac.Name(), keyname, conf.String(), 1, rootDiskSize)
	if err != nil {
		return nil, err
	}

	mach := &machine{
		cluster: ac,
		mach:    instances[0],
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

func (ac *cluster) Destroy() {
	ac.BaseCluster.Destroy()
	ac.flight.DelCluster(ac)
}
