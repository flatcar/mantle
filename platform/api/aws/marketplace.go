// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0
package aws

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/service/marketplacecatalog"
)

// UpdateProduct takes care of publishing the AMI to the AWS Marketplace by updating the existing product
// version.
// It takes care of scanning the AMI too.
func (a *API) UpdateProduct(amiID, accessRoleARN, username, version, productID, instanceType string, dryRun bool) error {
	// the type is always AmiProduct@1.0.
	// https://docs.aws.amazon.com/marketplace-catalog/latest/api-reference/ami-products.html
	t := "AmiProduct@1.0"

	catalog := "AWSMarketplace"
	changeType := "AddDeliveryOptions"

	details := fmt.Sprintf(`{
	"Version":{
		"VersionTitle":"%s",
		"ReleaseNotes":"https://www.flatcar.org/releases/#release-%s"
	},
	"DeliveryOptions":[{
		"Details":{
			"AmiDeliveryOptionDetails":{
				"AmiSource":{
					"AmiId":"%s",
					"AccessRoleArn":"%s",
					"UserName":"%s",
					"OperatingSystemName":"OTHERLINUX",
					"OperatingSystemVersion":"%s"
				},
				"UsageInstructions":"You can spin up single instances of Flatcar Container Linux using the instructions here: https://www.flatcar.org/docs/latest/installing/cloud/aws-ec2/\n\nFlatcar Container Linux supports a declarative provisioning using the Config format. Learn more from https://www.flatcar.org/docs/latest/provisioning/",
				"RecommendedInstanceType":"%s",
				"SecurityGroups":[{
					"IpProtocol":"tcp",
					"FromPort":22,
					"ToPort":22,
					"IpRanges":["0.0.0.0/0"]
				}]
			}
		}
	}]
}`, version, version, amiID, accessRoleARN, username, version, instanceType)

	// tricks to stringify the details JSON as required by the Marketplace API
	// https://docs.aws.amazon.com/marketplace-catalog/latest/api-reference/welcome.html#working-with-details
	m := make(map[string]interface{})
	if err := json.Unmarshal([]byte(details), &m); err != nil {
		return fmt.Errorf("unmarshalling the details: %w", err)
	}

	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshalling the details: %w", err)
	}

	details = string(data)

	input := marketplacecatalog.StartChangeSetInput{
		Catalog: &catalog,
		ChangeSet: []*marketplacecatalog.Change{
			&marketplacecatalog.Change{
				ChangeType: &changeType,
				Entity: &marketplacecatalog.Entity{
					Identifier: &productID,
					Type:       &t,
				},
				Details: &details,
			},
		},
	}

	if dryRun {
		fmt.Println(input.String())
		return nil
	}

	if _, err := a.marketplace.StartChangeSet(&input); err != nil {
		return fmt.Errorf("starting change set: %w", err)
	}

	return nil
}
