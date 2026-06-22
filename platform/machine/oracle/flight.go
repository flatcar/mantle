// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package oracle

import (
	"fmt"

	"github.com/coreos/pkg/capnslog"
	ctplatform "github.com/flatcar/container-linux-config-transpiler/config/platform"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/oracle"
)

const (
	Platform platform.Name = "oracle"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/machine/oracle")
)

type flight struct {
	*platform.BaseFlight
	api *oracle.API
}

func NewFlight(opts *oracle.Options) (platform.Flight, error) {
	api, err := oracle.New(opts)
	if err != nil {
		return nil, fmt.Errorf("creating oracle API client: %w", err)
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
		return nil, fmt.Errorf("creating oracle base cluster: %w", err)
	}

	c := &cluster{
		BaseCluster: bc,
		flight:      bf,
	}

	bf.AddCluster(c)

	return c, nil
}
