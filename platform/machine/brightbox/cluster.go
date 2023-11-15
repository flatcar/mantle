// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package brightbox

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
)

type cluster struct {
	*platform.BaseCluster
	flight *flight
}

func (bc *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	conf, err := bc.RenderUserData(userdata, map[string]string{
		"$public_ipv4":  "${COREOS_OPENSTACK_IPV4_PUBLIC}",
		"$private_ipv4": "${COREOS_OPENSTACK_IPV4_LOCAL}",
	})
	if err != nil {
		return nil, err
	}

	// Allocate a free cloudIP, this only works for low enough --parallel= values because "select default" does not block
	var cloudIP string
	select {
	case i := <-bc.flight.cloudIPs:
		cloudIP = i
	default:
		cloudIP = ""
	}

	instance, err := bc.flight.api.CreateServer(context.TODO(), bc.vmname(), conf.String(), cloudIP)
	if err != nil {
		return nil, err
	}

	mach := &machine{
		cluster: bc,
		mach:    instance,
	}

	mach.dir = filepath.Join(bc.RuntimeConf().OutputDir, mach.ID())
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
	bc.flight.DelCluster(bc)
}
