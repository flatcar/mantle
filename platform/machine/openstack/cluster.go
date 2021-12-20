// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/conf"
)

type cluster struct {
	*platform.BaseCluster
	flight *flight
}

func (oc *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	conf, err := oc.RenderUserData(userdata, map[string]string{
		"$public_ipv4":  "${COREOS_OPENSTACK_IPV4_PUBLIC}",
		"$private_ipv4": "${COREOS_OPENSTACK_IPV4_LOCAL}",
	})
	if err != nil {
		return nil, err
	}

	var keyname string
	if !oc.RuntimeConf().NoSSHKeyInMetadata {
		keyname = oc.flight.Name()
	}
	instance, err := oc.flight.api.CreateServer(oc.vmname(), keyname, conf.String())
	if err != nil {
		return nil, err
	}

	mach := &machine{
		cluster: oc,
		mach:    instance,
	}

	mach.dir = filepath.Join(oc.RuntimeConf().OutputDir, mach.ID())
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

	oc.AddMach(mach)

	return mach, nil
}

func (oc *cluster) vmname() string {
	b := make([]byte, 5)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", oc.Name()[0:13], b)
}

func (oc *cluster) Destroy() {
	oc.BaseCluster.Destroy()
	oc.flight.DelCluster(oc)
}
