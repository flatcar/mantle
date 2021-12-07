// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	ctplatform "github.com/coreos/container-linux-config-transpiler/config/platform"
	"github.com/coreos/pkg/capnslog"

	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/aws"
)

const (
	Platform platform.Name = "aws"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "platform/machine/aws")
)

type flight struct {
	*platform.BaseFlight
	api      *aws.API
	keyAdded bool
}

// NewFlight creates an instance of a Flight suitable for spawning
// instances on Amazon Web Services' Elastic Compute platform.
//
// NewFlight will consume the environment variables $AWS_REGION,
// $AWS_ACCESS_KEY_ID, and $AWS_SECRET_ACCESS_KEY to determine the region to
// spawn instances in and the credentials to use to authenticate.
func NewFlight(opts *aws.Options) (platform.Flight, error) {
	api, err := aws.New(opts)
	if err != nil {
		return nil, err
	}

	bf, err := platform.NewBaseFlight(opts.Options, Platform, ctplatform.EC2)
	if err != nil {
		return nil, err
	}

	af := &flight{
		BaseFlight: bf,
		api:        api,
	}

	keys, err := af.Keys()
	if err != nil {
		af.Destroy()
		return nil, err
	}
	if err := api.AddKey(af.Name(), keys[0].String()); err != nil {
		af.Destroy()
		return nil, err
	}
	af.keyAdded = true

	return af, nil
}

// NewCluster creates an instance of a Cluster suitable for spawning
// instances on Amazon Web Services' Elastic Compute platform.
func (af *flight) NewCluster(rconf *platform.RuntimeConfig) (platform.Cluster, error) {
	bc, err := platform.NewBaseCluster(af.BaseFlight, rconf)
	if err != nil {
		return nil, err
	}

	ac := &cluster{
		BaseCluster: bc,
		flight:      af,
	}

	af.AddCluster(ac)

	return ac, nil
}

func (af *flight) Destroy() {
	if af.keyAdded {
		if err := af.api.DeleteKey(af.Name()); err != nil {
			plog.Errorf("Error deleting key %v: %v", af.Name(), err)
		}
	}

	af.BaseFlight.Destroy()
}
