// Copyright The Mantle Authors and The Go Authors
// SPDX-License-Identifier: Apache-2.0

package gcloud

import (
	"github.com/coreos/pkg/capnslog"

	ctplatform "github.com/coreos/container-linux-config-transpiler/config/platform"
	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/gcloud"
)

type flight struct {
	*platform.BaseFlight
	api *gcloud.API
}

const (
	Platform platform.Name = "gcloud"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "platform/machine/gcloud")
)

func NewFlight(opts *gcloud.Options) (platform.Flight, error) {
	api, err := gcloud.New(opts)
	if err != nil {
		return nil, err
	}

	bf, err := platform.NewBaseFlight(opts.Options, Platform, ctplatform.GCE)
	if err != nil {
		return nil, err
	}

	gf := &flight{
		BaseFlight: bf,
		api:        api,
	}

	return gf, nil
}

func (gf *flight) NewCluster(rconf *platform.RuntimeConfig) (platform.Cluster, error) {
	bc, err := platform.NewBaseCluster(gf.BaseFlight, rconf)
	if err != nil {
		return nil, err
	}

	gc := &cluster{
		BaseCluster: bc,
		flight:      gf,
	}

	gf.AddCluster(gc)

	return gc, nil
}
