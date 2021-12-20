// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	ctplatform "github.com/coreos/container-linux-config-transpiler/config/platform"
	"github.com/coreos/pkg/capnslog"

	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/openstack"
)

const (
	Platform platform.Name = "openstack"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "platform/machine/openstack")
)

type flight struct {
	*platform.BaseFlight
	api      *openstack.API
	keyAdded bool
}

// NewFlight creates an instance of a Flight suitable for spawning
// instances on the OpenStack platform.
func NewFlight(opts *openstack.Options) (platform.Flight, error) {
	api, err := openstack.New(opts)
	if err != nil {
		return nil, err
	}

	bf, err := platform.NewBaseFlight(opts.Options, Platform, ctplatform.OpenStackMetadata)
	if err != nil {
		return nil, err
	}

	of := &flight{
		BaseFlight: bf,
		api:        api,
	}

	keys, err := of.Keys()
	if err != nil {
		of.Destroy()
		return nil, err
	}

	if err := api.AddKey(of.Name(), keys[0].String()); err != nil {
		of.Destroy()
		return nil, err
	}
	of.keyAdded = true

	return of, nil
}

// NewCluster creates an instance of a Cluster suitable for spawning
// instances on the OpenStack platform.
func (of *flight) NewCluster(rconf *platform.RuntimeConfig) (platform.Cluster, error) {
	bc, err := platform.NewBaseCluster(of.BaseFlight, rconf)
	if err != nil {
		return nil, err
	}

	oc := &cluster{
		BaseCluster: bc,
		flight:      of,
	}

	of.AddCluster(oc)

	return oc, nil
}

func (of *flight) Destroy() {
	if of.keyAdded {
		if err := of.api.DeleteKey(of.Name()); err != nil {
			plog.Errorf("Error deleting key %v: %v", of.Name(), err)
		}
	}

	of.BaseFlight.Destroy()
}
