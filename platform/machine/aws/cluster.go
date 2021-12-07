// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"os"
	"path/filepath"

	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/conf"
)

type cluster struct {
	*platform.BaseCluster
	flight *flight
}

func (ac *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
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
	instances, err := ac.flight.api.CreateInstances(ac.Name(), keyname, conf.String(), 1)
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
