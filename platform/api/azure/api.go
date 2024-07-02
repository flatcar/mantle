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
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/coreos/pkg/capnslog"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/api/azure")
)

type credData struct {
}

type API struct {
	cloudConfig cloud.Configuration
	creds       azcore.TokenCredential
	subID       string

	rgClient    *armresources.ResourceGroupsClient
	depClient   *armresources.DeploymentsClient
	imgClient   *armcompute.ImagesClient
	compClient  *armcompute.VirtualMachinesClient
	vmImgClient *armcompute.VirtualMachineImagesClient
	netClient   *armnetwork.VirtualNetworksClient
	subClient   *armnetwork.SubnetsClient
	ipClient    *armnetwork.PublicIPAddressesClient
	intClient   *armnetwork.InterfacesClient
	accClient   *armstorage.AccountsClient
	Opts        *Options
}

type Network struct {
	subnet armnetwork.Subnet
}

// New creates a new Azure client. If no publish settings file is provided or
// can't be parsed, an anonymous client is created.
func New(opts *Options) (*API, error) {
	var (
		err    error
		creds  azcore.TokenCredential
		subID  string
		config cloud.Configuration
	)

	disableInstanceDiscovery := strToBool(os.Getenv("AZURE_DISABLE_INSTANCE_DISCOVERY"), false)
	if opts.ADHost != "" {
		config.ActiveDirectoryAuthorityHost = opts.ADHost
	} else {
		if opts.CloudName == "" {
			opts.CloudName = os.Getenv("AZURE_CLOUD")
		}
		config, err = getCloudConfiguration(opts.CloudName)
		if err != nil {
			return nil, fmt.Errorf("failed to get cloud config: %w", err)
		}
	}

	authOpts := azidentity.DefaultAzureCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: config,
		},
		// Not setting the AdditionallyAllowedTenants here, use
		// AZURE_ADDITIONALLY_ALLOWED_TENANTS env var to set up
		// extra tenants - the azidentity module will take care
		// of it.
		AdditionallyAllowedTenants: nil,
		DisableInstanceDiscovery:   disableInstanceDiscovery,
		TenantID:                   os.Getenv("AZURE_TENANT_ID"),
	}
	creds, err = azidentity.NewDefaultAzureCredential(&authOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to get default Azure credentials: %w", err)
	}

	if opts.PreferredSubscriptionID == "" {
		opts.PreferredSubscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	}

	subIDs, err := querySubscriptions(context.TODO(), creds, config)
	if err != nil {
		return nil, fmt.Errorf("failed to query available subscriptions: %w", err)
	}
	if len(subIDs) == 0 {
		return nil, fmt.Errorf("no subscriptions are available for default credentials")
	}
	if opts.PreferredSubscriptionID == "" {
		if len(subIDs) > 1 {
			return nil, fmt.Errorf("many available subscriptions, need to specify a preferred subscription (e.g. through AZURE_SUBSCRIPTION_ID env var)")
		}
		subID = subIDs[0]
	} else {
		for _, queried := range subIDs {
			if queried == opts.PreferredSubscriptionID {
				subID = opts.PreferredSubscriptionID
				break
			}
		}
		if subID == "" {
			return nil, fmt.Errorf("preferred subscription %s is not a part of available subscriptions", opts.PreferredSubscriptionID)
		}
	}

	if opts.StorageEndpointSuffix == "" {
		opts.StorageEndpointSuffix = "core.windows.net"
	}

	if opts.AvailabilitySet != "" && opts.ResourceGroup == "" {
		return nil, fmt.Errorf("ResourceGroup must match AvailabilitySet")
	}

	api := &API{
		cloudConfig: config,
		creds:       creds,
		subID:       subID,
		Opts:        opts,
	}

	err = api.resolveImage()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve image: %v", err)
	}

	return api, nil
}

