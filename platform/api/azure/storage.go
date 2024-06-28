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
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/pageblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/sas"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
	"github.com/flatcar/azure-vhd-utils/op"
)

var (
	azureImageURL = "services/images"
)

func BlobExists(client *service.Client, containerName, blobName string) (bool, error) {
	blobClient := client.NewContainerClient(containerName).NewBlobClient(blobName)
	if _, err := blobClient.GetProperties(context.TODO(), nil); err != nil {
		if !bloberror.HasCode(err, bloberror.BlobNotFound, bloberror.ResourceNotFound) {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func UploadBlob(client *service.Client, vhd, containerName, blobName string, overwrite bool) error {
	opts := op.UploadOptions{
		Overwrite:   overwrite,
		Parallelism: 8,
		Logger: func(s string) {
			plog.Printf("%s", s)
		},
	}
	return op.Upload(context.TODO(), client, containerName, blobName, vhd, &opts)
}

// Used in ore and plume
func SignBlob(client *service.Client, containerName, blobName string) (string, error) {
	blobClient := client.NewContainerClient(containerName).NewBlobClient(blobName)
	sasPermissions := sas.BlobPermissions{}
	sasPermissions.Read = true
	sasPermissions.List = true
	expiry := time.Date(2099, time.December, 31, 23, 59, 59, 0, time.UTC)
	return blobClient.GetSASURL(sasPermissions, expiry, nil)
}

func DeleteBlob(client *service.Client, containerName, blobName string) error {
	blobClient := client.NewContainerClient(containerName).NewBlobClient(blobName)
	opts := blob.DeleteOptions{
		DeleteSnapshots: to.Ptr(blob.DeleteSnapshotsOptionTypeInclude),
	}
	if _, err := blobClient.Delete(context.TODO(), &opts); err != nil {
		return err
	}
	return nil
}

func ListBlobs(client *service.Client, containerName string, include container.ListBlobsInclude) ([]*container.BlobItem, error) {
	containerClient := client.NewContainerClient(containerName)

	var (
		marker *string
		items  []*container.BlobItem
	)

	for {
		opts := container.ListBlobsFlatOptions{
			Include: include,
			Marker:  marker,
		}
		pager := containerClient.NewListBlobsFlatPager(&opts)
		for pager.More() {
			page, err := pager.NextPage(context.TODO())
			if err != nil {
				return nil, fmt.Errorf("failed listing blobs for %q: %w", containerName, err)
			}
			items = append(items, page.Segment.BlobItems...)
			marker = page.Marker
		}
		if marker == nil || *marker == "" {
			break
		}
	}
	return items, nil
}

func BlobURL(client *service.Client, containerName, blobName string) string {
	return client.NewContainerClient(containerName).NewBlobClient(blobName).URL()
}

func GetBlob(client *service.Client, containerName, blobName string) (io.ReadCloser, error) {
	ctx := context.TODO()
	blobClient := client.NewContainerClient(containerName).NewBlobClient(blobName)
	response, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return nil, err
	}

	return response.NewRetryReader(ctx, nil), nil
}

func CopyBlob(client *service.Client, containerName, blobName, sourceURL string) error {
	if tryCopyPageBlob(client, containerName, blobName, sourceURL) {
		return nil
	}
	if tryCopyAzcopy(client, containerName, blobName, sourceURL) {
		return nil
	}
	if tryCopyBlockBlob(client, containerName, blobName, sourceURL) {
		return nil
	}

	return fmt.Errorf("Failed to copy %q to %q/%q", sourceURL, containerName, blobName)
}

func tryCopyPageBlob(client *service.Client, containerName, blobName, sourceURL string) bool {
	ctx := context.TODO()
	srcPageBlobClient, err := pageblob.NewClientWithNoCredential(sourceURL, nil)
	if err != nil {
		return false
	}
	srcBlobClient := srcPageBlobClient.BlobClient()
	srcProperties, err := srcBlobClient.GetProperties(ctx, nil)
	if err != nil {
		return false
	}
	if srcProperties.ContentLength == nil {
		return false
	}

	// we have a size, so we can create the target container and pageblob in it
	dstContainerClient := client.NewContainerClient(containerName)
	_, err = dstContainerClient.Create(ctx, nil)
	if err != nil && !bloberror.HasCode(err, bloberror.ContainerAlreadyExists, bloberror.ResourceAlreadyExists) {
		return false
	}
	dstPageBlobClient := dstContainerClient.NewPageBlobClient(blobName)
	_, err = dstPageBlobClient.Create(ctx, *srcProperties.ContentLength, nil)
	if err != nil {
		return false
	}
	deleteBlob := true
	defer func() {
		if deleteBlob {
			// ignore errors
			_ = DeleteBlob(client, containerName, blobName)
		}
	}()

	// we have allocated a target blob, now copy the pages
	var marker *string
	for {
		opts := pageblob.GetPageRangesOptions{
			Marker: marker,
		}
		pager := srcPageBlobClient.NewGetPageRangesPager(&opts)
		for pager.More() {
			response, err := pager.NextPage(ctx)
			if err != nil {
				return false
			}
			for _, page := range response.PageRange {
				offset := *page.Start
				// Both the page start and end are
				// inclusive, thus the "+ 1" to obtain
				// the length.
				count := *page.End - *page.Start + 1
				_, err = dstPageBlobClient.UploadPagesFromURL(ctx, sourceURL, offset, offset, count, nil)
				if err != nil {
					return false
				}
			}
			marker = response.NextMarker
		}
		if marker == nil || *marker == "" {
			break
		}
	}

	deleteBlob = false
	return true
}

func tryCopyAzcopy(client *service.Client, containerName, blobName, sourceURL string) bool {
	azcopy, err := exec.LookPath("azcopy")
	if err != nil {
		return false
	}

	blobClient := client.NewContainerClient(containerName).NewBlobClient(blobName)
	sasPermissions := sas.BlobPermissions{}
	sasPermissions.Read = true
	sasPermissions.Write = true
	sasPermissions.Create = true
	expiry := time.Now().Add(15 * time.Minute)
	dstSAS, err := blobClient.GetSASURL(sasPermissions, expiry, nil)
	if err != nil {
		return false
	}
	// log-level=NONE only affects the log file - stdout is unaffected
	cmd := exec.Command(azcopy, "cp", "--blob-type=PageBlob", "--log-level=NONE", sourceURL, dstSAS)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	// azcopy leaves behind "plan" files in case a job needs to be retried
	_ = exec.Command("azcopy", "jobs", "clean").Run()
	return err == nil
}

func tryCopyBlockBlob(client *service.Client, containerName, blobName, sourceURL string) bool {
	ctx := context.TODO()
	dstContainerClient := client.NewContainerClient(containerName)
	_, err := dstContainerClient.Create(ctx, nil)
	if err != nil && !bloberror.HasCode(err, bloberror.ContainerAlreadyExists, bloberror.ResourceAlreadyExists) {
		return false
	}

	dstBlobClient := dstContainerClient.NewBlobClient(blobName)
	_, err = dstBlobClient.CopyFromURL(ctx, sourceURL, nil)
	return err == nil
}

func (a *API) CreateStorageAccount(resourceGroup string) (string, error) {
	ctx := context.TODO()
	// Only lower-case letters & numbers allowed in storage account names
	name := strings.Replace(randomName("kolasa"), "-", "", -1)
	parameters := armstorage.AccountCreateParameters{
		SKU: &armstorage.SKU{
			Name: to.Ptr(armstorage.SKUNameStandardLRS),
		},
		Kind:     to.Ptr(armstorage.KindStorageV2),
		Location: &a.Opts.Location,
		Properties: &armstorage.AccountPropertiesCreateParameters{
			AllowSharedKeyAccess: to.Ptr(false),
		},
	}
	if a.Opts.KolaVnet != "" {
		net, err := a.findVnetSubnet(a.Opts.KolaVnet)
		if err != nil {
			return "", fmt.Errorf("CreateStorageAccount: %v", err)
		}
		parameters.Properties.NetworkRuleSet = &armstorage.NetworkRuleSet{
			DefaultAction: to.Ptr(armstorage.DefaultActionDeny),
			VirtualNetworkRules: []*armstorage.VirtualNetworkRule{
				{
					VirtualNetworkResourceID: net.subnet.ID,
				},
			},
		}
	}

	plog.Infof("Creating StorageAccount %s", name)
	poller, err := a.accClient.BeginCreate(ctx, resourceGroup, name, parameters, nil)
	if err != nil {
		return "", fmt.Errorf("start creating storage account: %v", err)
	}
	r, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("finish creating storage account: %v", err)
	}
	if r.Name == nil {
		return name, nil
	}
	return *r.Name, nil
}
