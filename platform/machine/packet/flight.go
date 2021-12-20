// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package packet

import (
	"github.com/coreos/pkg/capnslog"

	ctplatform "github.com/coreos/container-linux-config-transpiler/config/platform"
	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/packet"
)

const (
	Platform platform.Name = "packet"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "platform/machine/packet")
)

type flight struct {
	*platform.BaseFlight
	api      *packet.API
	sshKeyID string
	// devicesPool holds the devices available
	// to be recycled by EM in order to minimize the
	// number of created devices.
	devicesPool chan string
}

func NewFlight(opts *packet.Options) (platform.Flight, error) {
	api, err := packet.New(opts)
	if err != nil {
		return nil, err
	}

	bf, err := platform.NewBaseFlight(opts.Options, Platform, ctplatform.Packet)
	if err != nil {
		return nil, err
	}

	pf := &flight{
		BaseFlight:  bf,
		api:         api,
		devicesPool: make(chan string, 1000),
	}

	keys, err := pf.Keys()
	if err != nil {
		pf.Destroy()
		return nil, err
	}
	pf.sshKeyID, err = pf.api.AddKey(pf.Name(), keys[0].String())
	if err != nil {
		pf.Destroy()
		return nil, err
	}

	return pf, nil
}

func (pf *flight) NewCluster(rconf *platform.RuntimeConfig) (platform.Cluster, error) {
	bc, err := platform.NewBaseCluster(pf.BaseFlight, rconf)
	if err != nil {
		return nil, err
	}

	pc := &cluster{
		BaseCluster: bc,
		flight:      pf,
	}
	if !rconf.NoSSHKeyInMetadata {
		pc.sshKeyID = pf.sshKeyID
	}

	pf.AddCluster(pc)

	return pc, nil
}

func (pf *flight) Destroy() {
	if pf.sshKeyID != "" {
		if err := pf.api.DeleteKey(pf.sshKeyID); err != nil {
			plog.Errorf("Error deleting key %v: %v", pf.sshKeyID, err)
		}
	}

	// before delete the instances from the devices pool
	// we close it in order to avoid deadlocks.
	close(pf.devicesPool)
	for id := range pf.devicesPool {
		if err := pf.api.DeleteDevice(id); err != nil {
			plog.Errorf("deleting device %s: %v", id, err)
		}
	}

	pf.BaseFlight.Destroy()
}
