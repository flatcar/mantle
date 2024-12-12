// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package akamai

import (
	"net/http"

	"github.com/flatcar/mantle/platform"
	"github.com/linode/linodego"
	"golang.org/x/oauth2"
)

// API is a wrapper around Akamai client API
type API struct {
	opts   *Options
	client *linodego.Client
}

// Options hold the specific Akamai options.
type Options struct {
	*platform.Options
	// Token to access Akamai resources.
	Token string
	// Image is the ID of the Akamai image to deploy.
	Image string
	// Region where to deploy instances
	Region string
	// Type of the instance to deploy
	Type string
}

// New returns an Akamai API instance.
func New(opts *Options) (*API, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: opts.Token})
	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	client := linodego.NewClient(oauth2Client)

	return &API{
		client: &client,
		opts:   opts,
	}, nil
}
