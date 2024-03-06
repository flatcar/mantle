// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package scaleway

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"

	"github.com/flatcar/mantle/platform"
	maws "github.com/flatcar/mantle/platform/api/aws"
)

var (
	endpoint = "https://s3.%s.scw.cloud"
)

// Options hold the specific Scaleway options.
type Options struct {
	*platform.Options
	// AccessKey is the Scaleway access key in the AWS format.
	// Get the credentials at https://console.scaleway.com/iam/api-keys
	AccessKey string
	// Image is the ID of the Scaleway image to deploy.
	Image string
	// InstanceType is the type of the instance (e.g DEV1-S).
	InstanceType string
	// OrganizationID is the ID of the organization.
	// Get the ID at https://console.scaleway.com/organization/settings
	OrganizationID string
	// ProjectID is used as "default" project for all requests.
	ProjectID string
	// Region is used as "default" region for all requests.
	Region string
	// SecretKey is the Scaleway secret key in the AWS format.
	// Get the credentials at https://console.scaleway.com/iam/api-keys
	SecretKey string
	// Zone is used as "default" zone for all requests.
	Zone string
}

// API is a wrapper around Scaleway instance API and
// S3 AWS API (for object storage operation).
type API struct {
	opts *Options
	*maws.API
	instance *instance.API
}

// New returns a Scaleway API instance.
func New(opts *Options) (*API, error) {
	region, err := scw.ParseRegion(opts.Region)
	if err != nil {
		return nil, fmt.Errorf("parsing Scaleway region: %w", err)
	}

	zone, err := scw.ParseZone(opts.Zone)
	if err != nil {
		return nil, fmt.Errorf("parsing Scaleway zone: %w", err)
	}

	client, err := scw.NewClient(
		scw.WithDefaultOrganizationID(opts.OrganizationID),
		scw.WithAuth(opts.AccessKey, opts.SecretKey),
		scw.WithDefaultRegion(region),
		scw.WithDefaultZone(zone),
	)
	if err != nil {
		return nil, fmt.Errorf("creating Scaleway client: %w", err)
	}

	cfg := aws.Config{
		Region:      aws.String(opts.Region),
		Endpoint:    aws.String(fmt.Sprintf(endpoint, opts.Region)),
		Credentials: credentials.NewStaticCredentials(opts.AccessKey, opts.SecretKey, ""),
	}

	sess, err := session.NewSessionWithOptions(session.Options{Config: cfg})
	if err != nil {
		return nil, fmt.Errorf("creating s3 session: %w", err)
	}

	api := &API{
		API:      &maws.API{S3: s3.New(sess)},
		opts:     opts,
		instance: instance.NewAPI(client),
	}

	return api, nil
}
