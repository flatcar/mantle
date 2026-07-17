// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

import (
	"fmt"

	"github.com/coreos/pkg/capnslog"
	ctplatform "github.com/flatcar/container-linux-config-transpiler/config/platform"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/oraclecloud"
)

const (
	Platform platform.Name = "oraclecloud"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/machine/oraclecloud")
)

type flight struct {
	*platform.BaseFlight
	api *oraclecloud.API
}

func NewFlight(opts *oraclecloud.Options) (platform.Flight, error) {
	api, err := oraclecloud.New(opts)
	if err != nil {
		return nil, fmt.Errorf("creating oraclecloud API client: %w", err)
	}

	base, err := platform.NewBaseFlight(opts.Options, Platform, ctplatform.Custom)
	if err != nil {
		return nil, fmt.Errorf("creating base flight: %w", err)
	}

	return &flight{
		BaseFlight: base,
		api:        api,
	}, nil
}

func (bf *flight) NewCluster(rconf *platform.RuntimeConfig) (platform.Cluster, error) {
	bc, err := platform.NewBaseCluster(bf.BaseFlight, rconf)
	if err != nil {
		return nil, fmt.Errorf("creating oraclecloud base cluster: %w", err)
	}

	c := &cluster{
		BaseCluster: bc,
		flight:      bf,
	}

	bf.AddCluster(c)

	return c, nil
}
