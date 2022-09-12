// Copyright 2018 Red Hat
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

package openstack

import (
	"fmt"

	"github.com/coreos/pkg/capnslog"
	ctplatform "github.com/flatcar/container-linux-config-transpiler/config/platform"

	"github.com/flatcar/mantle/network"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/openstack"
)

const (
	Platform platform.Name = "openstack"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/machine/openstack")
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

	var bf *platform.BaseFlight

	if opts.Host != "" {
		if opts.User == "" || opts.Keyfile == "" {
			return nil, fmt.Errorf("--openstack-user and --openstack-keyfile can't be empty when using --openstack-host")
		}

		d, err := network.NewJumpDialer(opts.Host, opts.User, opts.Keyfile)
		if err != nil {
			return nil, fmt.Errorf("setting proxy jump dialer: %w", err)
		}

		bf, err = platform.NewBaseFlightWithDialer(opts.Options, Platform, ctplatform.OpenStackMetadata, d)
		if err != nil {
			return nil, fmt.Errorf("creating base flight with jump dialer: %w", err)
		}
	} else {
		if opts.User == "" || opts.Keyfile == "" {
			plog.Info("--openstack-user and/or --openstack-keyfile are provided but ignored")
		}

		bf, err = platform.NewBaseFlight(opts.Options, Platform, ctplatform.OpenStackMetadata)
		if err != nil {
			return nil, err
		}
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
