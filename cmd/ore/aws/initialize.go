// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdInitialize = &cobra.Command{
		Use:   "initialize",
		Short: "initialize any uncreated resources for a given AWS region",
		RunE:  runInitialize,
	}

	bucket string
)

func init() {
	AWS.AddCommand(cmdInitialize)
	cmdInitialize.Flags().StringVar(&bucket, "bucket", "", "the S3 bucket URI to initialize; will default to a regional bucket")
}

func runInitialize(cmd *cobra.Command, args []string) error {
	if bucket == "" {
		bucket = defaultBucketNameForRegion(region)
	}

	err := API.InitializeBucket(bucket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not initialize bucket %v: %v\n", bucket, err)
		os.Exit(1)
	}

	err = API.CreateImportRole(bucket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create import role for %v: %v\n", bucket, err)
		os.Exit(1)
	}
	return nil
}
