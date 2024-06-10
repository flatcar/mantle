// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package hetzner

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/hetzner"
	"github.com/flatcar/mantle/platform/conf"
)

type cluster struct {
	*platform.BaseCluster
	flight  *flight
	sshKey  *hetzner.SSHKey
	network *hetzner.Network
}

func (bc *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	conf, err := bc.RenderUserData(userdata, map[string]string{
		"$public_ipv4":  "${COREOS_CUSTOM_PUBLIC_IPV4}",
		"$private_ipv4": "${COREOS_CUSTOM_PRIVATE_IPV4}",
	})
	if err != nil {
		return nil, err
	}

	// Hack to workaround CT inheritance.
	// Can be dropped once we remove CT dependency.
	// https://github.com/flatcar/Flatcar/issues/1386
	// Does this need a workaround for https://github.com/coreos/afterburn/pull/1083?
	// The COREOS_COREOS line is a workaround for https://github.com/coreos/afterburn/pull/1083
	conf.AddSystemdUnitDropin("coreos-metadata.service", "00-custom-metadata.conf", `[Service]
ExecStartPost=/usr/bin/sed -i "s/HETZNER/CUSTOM/" /run/metadata/flatcar
ExecStartPost=/usr/bin/sed -i "s/COREOS_CUSTOM_PRIVATE_IPV4_0/COREOS_CUSTOM_PRIVATE_IPV4/" /run/metadata/flatcar
`)

	instance, err := bc.flight.api.CreateServer(context.TODO(), bc.vmname(), conf.String(), bc.sshKey, bc.network)
	if err != nil {
		return nil, err
	}

	mach := &machine{
		cluster: bc,
		mach:    instance,
	}

	// machine to destroy
	m := mach
	defer func() {
		if m != nil {
			m.Destroy()
		}
	}()

	mach.dir = filepath.Join(bc.RuntimeConf().OutputDir, mach.ID())
	if err := os.Mkdir(mach.dir, 0777); err != nil {
		return nil, err
	}

	confPath := filepath.Join(mach.dir, "ignition.json")
	if err := conf.WriteFile(confPath); err != nil {
		return nil, err
	}

	if mach.journal, err = platform.NewJournal(mach.dir); err != nil {
		return nil, err
	}

	if err := platform.StartMachine(mach, mach.journal); err != nil {
		return nil, err
	}

	m = nil
	bc.AddMach(mach)

	return mach, nil
}

func (bc *cluster) vmname() string {
	b := make([]byte, 5)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", bc.Name()[0:13], b)
}

func (bc *cluster) Destroy() {
	bc.BaseCluster.Destroy()
	if bc.network != nil {
		if err := bc.flight.api.DeleteNetwork(context.TODO(), bc.network); err != nil {
			plog.Errorf("deleting network %v: %v", bc.network.Name, err)
		}

	}

	bc.flight.DelCluster(bc)
}
