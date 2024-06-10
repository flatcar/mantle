// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package hetzner

import (
	"context"
	"time"

	"github.com/flatcar/mantle/platform"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

// Options hold the specific Hetzner options.
type Options struct {
	*platform.Options
	// Image is the ID of the Hetzner image to deploy.
	Image string
	// InstanceType is the type of the instance (e.g cx11).
	InstanceType string
	// Location is used as "default" zone for all requests.
	Location string
	// Token is used to construct the Hetzner client
	Token string
}

// API is a wrapper around Hetzner instance API.
type API struct {
	opts   *Options
	client *hcloud.Client
}

type Server struct{ *hcloud.Server }

// New returns a Hetzner API instance.
func New(opts *Options) (*API, error) {
	return nil, nil
}

// CreateServer using the Hetzner API.
func (a *API) CreateServer(ctx context.Context, name, userdata string) (*Server, error) {
	return nil, nil
}

func (a *API) DeleteServer(ctx context.Context, id string) error { return nil }

// GC is the garbage collection (when we want to clear the project from resources created by Mantle (servers, images, etc.))
func (a *API) GC(ctx context.Context, gracePeriod time.Duration) error { return nil }

// GetConsoleOutput returns the console output using API calls or other.
func (a *API) GetConsoleOutput(id string) (string, error) { return "", nil }
