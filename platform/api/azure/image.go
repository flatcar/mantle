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
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
)

//go:embed gallery-image-template.json
var galleryImageTemplate []byte

const (
	deploymentName     = "kolagalleryimage"
	galleryNamePrefix  = "kolaSIG"
	imageVersion       = "0.0.0"
	secureImageVersion = "0.0.1"
)

type paramValue struct {
	Value string `json:"value"`
}

type galleryParams struct {
	GalleriesName       paramValue `json:"galleries_name"`
	ImageName           paramValue `json:"image_name"`
	ImageVersion        paramValue `json:"image_version"`
	SecureImageVersion  paramValue `json:"secure_image_version"`
	DB                  paramValue `json:"db"`
	StorageAccountsName paramValue `json:"storageAccounts_name"`
	VhdUri              paramValue `json:"vhd_uri"`
	Location            paramValue `json:"location"`
	Architecture        paramValue `json:"architecture"`
	HyperVGeneration    paramValue `json:"hyperVGeneration"`
}

func azureArchForBoard(board string) string {
	switch board {
	case "amd64-usr":
		return "x64"
	case "arm64-usr":
		return "Arm64"
	}
	return ""
}

// CreateGalleryImage creates an Azure Compute Gallery with 1 image version referencing the blob as the disk
func (a *API) CreateGalleryImage(name, resourceGroup, storageAccount, blobURI, db string) (string, error) {
	plog.Infof("Creating Gallery Image %s", name)
	galleryName := randomNameEx(galleryNamePrefix, "")
	template := make(map[string]interface{})
	err := json.Unmarshal(galleryImageTemplate, &template)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal gallery template: %w", err)
	}
	galleryParams := galleryParams{
		GalleriesName:       paramValue{galleryName},
		ImageName:           paramValue{name},
		ImageVersion:        paramValue{imageVersion},
		SecureImageVersion:  paramValue{secureImageVersion},
		DB:                  paramValue{db},
		StorageAccountsName: paramValue{storageAccount},
		VhdUri:              paramValue{blobURI},
		Location:            paramValue{a.Opts.Location},
		Architecture:        paramValue{azureArchForBoard(a.Opts.Board)},
		HyperVGeneration:    paramValue{a.Opts.HyperVGeneration},
	}
	params := make(map[string]interface{})
	paramsData, err := json.Marshal(&galleryParams)
	if err != nil {
		return "", fmt.Errorf("failed to marshal gallery params: %w", err)
	}
	err = json.Unmarshal(paramsData, &params)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal gallery params: %w", err)
	}

	poller, err := a.depClient.BeginCreateOrUpdate(context.TODO(),
		resourceGroup,
		deploymentName,
		armresources.Deployment{
			Properties: &armresources.DeploymentProperties{
				Mode:       to.Ptr(armresources.DeploymentModeIncremental),
				Parameters: params,
				Template:   template,
			},
		},
		nil,
	)
	if err != nil {
		return "", err
	}
	result, err := poller.PollUntilDone(context.TODO(), nil)
	if err != nil {
		return "", err
	}
	for _, entry := range result.Properties.OutputResources {
		if entry == nil || entry.ID == nil {
			continue
		}
		if strings.Contains(*entry.ID, "/versions/") {
			return *entry.ID, nil
		}
	}
	return "", fmt.Errorf("failed to find image version ID")
}

// CreateImage creates a managed image referencing the blob as the disk
func (a *API) CreateImage(name, resourceGroup, blobURI string) (armcompute.Image, error) {
	plog.Infof("Creating Image %s", name)
	poller, err := a.imgClient.BeginCreateOrUpdate(context.TODO(), resourceGroup, name, armcompute.Image{
		Name:     &name,
		Location: &a.Opts.Location,
		Properties: &armcompute.ImageProperties{
			HyperVGeneration: to.Ptr(armcompute.HyperVGenerationTypes(a.Opts.HyperVGeneration)),
			StorageProfile: &armcompute.ImageStorageProfile{
				OSDisk: &armcompute.ImageOSDisk{
					OSType:  to.Ptr(armcompute.OperatingSystemTypesLinux),
					OSState: to.Ptr(armcompute.OperatingSystemStateTypesGeneralized),
					BlobURI: &blobURI,
				},
			},
		},
	}, nil)
	if err != nil {
		return armcompute.Image{}, err
	}
	r, err := poller.PollUntilDone(context.TODO(), nil)
	if err != nil {
		return armcompute.Image{}, err
	}
	return r.Image, nil
}

// resolveImage is used to ensure that either a Version or DiskURI/BlobURL/ImageFile
// are provided present for a run. If neither is given via arguments
// it attempts to parse the Version from the version.txt in the Sku's
// release bucket.
func (a *API) resolveImage() error {
	// immediately return if the version has been set or if the channel
	// is not set via the Sku (this happens in ore)
	if a.Opts.DiskURI != "" || a.Opts.BlobURL != "" || a.Opts.ImageFile != "" || a.Opts.Version != "" || a.Opts.Sku == "" {
		return nil
	}
	sku := strings.TrimSuffix(a.Opts.Sku, "-gen2")
	resp, err := http.DefaultClient.Get(fmt.Sprintf("https://%s.release.flatcar-linux.net/amd64-usr/current/version.txt", sku))
	if err != nil {
		return fmt.Errorf("unable to fetch release bucket %v version: %v", a.Opts.Sku, err)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.SplitN(scanner.Text(), "=", 2)
		if len(line) != 2 {
			continue
		}
		if line[0] == "FLATCAR_VERSION" {
			a.Opts.Version = line[1]
			return nil
		}
	}

	return fmt.Errorf("couldn't find FLATCAR_VERSION in version.txt")
}
