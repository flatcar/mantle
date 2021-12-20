// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"fmt"
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar-linux/mantle/cli"
	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/aws"
	"github.com/spf13/cobra"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "ore/aws")

	AWS = &cobra.Command{
		Use:   "aws [command]",
		Short: "aws image and vm utilities",
	}

	API             *aws.API
	region          string
	credentialsFile string
	profileName     string
	accessKeyID     string
	secretAccessKey string
)

func init() {
	defaultRegion := os.Getenv("AWS_REGION")
	if defaultRegion == "" {
		defaultRegion = "us-west-2"
	}

	AWS.PersistentFlags().StringVar(&credentialsFile, "credentials-file", "", "AWS credentials file")
	AWS.PersistentFlags().StringVar(&profileName, "profile", "", "AWS profile name")
	AWS.PersistentFlags().StringVar(&accessKeyID, "access-id", "", "AWS access key")
	AWS.PersistentFlags().StringVar(&secretAccessKey, "secret-key", "", "AWS secret key")
	AWS.PersistentFlags().StringVar(&region, "region", defaultRegion, "AWS region")
	cli.WrapPreRun(AWS, preflightCheck)
}

func preflightCheck(cmd *cobra.Command, args []string) error {
	plog.Debugf("Running AWS Preflight check. Region: %v", region)
	api, err := aws.New(&aws.Options{
		Region:          region,
		CredentialsFile: credentialsFile,
		Profile:         profileName,
		Options:         &platform.Options{},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create AWS client: %v\n", err)
		os.Exit(1)
	}
	if err := api.PreflightCheck(); err != nil {
		fmt.Fprintf(os.Stderr, "could not complete AWS preflight check: %v\n", err)
		os.Exit(1)
	}

	plog.Debugf("Preflight check success; we have liftoff")
	API = api
	return nil
}
