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

package main

import (
	"net/url"
	"path"
	"strings"

	"github.com/spf13/pflag"

	"github.com/coreos/mantle/lang/maps"
	"github.com/coreos/mantle/sdk"
)

type storageSpec struct {
	BaseURL       string
	Title         string // Replace the bucket name in index page titles
	NamedPath     string // Copy to $BaseURL/$Board/$NamedPath
	VersionPath   bool   // Copy to $BaseURL/$Board/$Version
	DirectoryHTML bool
	IndexHTML     bool
}

type gceSpec struct {
	Project     string   // GCE project name
	Family      string   // A group name, also used as name prefix
	Description string   // Human readable-ish description
	Licenses    []string // Identifiers for tracking usage
	Image       string   // File name of image source
	Publish     string   // Write published image name to given file
	Limit       int      // Limit on # of old images to keep
}

type azureEnvironmentSpec struct {
	SubscriptionName     string   // Name of subscription in Azure profile
	AdditionalContainers []string // Extra containers to upload the disk image to
}

type azureSpec struct {
	Offer          string                 // Azure offer name
	Image          string                 // File name of image source
	StorageAccount string                 // Storage account to use for image uploads in each environment
	Container      string                 // Container to hold the disk image in each environment
	Environments   []azureEnvironmentSpec // Azure environments to upload to

	// Fields for azure.OSImage
	Label             string
	Description       string // Description of an image in this channel
	RecommendedVMSize string
	IconURI           string
	SmallIconURI      string
}

type awsPartitionSpec struct {
	Name              string   // Printable name for the partition
	Profile           string   // Authentication profile in ~/.aws
	Bucket            string   // S3 bucket for uploading image
	BucketRegion      string   // Region of the bucket
	LaunchPermissions []string // Other accounts to give launch permission
	Regions           []string // Regions to create the AMI in
}

type awsSpec struct {
	BaseName        string             // Prefix of image name
	BaseDescription string             // Prefix of image description
	Prefix          string             // Prefix for filenames of AMI lists
	Image           string             // File name of image source
	Partitions      []awsPartitionSpec // AWS partitions
}

type channelSpec struct {
	BaseURL      string // Copy from $BaseURL/$Board/$Version
	Boards       []string
	Destinations []storageSpec
	GCE          gceSpec
	Azure        azureSpec
	AWS          awsSpec
}

var (
	specBoard         string
	specChannel       string
	specVersion       string
	gceBoards         = []string{"amd64-usr"}
	azureBoards       = []string{"amd64-usr"}
	awsBoards         = []string{"amd64-usr"}
	azureEnvironments = []azureEnvironmentSpec{
		azureEnvironmentSpec{
			SubscriptionName:     "AzureCloud",
			AdditionalContainers: []string{"pre-publish"},
		},
		azureEnvironmentSpec{
			SubscriptionName: "BlackForest",
		},
		azureEnvironmentSpec{
			SubscriptionName: "Mooncake",
		},
	}
	awsPartitions = []awsPartitionSpec{
		awsPartitionSpec{
			Name:              "AWS",
			Profile:           "default",
			Bucket:            "flatcar-prod-ami-import-eu-central-1",
			BucketRegion:      "eu-central-1",
			LaunchPermissions: []string{},
			Regions: []string{
				"us-east-1",
				"us-east-2",
				"us-west-1",
				"us-west-2",
				"eu-west-1",
				"eu-west-2",
				"eu-west-3",
				"eu-central-1",
				"ap-south-1",
				"ap-southeast-1",
				"ap-southeast-2",
				"ap-northeast-1",
				"ap-northeast-2",
				"sa-east-1",
				"ca-central-1",
			},
		},
	}
	specs = map[string]channelSpec{
		"alpha": channelSpec{
			BaseURL:      "gs://flatcar-jenkins/alpha/boards",
			Boards:       []string{"amd64-usr"},
			Destinations: []storageSpec{},
			GCE:          gceSpec{},
			Azure: azureSpec{
				Offer:             "Flatcar",
				Image:             "flatcar_production_azure_image.vhd.bz2",
				StorageAccount:    "flatcar",
				Container:         "publish",
				Environments:      azureEnvironments,
				Label:             "Flatcar Alpha",
				Description:       "The Alpha channel closely tracks current development work and is released frequently. The newest versions of the Linux kernel, systemd, and other components will be available for testing.",
				RecommendedVMSize: "Medium",
				IconURI:           "coreos-globe-color-lg-100px.png",
				SmallIconURI:      "coreos-globe-color-lg-45px.png",
			},
			AWS: awsSpec{
				BaseName:        "Flatcar",
				BaseDescription: "Flatcar Linux",
				Prefix:          "flatcar_production_ami_",
				Image:           "flatcar_production_ami_vmdk_image.vmdk.bz2",
				Partitions:      awsPartitions,
			},
		},
		"beta": channelSpec{
			BaseURL:      "gs://flatcar-jenkins/beta/boards",
			Boards:       []string{"amd64-usr"},
			Destinations: []storageSpec{},
			GCE:          gceSpec{},
			Azure: azureSpec{
				Offer:             "Flatcar",
				Image:             "flatcar_production_azure_image.vhd.bz2",
				StorageAccount:    "flatcar",
				Container:         "publish",
				Environments:      azureEnvironments,
				Label:             "Flatcar Beta",
				Description:       "The Beta channel consists of promoted Alpha releases. Mix a few beta machines into your production clusters to catch any bugs specific to your hardware or configuration.",
				RecommendedVMSize: "Medium",
				IconURI:           "coreos-globe-color-lg-100px.png",
				SmallIconURI:      "coreos-globe-color-lg-45px.png",
			},
			AWS: awsSpec{
				BaseName:        "Flatcar",
				BaseDescription: "Flatcar Linux",
				Prefix:          "flatcar_production_ami_",
				Image:           "flatcar_production_ami_vmdk_image.vmdk.bz2",
				Partitions:      awsPartitions,
			},
		},
		"stable": channelSpec{
			BaseURL:      "gs://flatcar-jenkins/stable/boards",
			Boards:       []string{"amd64-usr"},
			Destinations: []storageSpec{},
			GCE:          gceSpec{},
			Azure: azureSpec{
				Offer:             "Flatcar",
				Image:             "flatcar_production_azure_image.vhd.bz2",
				StorageAccount:    "flatcar",
				Container:         "publish",
				Environments:      azureEnvironments,
				Label:             "Flatcar Stable",
				Description:       "The Stable channel should be used by production clusters. Versions of CoreOS Container Linux are battle-tested within the Beta and Alpha channels before being promoted.",
				RecommendedVMSize: "Medium",
				IconURI:           "coreos-globe-color-lg-100px.png",
				SmallIconURI:      "coreos-globe-color-lg-45px.png",
			},
			AWS: awsSpec{
				BaseName:        "Flatcar",
				BaseDescription: "Flatcar Linux",
				Prefix:          "flatcar_production_ami_",
				Image:           "flatcar_production_ami_vmdk_image.vmdk.bz2",
				Partitions:      awsPartitions,
			},
		},
	}
)

