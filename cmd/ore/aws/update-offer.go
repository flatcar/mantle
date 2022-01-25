// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0
package aws

import (
	"github.com/spf13/cobra"
)

var (
	cmdUpdate = &cobra.Command{
		Use:   "update-offer",
		Short: "Update a product offer on the AWS Marketplace",
		Long:  "Use this command to update the marketplace product offer with the new AMI and the new version",
		Example: `ore aws update-offer \
	--ami-id ami-1234567890abcdef \
	--access-role-arn arn:aws:iam::12345678901:role/AwsMarketplaceAmiIngestion \
	--version 3066.1.1 \
	--instance-type m6g.medium \
	--product-id 12345-67890-1111-2222`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return API.UpdateProduct(amiID, accessRoleARN, username, version, productID, instanceType, dryRun)
		},
	}

	amiID, accessRoleARN, username, version, productID, instanceType string

	dryRun bool
)

func init() {
	AWS.AddCommand(cmdUpdate)
	cmdUpdate.Flags().StringVar(&amiID, "ami-id", "", "ID of the AMI")
	cmdUpdate.Flags().StringVar(&accessRoleARN, "access-role-arn", "", "ARN to give marketplace access to the AMI")
	cmdUpdate.Flags().StringVar(&username, "username", "core", "default username")
	cmdUpdate.Flags().StringVar(&version, "version", "", "new version title")
	cmdUpdate.Flags().StringVar(&productID, "product-id", "", "AWS Marketplace offer ID")
	cmdUpdate.Flags().StringVar(&instanceType, "instance-type", "", "AWS recommended instance type")
	cmdUpdate.Flags().BoolVar(&dryRun, "dry-run", false, "Dry run to inspect update offer request")
}
