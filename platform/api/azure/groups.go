// Copyright 2018 CoreOS, Inc.
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
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

func (a *API) CreateResourceGroup(prefix string) (string, error) {
	name := randomName(prefix)
	tags := map[string]*string{
		"createdAt": to.Ptr(time.Now().Format(time.RFC3339)),
		"createdBy": to.Ptr("mantle"),
	}
	plog.Infof("Creating ResourceGroup %s", name)
	r, err := a.rgClient.CreateOrUpdate(context.TODO(), name, armresources.ResourceGroup{
		Location: &a.Opts.Location,
		Tags:     tags,
	}, nil)
	if err != nil {
		return "", err
	}
	if r.Name == nil {
		return name, nil
	}

	return *r.Name, nil
}

func (a *API) TerminateResourceGroup(name string) error {
	{
		r, err := a.rgClient.CheckExistence(context.TODO(), name, nil)
		if err != nil {
			return err
		}
		if !r.Success {
			return nil
		}
	}

	opts := &armresources.ResourceGroupsClientBeginDeleteOptions{
		ForceDeletionTypes: to.Ptr("Microsoft.Compute/virtualMachines,Microsoft.Compute/virtualMachineScaleSets"),
	}
	_, err := a.rgClient.BeginDelete(context.TODO(), name, opts)
	return err
}

func (a *API) ListResourceGroups(filter string) ([]*armresources.ResourceGroup, error) {
	opts := &armresources.ResourceGroupsClientListOptions{
		Filter: &filter,
	}
	pager := a.rgClient.NewListPager(opts)
	var list []*armresources.ResourceGroup
	for pager.More() {
		page, err := pager.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		list = append(list, page.Value...)
	}
	return list, nil
}
