// Copyright The Mantle Authors and The Go Authors
// SPDX-License-Identifier: Apache-2.0

package gcloud

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/pkg/capnslog"
	"google.golang.org/api/compute/v1"

	"github.com/flatcar-linux/mantle/auth"
	"github.com/flatcar-linux/mantle/platform"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "platform/api/gcloud")
)

type Options struct {
	Image       string
	Project     string
	Zone        string
	MachineType string
	DiskType    string
	Network     string
	JSONKeyFile string
	ServiceAuth bool
	*platform.Options
}

type API struct {
	client  *http.Client
	compute *compute.Service
	options *Options
}

func New(opts *Options) (*API, error) {
	const endpointPrefix = "https://www.googleapis.com/compute/v1/"

	// If the image name isn't a full api endpoint accept a name beginning
	// with "projects/" to specify a different project from the instance.
	// Also accept a short name and use instance project.
	if strings.HasPrefix(opts.Image, "projects/") {
		opts.Image = endpointPrefix + opts.Image
	} else if !strings.Contains(opts.Image, "/") {
		opts.Image = fmt.Sprintf("%sprojects/%s/global/images/%s", endpointPrefix, opts.Project, opts.Image)
	} else if !strings.HasPrefix(opts.Image, endpointPrefix) {
		return nil, fmt.Errorf("GCE Image argument must be the full api endpoint, begin with 'projects/', or use the short name")
	}

	var (
		client *http.Client
		err    error
	)

	if opts.ServiceAuth {
		client = auth.GoogleServiceClient()
	} else if opts.JSONKeyFile != "" {
		b, err := ioutil.ReadFile(opts.JSONKeyFile)
		if err != nil {
			plog.Fatal(err)
		}
		client, err = auth.GoogleClientFromJSONKey(b)
	} else {
		client, err = auth.GoogleClient()
	}

	if err != nil {
		return nil, err
	}

	capi, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	api := &API{
		client:  client,
		compute: capi,
		options: opts,
	}

	return api, nil
}

func (a *API) Client() *http.Client {
	return a.client
}

func (a *API) GC(gracePeriod time.Duration) error {
	return a.gcInstances(gracePeriod)
}
