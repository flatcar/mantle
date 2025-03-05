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

	"github.com/coreos/pkg/capnslog"
	ctplatform "github.com/flatcar/container-linux-config-transpiler/config/platform"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/azure"
)

const (
	Platform platform.Name = "azure"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/machine/azure")
)

type flight struct {
	*platform.BaseFlight
	Api                 *azure.API
	SSHKey              string
	FakeSSHKey          string
	ImageResourceGroup  string
	ImageStorageAccount string
	Network             azure.Network
	UseFlightRG         bool
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
		Api:        api,
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

	af.ImageResourceGroup, err = af.Api.CreateResourceGroup("kola-cluster-image")
	if err != nil {
		return nil, err
	}
	af.ImageStorageAccount, err = af.Api.CreateStorageAccount(af.ImageResourceGroup)
	if err != nil {
		return nil, err
	}

	if opts.BlobURL != "" || opts.ImageFile != "" {
		af.UseFlightRG = true
		imageName := fmt.Sprintf("%v", time.Now().UnixNano())
		blobName := imageName + ".vhd"
		container := "temp"

		af.Network, err = af.Api.PrepareNetworkResources(af.ImageResourceGroup)
		if err != nil {
			af.Destroy()
			return nil, err
		}

		client, err := af.Api.GetBlobServiceClient(af.ImageStorageAccount)
		if err != nil {
			return nil, fmt.Errorf("failed to create blob service client for %q: %w", af.ImageStorageAccount, err)
		}

		if opts.BlobURL != "" {
			plog.Infof("Copying blob")
			if err := azure.CopyBlob(client, container, blobName, opts.BlobURL); err != nil {
				return nil, fmt.Errorf("Copying blob failed: %v", err)
			}
			plog.Infof("Blob copy done")
		} else if opts.ImageFile != "" {
			if err := azure.UploadBlob(client, opts.ImageFile, container, blobName, true); err != nil {
				return nil, fmt.Errorf("Uploading blob failed: %v", err)
			}
		}
		targetBlobURL := azure.BlobURL(client, container, blobName)
		var imgID string
		if opts.UseGallery {
			imgID, err = af.Api.CreateGalleryImage(imageName, af.ImageResourceGroup, af.ImageStorageAccount, targetBlobURL)
			if err != nil {
				return nil, fmt.Errorf("couldn't create gallery image: %w", err)
			}
			plog.Infof("Created gallery image: %v\n", imgID)
		} else {
			img, err := af.Api.CreateImage(imageName, af.ImageResourceGroup, targetBlobURL)
			if err != nil {
				return nil, fmt.Errorf("couldn't create image: %w", err)
			}
			if img.ID == nil {
				return nil, fmt.Errorf("received nil image")
			}
			imgID = *img.ID
		}

		opts.DiskURI = imgID
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

	ac.StorageAccountRG = af.ImageResourceGroup
	ac.StorageAccount = af.ImageStorageAccount

	if af.UseFlightRG {
		ac.ResourceGroup = af.ImageResourceGroup
		ac.Network = af.Network
	} else {
		ac.ResourceGroup, err = af.Api.CreateResourceGroup("kola-cluster")
		if err != nil {
			return nil, err
		}
		ac.Network, err = af.Api.PrepareNetworkResources(ac.ResourceGroup)
		if err != nil {
			ac.Destroy()
			return nil, err
		}
	}

	af.AddCluster(ac)

	return ac, nil
}

func (af *flight) Destroy() {
	af.BaseFlight.Destroy()

	if af.ImageResourceGroup != "" {
		if e := af.Api.TerminateResourceGroup(af.ImageResourceGroup); e != nil {
			plog.Errorf("Deleting image resource group %v: %v", af.ImageResourceGroup, e)
		}
	}
}
