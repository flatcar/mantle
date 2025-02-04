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

package azure

import (
	"github.com/flatcar/mantle/platform"
)

type Options struct {
	*platform.Options

	// Base URL to cloud's Azure Active Directory. If not empty,
	// CloudName is ignored.
	ADHost string
	// Name of the Azure cloud. Can be "public" or "pub" for Azure
	// Public Cloud, "government" or "gov" for Azure Government
	// and "china" or "cn" for Azure China. If empty, AZURE_CLOUD
	// environment variable will be checked too. If still empty,
	// default to Azure Public Cloud.
	CloudName string
	// ID of a preferred subscription in case where more than 1 is
	// available for given credentials. If empty,
	// AZURE_SUBSCRIPTION_ID environment variable will be checked
	// too.
	PreferredSubscriptionID string

	// Common opts

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
	VnetSubnetName   string
	KolaVnet         string
	UseGallery       bool
	UsePrivateIPs    bool

	DiskController string

	// Azure Storage API endpoint suffix. If unset, the Azure SDK default will be used.
	StorageEndpointSuffix string
	// ResourceGroup is an existing resource group to deploy resources in.
	ResourceGroup string
	// ResourceGroupBasename is the prefix used for creating new resource groups
	ResourceGroupBasename string
	// AvailabilitySet is an existing availability set to deploy the instance in.
	AvailabilitySet string
	// VMIdentity is the name of a managed identity to assign to the VM.
	VMIdentity string
}
