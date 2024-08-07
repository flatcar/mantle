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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flatcar/azure-vhd-utils/op"
	"github.com/flatcar/mantle/platform/api/azure"
	"github.com/flatcar/mantle/sdk"
	"github.com/spf13/cobra"
)

var (
	cmdCreateGalleryImage = &cobra.Command{
		Use:   "create-gallery-image",
		Short: "Create Azure Gallery Image",
		Long:  "Create Azure Gallery Image mage from a VHD image",
		RunE:  runCreateGalleryImage,
	}

	vhd              string
	blobName         string
	storageAccount   string
	resourceGrp      string
	hyperVGeneration string
	board            string
	dbFile           string
)

func init() {
	sv := cmdCreateGalleryImage.Flags().StringVar

	sv(&imageName, "image-name", "", "image name (optional)")
	sv(&blobName, "blob-name", "", "source blob name (optional)")
	sv(&vhd, "file", defaultUploadFile(), "source VHD file")
	sv(&resourceGrp, "resource-group", "", "resource group name (optional)")
	sv(&hyperVGeneration, "hyper-v-generation", "V2", "Hyper-V generation (V2 or V1)")
	sv(&board, "board", "amd64-usr", "board name (amd64-usr or arm64-usr)")
	sv(&storageAccount, "storage-account", "", "storage account name (optional)")
	sv(&dbFile, "db-file", "", "path to the DB secure boot certificate (optional)")

	Azure.AddCommand(cmdCreateGalleryImage)
}

func readDbString(fname string) (string, error) {
	buf, err := os.ReadFile(fname)
	if err != nil {
		return "", err
	}
	db := ""
	for _, str := range strings.Split(string(buf), "\n") {
		if strings.Contains(str, "-----BEGIN CERTIFICATE-----") {
			continue
		}
		if strings.Contains(str, "-----END CERTIFICATE-----") {
			continue
		}
		db += str
	}
	return db, nil
}

func azureSanitize(name string) string {
	name = strings.Replace(name, ".", "-", -1)
	name = strings.Replace(name, "+", "-", -1)
	return name
}

func runCreateGalleryImage(cmd *cobra.Command, args []string) error {
	var err error
	if err = api.SetupClients(); err != nil {
		plog.Fatalf("setting up clients: %v\n", err)
	}
	api.Opts.Board = board
	api.Opts.HyperVGeneration = hyperVGeneration

	var db string
	if dbFile != "" {
		db, err = readDbString(dbFile)
		if err != nil {
			plog.Fatalf("failed to read db file: %v", err)
		}
	}
	if blobName == "" {
		ver, err := sdk.VersionsFromDir(filepath.Dir(vhd))
		if err != nil {
			plog.Fatalf("Unable to get version from image directory, provide a -blob-name flag or include a version.txt in the image directory: %v\n", err)
		}
		blobName = fmt.Sprintf("flatcar-dev-%s-%s.vhd", os.Getenv("USER"), ver.Version)
	}
	if imageName == "" {
		imageName = azureSanitize(strings.TrimSuffix(blobName, ".vhd"))
	}
	if resourceGrp == "" {
		resourceGrp, err = api.CreateResourceGroup("kola-cluster-image")
		if err != nil {
			plog.Fatalf("Couldn't create resource group: %v\n", err)
		}
	}
	if storageAccount == "" {
		storageAccount, err = api.CreateStorageAccount(resourceGrp)
		if err != nil {
			plog.Fatalf("Couldn't create storage account: %v\n", err)
		}
	}
	client, err := api.GetBlobServiceClient(storageAccount)
	if err != nil {
		plog.Fatalf("failed to create blob service client for %q: %v", ubo.storageacct, err)
	}

	container := "vhds"
	if err := azure.UploadBlob(client, vhd, container, blobName, false); err != nil {
		var operr op.Error
		if errors.As(err, &operr) && operr == op.BlobAlreadyExists {
			plog.Noticef("Blob %q already exists, skipping upload", blobName)
		} else {
			plog.Fatalf("Uploading blob failed: %v", err)
		}
	}
	blobUrl := azure.BlobURL(client, container, blobName)
	imgID, err := api.CreateGalleryImage(imageName, resourceGrp, storageAccount, blobUrl, db)
	if err != nil {
		plog.Fatalf("Couldn't create gallery image: %v\n", err)
	}
	err = json.NewEncoder(os.Stdout).Encode(&struct {
		ID *string
	}{
		ID: &imgID,
	})
	if err != nil {
		plog.Fatalf("Couldn't encode result: %v\n", err)
	}
	return nil
}
