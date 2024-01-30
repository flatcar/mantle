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
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/classic/management"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2021-01-01/subscriptions"
	armStorage "github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2021-01-01/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/coreos/pkg/capnslog"

	internalAuth "github.com/flatcar/mantle/auth"
)

const (
	APIVersion = "2023-09-01"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/api/azure")
)

type API struct {
	client          management.Client
	rgClient        resources.GroupsClient
	depClient       resources.DeploymentsClient
	resourcesClient resources.Client
	imgClient       compute.ImagesClient
	compClient      compute.VirtualMachinesClient
	vmImgClient     compute.VirtualMachineImagesClient
	netClient       network.VirtualNetworksClient
	subClient       network.SubnetsClient
	ipClient        network.PublicIPAddressesClient
	intClient       network.InterfacesClient
	accClient       armStorage.AccountsClient
	Opts            *Options
}

type Network struct {
	subnet network.Subnet
}

func setOptsFromProfile(opts *Options) error {
	profiles, err := internalAuth.ReadAzureProfile(opts.AzureProfile)
	if err != nil {
		return fmt.Errorf("couldn't read Azure profile: %v", err)
	}

	if os.Getenv("AZURE_AUTH_LOCATION") == "" {
		if opts.AzureAuthLocation == "" {
			user, err := user.Current()
			if err != nil {
				return err
			}
			opts.AzureAuthLocation = filepath.Join(user.HomeDir, internalAuth.AzureAuthPath)
		}
		// TODO: Move to Flight once built to allow proper unsetting
		os.Setenv("AZURE_AUTH_LOCATION", opts.AzureAuthLocation)
	}

	var subOpts *internalAuth.Options
	if opts.AzureSubscription == "" {
		settings, err := auth.GetSettingsFromFile()
		if err != nil {
			return err
		}
		subOpts = profiles.SubscriptionOptions(internalAuth.FilterByID(settings.GetSubscriptionID()))
	} else {
		subOpts = profiles.SubscriptionOptions(internalAuth.FilterByName(opts.AzureSubscription))
	}
	if subOpts == nil {
		return fmt.Errorf("Azure subscription named %q doesn't exist in %q", opts.AzureSubscription, opts.AzureProfile)
	}

	if opts.SubscriptionID == "" {
		opts.SubscriptionID = subOpts.SubscriptionID
	}

	if opts.SubscriptionName == "" {
		opts.SubscriptionName = subOpts.SubscriptionName
	}

	if opts.ManagementURL == "" {
		opts.ManagementURL = subOpts.ManagementURL
	}

	if opts.ManagementCertificate == nil {
		opts.ManagementCertificate = subOpts.ManagementCertificate
	}

	if opts.StorageEndpointSuffix == "" {
		opts.StorageEndpointSuffix = subOpts.StorageEndpointSuffix
	}

	return nil
}

// New creates a new Azure client. If no publish settings file is provided or
// can't be parsed, an anonymous client is created.
func New(opts *Options) (*API, error) {
	var err error
	conf := management.DefaultConfig()
	conf.APIVersion = APIVersion

	if opts.ManagementURL != "" {
		conf.ManagementURL = opts.ManagementURL
	}

	if opts.StorageEndpointSuffix == "" {
		opts.StorageEndpointSuffix = storage.DefaultBaseURL
	}

	if !opts.UseIdentity {
		err = setOptsFromProfile(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get options from azure profile: %w", err)
		}
	} else {
		subid, err := msiGetSubscriptionID()
		if err != nil {
			return nil, fmt.Errorf("failed to query subscription id: %w", err)
		}
		opts.SubscriptionID = subid
	}

	var client management.Client
	if opts.ManagementCertificate != nil {
		client, err = management.NewClientFromConfig(opts.SubscriptionID, opts.ManagementCertificate, conf)
		if err != nil {
			return nil, fmt.Errorf("failed to create azure client: %v", err)
		}
	} else {
		client = management.NewAnonymousClient()
	}

	api := &API{
		client: client,
		Opts:   opts,
	}

	err = api.resolveImage()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve image: %v", err)
	}

	return api, nil
}

func (a *API) newAuthorizer(baseURI string) (autorest.Authorizer, error) {
	if !a.Opts.UseIdentity {
		return auth.NewAuthorizerFromFile(baseURI)
	}
	settings, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return nil, err
	}
	return settings.GetMSI().Authorizer()
}

func msiGetSubscriptionID() (string, error) {
	settings, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return "", err
	}
	subid := settings.GetSubscriptionID()
	if subid != "" {
		return subid, nil
	}
	auther, err := settings.GetMSI().Authorizer()
	if err != nil {
		return "", err
	}
	client := subscriptions.NewClient()
	client.Authorizer = auther
	iter, err := client.ListComplete(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to list subscriptions: %w", err)
	}
	for sub := iter.Value(); iter.NotDone(); iter.Next() {
		// this should never happen
		if sub.SubscriptionID == nil {
			continue
		}
		if subid != "" {
			return "", fmt.Errorf("multiple subscriptions found; pass one explicitly using the %s environment variable", auth.SubscriptionID)
		}
		subid = *sub.SubscriptionID
	}
	if subid == "" {
		return "", fmt.Errorf("no subscriptions found; pass one explicitly using the %s environment variable", auth.SubscriptionID)
	}
	plog.Infof("Using subscription %s", subid)
	return subid, nil
}

func (a *API) SetupClients() error {
	auther, err := a.newAuthorizer(resources.DefaultBaseURI)
	if err != nil {
		return err
	}
	subid := a.Opts.SubscriptionID

	a.rgClient = resources.NewGroupsClient(subid)
	a.rgClient.Authorizer = auther

	a.depClient = resources.NewDeploymentsClient(subid)
	a.depClient.Authorizer = auther

	a.resourcesClient = resources.NewClient(subid)
	a.resourcesClient.Authorizer = auther

	auther, err = a.newAuthorizer(compute.DefaultBaseURI)
	if err != nil {
		return err
	}
	a.imgClient = compute.NewImagesClient(subid)
	a.imgClient.Authorizer = auther
	a.compClient = compute.NewVirtualMachinesClient(subid)
	a.compClient.Authorizer = auther
	a.vmImgClient = compute.NewVirtualMachineImagesClient(subid)
	a.vmImgClient.Authorizer = auther

	auther, err = a.newAuthorizer(network.DefaultBaseURI)
	if err != nil {
		return err
	}
	a.netClient = network.NewVirtualNetworksClient(subid)
	a.netClient.Authorizer = auther
	a.subClient = network.NewSubnetsClient(subid)
	a.subClient.Authorizer = auther
	a.ipClient = network.NewPublicIPAddressesClient(subid)
	a.ipClient.Authorizer = auther
	a.intClient = network.NewInterfacesClient(subid)
	a.intClient.Authorizer = auther

	auther, err = a.newAuthorizer(armStorage.DefaultBaseURI)
	if err != nil {
		return err
	}
	a.accClient = armStorage.NewAccountsClient(subid)
	a.accClient.Authorizer = auther

	return nil
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

	for _, l := range *listGroups.Value {
		if strings.HasPrefix(*l.Name, "kola-cluster") {
			createdAt := *l.Tags["createdAt"]
			timeCreated, err := time.Parse(time.RFC3339, createdAt)
			if err != nil {
				return fmt.Errorf("error parsing time: %v", err)
			}
			if !timeCreated.After(durationAgo) {
				if err = a.TerminateResourceGroup(*l.Name, false); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
