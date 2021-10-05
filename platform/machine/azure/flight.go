// Copyright 2018 Red Hat
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
	"fmt"
	"time"

	ctplatform "github.com/coreos/container-linux-config-transpiler/config/platform"
	"github.com/coreos/pkg/capnslog"

	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/azure"
)

const (
	Platform platform.Name = "azure"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "platform/machine/azure")
)

type flight struct {
	*platform.BaseFlight
	api                 *azure.API
	SSHKey              string
	FakeSSHKey          string
	ImageResourceGroup  string
	ImageStorageAccount string
}

// NewFlight creates an instance of a Flight suitable for spawning
// instances on the Azure platform. The flight creates a new Resource Group
// if an image needs to be created from a blob URL or a local image file.
// Clusters created in the Flight will reuse the image Resource Group or
// create their own Resource Group if no image Resource Group exists.
func NewFlight(opts *azure.Options) (platform.Flight, error) {
	api, err := azure.New(opts)
	if err != nil {
		return nil, err
	}

	if err = api.SetupClients(); err != nil {
		return nil, fmt.Errorf("setting up clients: %v", err)
	}

	bf, err := platform.NewBaseFlight(opts.Options, Platform, ctplatform.Azure)
	if err != nil {
		return nil, err
	}

	af := &flight{
		BaseFlight: bf,
		api:        api,
	}

	keys, err := af.Keys()
	if err != nil {
		af.Destroy()
		return nil, err
	}
	af.SSHKey = keys[0].String()
	af.FakeSSHKey, err = platform.GenerateFakeKey()
	if err != nil {
		return nil, err
	}

	if opts.BlobURL != "" || opts.ImageFile != "" {
		imageName := fmt.Sprintf("%v", time.Now().UnixNano())
		blobName := imageName + ".vhd"
		container := "temp"

		af.ImageResourceGroup, err = af.api.CreateResourceGroup("kola-cluster-image")
		if err != nil {
			return nil, err
		}

		af.ImageStorageAccount, err = af.api.CreateStorageAccount(af.ImageResourceGroup)
		if err != nil {
			return nil, err
		}

		_, err = af.api.PrepareNetworkResources(af.ImageResourceGroup)
		if err != nil {
			return nil, err
		}

		kr, err := af.api.GetStorageServiceKeysARM(af.ImageStorageAccount, af.ImageResourceGroup)
		if err != nil {
			return nil, fmt.Errorf("Fetching storage service keys failed: %v", err)
		}

		if kr.Keys == nil {
			return nil, fmt.Errorf("No storage service keys found")
		}

		if opts.BlobURL != "" {
			for _, k := range *kr.Keys {
				plog.Infof("Copying blob")
				if err := af.api.CopyBlob(af.ImageStorageAccount, *k.Value, container, blobName, opts.BlobURL); err != nil {
					return nil, fmt.Errorf("Copying blob failed: %v", err)
				}
				plog.Infof("Blob copy done")
				break
			}
		} else if opts.ImageFile != "" {
			for _, k := range *kr.Keys {
				if err := af.api.UploadBlob(af.ImageStorageAccount, *k.Value, opts.ImageFile, container, blobName, true); err != nil {
					return nil, fmt.Errorf("Uploading blob failed: %v", err)
				}
				break
			}
		}
		targetBlobURL := af.api.UrlOfBlob(af.ImageStorageAccount, container, blobName).String()
		img, err := af.api.CreateImage(imageName, af.ImageResourceGroup, targetBlobURL)
		if err != nil {
			return nil, fmt.Errorf("Couldn't create image: %v\n", err)
		}
		if img.ID == nil {
			return nil, fmt.Errorf("received nil image\n")
		}
		opts.DiskURI = *img.ID
	}

	return af, nil
}

// NewCluster creates an instance of a Cluster suitable for spawning
// instances on the Azure platform. The cluster is created in the Flight's
// Resource Group if it has one. Otherwise the cluster is created in a new
// Resource Group that is deleted when the cluster is destroyed.
func (af *flight) NewCluster(rconf *platform.RuntimeConfig) (platform.Cluster, error) {
	bc, err := platform.NewBaseCluster(af.BaseFlight, rconf)
	if err != nil {
		return nil, err
	}

	ac := &cluster{
		BaseCluster: bc,
		flight:      af,
	}

	if !rconf.NoSSHKeyInMetadata {
		ac.sshKey = af.SSHKey
	} else {
		ac.sshKey = af.FakeSSHKey
	}

	if af.ImageResourceGroup != "" && af.ImageStorageAccount != "" {
		ac.ResourceGroup = af.ImageResourceGroup
		ac.StorageAccount = af.ImageStorageAccount

	} else {
		ac.ResourceGroup, err = af.api.CreateResourceGroup("kola-cluster")
		if err != nil {
			return nil, err
		}

		ac.StorageAccount, err = af.api.CreateStorageAccount(ac.ResourceGroup)
		if err != nil {
			return nil, err
		}

		_, err = af.api.PrepareNetworkResources(ac.ResourceGroup)
		if err != nil {
			return nil, err
		}
	}

	af.AddCluster(ac)

	return ac, nil
}

func (af *flight) Destroy() {
	af.BaseFlight.Destroy()

	if af.ImageResourceGroup != "" {
		if e := af.api.TerminateResourceGroup(af.ImageResourceGroup); e != nil {
			plog.Errorf("Deleting image resource group %v: %v", af.ImageResourceGroup, e)
		}
	}
}
