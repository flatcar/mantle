// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package brightbox

import (
	"context"
	"fmt"

	"github.com/coreos/pkg/capnslog"
	ctplatform "github.com/flatcar/container-linux-config-transpiler/config/platform"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/brightbox"
)

const (
	Platform platform.Name = "brightbox"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/machine/brightbox")
)

type flight struct {
	*platform.BaseFlight
	api      *brightbox.API
	cloudIPs chan string
}

func NewFlight(opts *brightbox.Options) (platform.Flight, error) {
	api, err := brightbox.New(opts)
	if err != nil {
		return nil, fmt.Errorf("creating brightbox API client: %w", err)
	}

	base, err := platform.NewBaseFlight(opts.Options, Platform, ctplatform.OpenStackMetadata)
	if err != nil {
		return nil, fmt.Errorf("creating base flight: %w", err)
	}

	bf := &flight{
		BaseFlight: base,
		api:        api,
		// Current CloudIPs limit is 5.
		cloudIPs: make(chan string, 999),
	}

	return bf, nil
}

// NewCluster creates an instance of a Cluster suitable for spawning
// instances on the OpenStack platform.
func (bf *flight) NewCluster(rconf *platform.RuntimeConfig) (platform.Cluster, error) {
	bc, err := platform.NewBaseCluster(bf.BaseFlight, rconf)
	if err != nil {
		return nil, fmt.Errorf("creating brightbox base cluster: %w", err)
	}

	c := &cluster{
		BaseCluster: bc,
		flight:      bf,
	}

	bf.AddCluster(c)

	return c, nil
}

func (bf *flight) Destroy() {
	// Clean the provisioned cloud IPs.
	close(bf.cloudIPs)
	for id := range bf.cloudIPs {
		if err := bf.api.DeleteCloudIP(context.TODO(), id); err != nil {
			plog.Errorf("deleting cloud IP %s: %v", id, err)
		}
	}

	bf.BaseFlight.Destroy()
}