func AddSpecFlags(flags *pflag.FlagSet) {
	board := sdk.DefaultBoard()
	channels := strings.Join(maps.SortedKeys(specs), " ")
	versions, _ := sdk.VersionsFromManifest()
	flags.StringVarP(&specBoard, "board", "B",
		board, "target board")
	flags.StringVarP(&specChannel, "channel", "C",
		"user", "channels: "+channels)
	flags.StringVarP(&specVersion, "version", "V",
		versions.VersionID, "release version")
}

func ChannelSpec() channelSpec {
	if specBoard == "" {
		plog.Fatal("--board is required")
	}
	if specChannel == "" {
		plog.Fatal("--channel is required")
	}
	if specVersion == "" {
		plog.Fatal("--version is required")
	}

	spec, ok := specs[specChannel]
	if !ok {
		plog.Fatalf("Unknown channel: %s", specChannel)
	}

	boardOk := false
	for _, board := range spec.Boards {
		if specBoard == board {
			boardOk = true
			break
		}
	}
	if !boardOk {
		plog.Fatalf("Unknown board %q for channel %q", specBoard, specChannel)
	}

	gceOk := false
	for _, board := range gceBoards {
		if specBoard == board {
			gceOk = true
			break
		}
	}
	if !gceOk {
		spec.GCE = gceSpec{}
	}

	azureOk := false
	for _, board := range azureBoards {
		if specBoard == board {
			azureOk = true
			break
		}
	}
	if !azureOk {
		spec.Azure = azureSpec{}
	}

	awsOk := false
	for _, board := range awsBoards {
		if specBoard == board {
			awsOk = true
			break
		}
	}
	if !awsOk {
		spec.AWS = awsSpec{}
	}

	return spec
}

func (cs channelSpec) SourceURL() string {
	u, err := url.Parse(cs.BaseURL)
	if err != nil {
		panic(err)
	}
	u.Path = path.Join(u.Path, specBoard, specVersion)
	return u.String()
}

func (ss storageSpec) ParentPrefixes() []string {
	u, err := url.Parse(ss.BaseURL)
	if err != nil {
		panic(err)
	}
	return []string{u.Path, path.Join(u.Path, specBoard)}
}

func (ss storageSpec) FinalPrefixes() []string {
	u, err := url.Parse(ss.BaseURL)
	if err != nil {
		plog.Panic(err)
	}

	prefixes := []string{}
	if ss.VersionPath {
		prefixes = append(prefixes,
			path.Join(u.Path, specBoard, specVersion))
	}
	if ss.NamedPath != "" {
		prefixes = append(prefixes,
			path.Join(u.Path, specBoard, ss.NamedPath))
	}
	if len(prefixes) == 0 {
		plog.Panicf("Invalid destination: %#v", ss)
	}

	return prefixes
}
