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
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"

	"github.com/flatcar/mantle/util"
)

func (a *API) CreateResourceGroup(prefix string) (string, error) {
	name := randomName(prefix)
	tags := map[string]*string{
		"createdAt": util.StrToPtr(time.Now().Format(time.RFC3339)),
		"createdBy": util.StrToPtr("mantle"),
	}
	plog.Infof("Creating ResourceGroup %s", name)
	_, err := a.rgClient.CreateOrUpdate(context.TODO(), name, resources.Group{
		Location: &a.Opts.Location,
		Tags:     tags,
	})
	if err != nil {
		return "", err
	}

	return name, nil
}

// keepResourceGroup can be used to terminate all the resources created in the group but keep the group itself.
func (a *API) TerminateResourceGroup(name string, keepResourceGroup bool) error {
	if !keepResourceGroup {
		resp, err := a.rgClient.CheckExistence(context.TODO(), name)
		if err != nil {
			return err
		}
		if resp.StatusCode != 204 {
			return nil
		}

		_, err = a.rgClient.Delete(context.TODO(), name)
		return err
	}

	// Get the list of resources to delete in the group
	listResult, err := a.resourcesClient.ListByResourceGroup(context.TODO(), name, "", "", nil)
	if err != nil {
		return fmt.Errorf("listing by resource group: %v", err)
	}

	for _, value := range listResult.Values() {
		id := *value.ID
		t := *value.Type

		if id != "" && t != "" {
			if t == a.Opts.ResourceToKeep {
				continue
			}

			if _, err := a.resourcesClient.DeleteByID(context.TODO(), id, APIVersion); err != nil {
				return fmt.Errorf("deleting resource %s: %v", id, err)
			}

			plog.Infof("deleted collected resource: kind: %s", *value.Type)
		}
	}

	return nil
}

func (a *API) ListResourceGroups(filter string) (resources.GroupListResult, error) {
	iter, err := a.rgClient.ListComplete(context.TODO(), filter, nil)
	if err != nil {
		return resources.GroupListResult{}, err
	}
	var results resources.GroupListResult
	arr := make([]resources.Group, 0)
	results.Value = &arr

	for ; iter.NotDone(); err = iter.NextWithContext(context.TODO()) {
		if err != nil {
			return resources.GroupListResult{}, err
		}
		arr = append(arr, iter.Value())
	}
	return results, err
}
