// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package akamai

import (
	"fmt"

	"github.com/coreos/pkg/capnslog"
	ctplatform "github.com/flatcar/container-linux-config-transpiler/config/platform"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/akamai"
)

const (
	Platform platform.Name = "akamai"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/machine/akamai")
)

type flight struct {
	*platform.BaseFlight
	api *akamai.API
}

func NewFlight(opts *akamai.Options) (platform.Flight, error) {
	api, err := akamai.New(opts)
	if err != nil {
		return nil, fmt.Errorf("creating akamai API client: %w", err)
	}

	// TODO: Rework the Base Flight to remove the CT dependency.
	base, err := platform.NewBaseFlight(opts.Options, Platform, ctplatform.Custom)
	if err != nil {
		return nil, fmt.Errorf("creating base flight: %w", err)
	}

	bf := &flight{
		BaseFlight: base,
		api:        api,
	}

	return bf, nil
}

// NewCluster creates an instance of a Cluster suitable for spawning
// instances on the Akamai platform.
func (bf *flight) NewCluster(rconf *platform.RuntimeConfig) (platform.Cluster, error) {
	bc, err := platform.NewBaseCluster(bf.BaseFlight, rconf)
	if err != nil {
		return nil, fmt.Errorf("creating akamai base cluster: %w", err)
	}

	c := &cluster{
		BaseCluster: bc,
		flight:      bf,
	}

	bf.AddCluster(c)

	return c, nil
}

func (bf *flight) Destroy() {
	bf.BaseFlight.Destroy()
}
