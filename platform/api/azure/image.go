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
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	"github.com/flatcar/mantle/sdk"

	"github.com/Azure/azure-sdk-for-go/services/classic/management"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2022-08-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
)

// OSImage struct for https://msdn.microsoft.com/en-us/library/azure/jj157192.aspx call.
//
// XXX: the field ordering is important!
type OSImage struct {
	XMLName           xml.Name `xml:"http://schemas.microsoft.com/windowsazure OSImage"`
	Category          string   `xml:",omitempty"` // Public || Private || MSDN
	Label             string   `xml:",omitempty"` // Specifies an identifier for the image.
	MediaLink         string   `xml:",omitempty"` // Specifies the location of the vhd file for the image. The storage account where the vhd is located must be associated with the specified subscription.
	Name              string   // Specifies the name of the operating system image. This is the name that is used when creating one or more virtual machines using the image.
	OS                string   // Linux || Windows
	Eula              string   `xml:",omitempty"` // Specifies the End User License Agreement that is associated with the image. The value for this element is a string, but it is recommended that the value be a URL that points to a EULA.
	Description       string   `xml:",omitempty"` // Specifies the description of the image.
	ImageFamily       string   `xml:",omitempty"` // Specifies a value that can be used to group images.
	PublishedDate     string   `xml:",omitempty"` // Specifies the date when the image was added to the image repository.
	ShowInGui         bool     // Specifies whether the image should appear in the image gallery.
	PrivacyURI        string   `xml:"PrivacyUri,omitempty"`   // Specifies the URI that points to a document that contains the privacy policy related to the image.
	IconURI           string   `xml:"IconUri,omitempty"`      // Specifies the Uri to the icon that is displayed for the image in the Management Portal.
	RecommendedVMSize string   `xml:",omitempty"`             // Specifies the size to use for the virtual machine that is created from the image.
	SmallIconURI      string   `xml:"SmallIconUri,omitempty"` // Specifies the URI to the small icon that is displayed when the image is presented in the Microsoft Azure Management Portal.
	Language          string   `xml:",omitempty"`             // Specifies the language of the image.

	LogicalSizeInGB   float64 `xml:",omitempty"` //Specifies the size, in GB, of the image.
	Location          string  `xml:",omitempty"` // The geo-location in which this media is located. The Location value is derived from storage account that contains the blob in which the media is located. If the storage account belongs to an affinity group the value is NULL.
	AffinityGroup     string  `xml:",omitempty"` // Specifies the affinity in which the media is located. The AffinityGroup value is derived from storage account that contains the blob in which the media is located. If the storage account does not belong to an affinity group the value is NULL and the element is not displayed in the response. This value is NULL for platform images.
	IsPremium         string  `xml:",omitempty"` // Indicates whether the image contains software or associated services that will incur charges above the core price for the virtual machine. For additional details, see the PricingDetailLink element.
	PublisherName     string  `xml:",omitempty"` // The name of the publisher of the image. All user images have a publisher name of User.
	PricingDetailLink string  `xml:",omitempty"` // Specifies a URL for an image with IsPremium set to true, which contains the pricing details for a virtual machine that is created from the image.
}

var azureImageShareURL = "services/images/%s/share?permission=%s"

func (a *API) ShareImage(image, permission string) error {
	url := fmt.Sprintf(azureImageShareURL, image, permission)
	op, err := a.client.SendAzurePutRequest(url, "", nil)
	if err != nil {
		return err
	}

	return a.client.WaitForOperation(op, nil)
}

func IsConflictError(err error) bool {
	azerr, ok := err.(management.AzureError)
	return ok && azerr.Code == "ConflictError"
}

//go:embed gallery-image-template.json
var galleryImageTemplate []byte

const (
	deploymentName    = "kolagalleryimage"
	galleryNamePrefix = "kolaSIG"
	imageVersion      = "0.0.0"
)

type paramValue struct {
	Value string `json:"value"`
}
type galleryParams struct {
	GalleriesName       paramValue `json:"galleries_name"`
	ImageName           paramValue `json:"image_name"`
	ImageVersion        paramValue `json:"image_version"`
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
func (a *API) CreateGalleryImage(name, resourceGroup, storageAccount, blobURI string) (string, error) {
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

	future, err := a.depClient.CreateOrUpdate(context.TODO(),
		resourceGroup,
		deploymentName,
		resources.Deployment{
			Properties: &resources.DeploymentProperties{
				Template:   template,
				Parameters: params,
				Mode:       resources.DeploymentModeIncremental,
			},
		},
	)
	if err != nil {
		return "", err
	}
	err = future.WaitForCompletionRef(context.TODO(), a.depClient.Client)
	if err != nil {
		return "", err
	}
	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/galleries/%s/images/%s/versions/%s",
		a.Opts.SubscriptionID, resourceGroup, galleryName, name, imageVersion)
	return id, nil
}

// CreateImage creates a managed image referencing the blob as the disk
func (a *API) CreateImage(name, resourceGroup, blobURI string) (compute.Image, error) {
	plog.Infof("Creating Image %s", name)
	future, err := a.imgClient.CreateOrUpdate(context.TODO(), resourceGroup, name, compute.Image{
		Name:     &name,
		Location: &a.Opts.Location,
		ImageProperties: &compute.ImageProperties{
			HyperVGeneration: compute.HyperVGenerationTypes(a.Opts.HyperVGeneration),
			StorageProfile: &compute.ImageStorageProfile{
				OsDisk: &compute.ImageOSDisk{
					OsType:  compute.OperatingSystemTypesLinux,
					OsState: compute.Generalized,
					BlobURI: &blobURI,
				},
			},
		},
	})
	if err != nil {
		return compute.Image{}, err
	}
	err = future.WaitForCompletionRef(context.TODO(), a.imgClient.Client)
	if err != nil {
		return compute.Image{}, err
	}
	return future.Result(a.imgClient)
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

	resp, err := http.DefaultClient.Get(fmt.Sprintf("https://%s.release.flatcar-linux.net/amd64-usr/current/version.txt", a.Opts.Sku))
	if err != nil {
		return fmt.Errorf("unable to fetch release bucket %v version: %v", a.Opts.Sku, err)
	}

	versions, err := sdk.ParseFlatcarVersions(resp.Body)
	if err != nil {
		return fmt.Errorf("couldn't parse version.txt: %v", err)
	}
	a.Opts.Version = versions.Version
	return nil
}