func (a *API) SetupClients() error {
	opts := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud: a.cloudConfig,
		},
	}

	rcf, err := armresources.NewClientFactory(a.subID, a.creds, opts)
	if err != nil {
		return err
	}
	a.rgClient = rcf.NewResourceGroupsClient()
	a.depClient = rcf.NewDeploymentsClient()

	ccf, err := armcompute.NewClientFactory(a.subID, a.creds, opts)
	if err != nil {
		return err
	}
	a.imgClient = ccf.NewImagesClient()
	a.compClient = ccf.NewVirtualMachinesClient()
	a.vmImgClient = ccf.NewVirtualMachineImagesClient()

	ncf, err := armnetwork.NewClientFactory(a.subID, a.creds, opts)
	if err != nil {
		return err
	}
	a.netClient = ncf.NewVirtualNetworksClient()
	a.subClient = ncf.NewSubnetsClient()
	a.ipClient = ncf.NewPublicIPAddressesClient()
	a.intClient = ncf.NewInterfacesClient()

	scf, err := armstorage.NewClientFactory(a.subID, a.creds, opts)
	if err != nil {
		return err
	}
	a.accClient = scf.NewAccountsClient()

	return nil
}

func (a *API) GetBlobServiceClient(storageAccount string) (*service.Client, error) {
	accountURL := fmt.Sprintf("https://%s.blob.%s", url.PathEscape(storageAccount), url.PathEscape(a.Opts.StorageEndpointSuffix))
	if _, err := url.Parse(accountURL); err != nil {
		return nil, err
	}
	opts := &service.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: a.cloudConfig,
		},
	}
	return service.NewClient(accountURL, a.creds, opts)
}

func strToBool(v string, onInvalid bool) bool {
	switch v {
	case "yes", "y", "true", "t", "1":
		return true
	case "no", "n", "false", "f", "0":
		return false
	default:
		return onInvalid
	}
}

func getCloudConfiguration(name string) (cloud.Configuration, error) {
	switch name {
	case "", "public", "pub":
		return cloud.AzurePublic, nil
	case "china", "cn":
		return cloud.AzureChina, nil
	case "government", "gov":
		return cloud.AzureGovernment, nil
	default:
		return cloud.Configuration{}, fmt.Errorf("invalid Azure cloud name: %s", name)
	}
}

func querySubscriptions(ctx context.Context, creds azcore.TokenCredential, config cloud.Configuration) ([]string, error) {
	opts := &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{
			Cloud: config,
		},
	}
	client, err := armsubscriptions.NewClient(creds, opts)
	if err != nil {
		return nil, err
	}
	// ClientListOptions is an empty struct, so pass nil instead
	pager := client.NewListPager(nil)
	var subIDs []string
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, sub := range page.Value {
			// this should never happen
			if sub.SubscriptionID == nil {
				continue
			}
			subIDs = append(subIDs, *sub.SubscriptionID)
		}
	}
	return subIDs, nil
}

func randomNameEx(prefix, separator string) string {
	b := make([]byte, 5)
	rand.Read(b)
	return fmt.Sprintf("%s%s%x", prefix, separator, b)
}

func randomName(prefix string) string {
	return randomNameEx(prefix, "-")
}

func (a *API) GetOpts() *Options {
	return a.Opts
}

func (a *API) GC(gracePeriod time.Duration) error {
	durationAgo := time.Now().Add(-1 * gracePeriod)

	listGroups, err := a.ListResourceGroups("")
	if err != nil {
		return fmt.Errorf("listing resource groups: %v", err)
	}

	for _, l := range listGroups {
		if strings.HasPrefix(*l.Name, "kola-cluster") {
			createdAt := *l.Tags["createdAt"]
			timeCreated, err := time.Parse(time.RFC3339, createdAt)
			if err != nil {
				return fmt.Errorf("error parsing time: %v", err)
			}
			if !timeCreated.After(durationAgo) {
				if err = a.TerminateResourceGroup(*l.Name); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
