// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"github.com/flatcar-linux/mantle/platform"
)

type Options struct {
	*platform.Options

	AzureProfile      string
	AzureAuthLocation string
	AzureSubscription string

	BlobURL          string
	ImageFile        string
	DiskURI          string
	Publisher        string
	Offer            string
	Sku              string
	Version          string
	Size             string
	Location         string
	HyperVGeneration string

	SubscriptionName string
	SubscriptionID   string

	// Azure API endpoint. If unset, the Azure SDK default will be used.
	ManagementURL         string
	ManagementCertificate []byte

	// Azure Storage API endpoint suffix. If unset, the Azure SDK default will be used.
	StorageEndpointSuffix string
}
