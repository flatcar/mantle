// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/marketplacecatalog"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/coreos/pkg/capnslog"

	"github.com/flatcar/mantle/platform"
)

var plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/api/aws")

type Options struct {
	*platform.Options
	// The AWS region regional api calls should use
	Region string

	// The path to the shared credentials file, if not ~/.aws/credentials
	CredentialsFile string
	// The profile to use when resolving credentials, if applicable
	Profile string

	// AccessKeyID is the optional access key to use. It will override all other sources
	AccessKeyID string
	// SecretKey is the optional secret key to use. It will override all other sources
	SecretKey string

	// AMI is the AWS AMI to launch EC2 instances with.
	// If it is one of the special strings alpha|beta|stable, it will be resolved
	// to an actual ID.
	AMI                string
	InstanceType       string
	SecurityGroup      string
	IAMInstanceProfile string
}

type API struct {
	cfg         aws.Config
	ec2         *ec2.Client
	iam         *iam.Client
	marketplace *marketplacecatalog.Client
	S3          *s3.Client
	opts        *Options
}

// New creates a new AWS API wrapper. It uses credentials from any of the
// standard credentials sources, including the environment and the profile
// configured in ~/.aws.
// No validation is done that credentials exist and before using the API a
// preflight check is recommended via api.PreflightCheck
// Note that this method may modify Options to update the AMI ID
func New(opts *Options) (*API, error) {
	loadOpts := []func(*config.LoadOptions) error{
		config.WithRegion(opts.Region),
	}

	if opts.AccessKeyID != "" {
		loadOpts = append(loadOpts, config.WithCredentialsProvider(
			aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(
					opts.AccessKeyID,
					opts.SecretKey,
					"",
				),
			),
		))
	} else if opts.CredentialsFile != "" {
		loadOpts = append(loadOpts,
			config.WithSharedCredentialsFiles([]string{opts.CredentialsFile}),
		)
	}

	if opts.Profile != "" {
		loadOpts = append(loadOpts,
			config.WithSharedConfigProfile(opts.Profile),
		)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), loadOpts...)
	if err != nil {
		return nil, err
	}

	opts.AMI = resolveAMI(opts.AMI, opts.Region)

	api := &API{
		cfg:         cfg,
		ec2:         ec2.NewFromConfig(cfg),
		marketplace: marketplacecatalog.NewFromConfig(cfg),
		iam:         iam.NewFromConfig(cfg),
		S3:          s3.NewFromConfig(cfg),
		opts:        opts,
	}

	return api, nil
}

// GC removes AWS resources that are at least gracePeriod old.
// It attempts to only operate on resources that were created by a mantle tool.
func (a *API) GC(gracePeriod time.Duration) error {
	return a.gcEC2(gracePeriod)
}

// PreflightCheck validates that the aws configuration provided has valid
// credentials
func (a *API) PreflightCheck() error {
	stsClient := sts.NewFromConfig(a.cfg)
	_, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})

	return err
}

func (a *API) tagCreatedByMantle(resources []string) error {
	return a.CreateTags(resources, map[string]string{
		"CreatedBy": "mantle",
	})
}
