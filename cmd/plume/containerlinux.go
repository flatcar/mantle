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
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/spf13/pflag"

	"github.com/flatcar/mantle/lang/maps"
	"github.com/flatcar/mantle/sdk"
)

var (
	specBoard         string
	specChannel       string
	specVersion       string
	specAwsPartition  string
	specPrivateBucket bool
	gceBoards         = []string{"amd64-usr", "arm64-usr"}
	azureBoards       = []string{"amd64-usr", "arm64-usr"}
	awsBoards         = []string{"amd64-usr", "arm64-usr"}
	azureEnvironments = []azureEnvironmentSpec{
		azureEnvironmentSpec{
			CloudName: "public",
		},
	}
	awsPartitions = map[string]awsPartitionSpec{
		"default": awsPartitionSpec{
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
				"eu-north-1",
				"eu-south-1",
				"eu-central-1",
				"ap-south-1",
				"ap-southeast-1",
				"ap-southeast-2",
				"ap-southeast-3",
				"ap-northeast-1",
				"ap-northeast-2",
				"af-south-1",
				// "ap-northeast-3", // Disabled for now because we do not have access
				"sa-east-1",
				"ca-central-1",
				"ap-east-1",
				"me-south-1",
			},
		},
		"china": awsPartitionSpec{
			Name:         "AWS China",
			Profile:      "china",
			Bucket:       "flatcar-prod-ami-import-cn-north-1",
			BucketRegion: "cn-north-1",
			Regions: []string{
				"cn-north-1",
				"cn-northwest-1",
			},
		},
		"developer": awsPartitionSpec{
			Name:         "AWS Developer",
			Profile:      "default",
			Bucket:       "flatcar-developer-ami-import-us-west-2",
			BucketRegion: "us-west-2",
			Regions: []string{
				"us-west-2",
			},
		},
	}

	alpha_desc  = "The Alpha channel closely tracks current development work and is released frequently. The newest versions of the Linux kernel, systemd, and other components will be available for testing."
	beta_desc   = "The Beta channel consists of promoted Alpha releases. Mix a few beta machines into your production clusters to catch any bugs specific to your hardware or configuration."
	stable_desc = "The Stable channel should be used by production clusters. Versions of Flatcar Container Linux are battle-tested within the Beta and Alpha channels before being promoted."
	edge_desc   = "The Edge channel closely tracks current development work and is released frequently. The newest versions of the Linux kernel, systemd, and other components will be available for testing."
	lts_desc    = "The LTS channel should be used by production clusters. Versions of Flatcar Container Linux are battle-tested within the Stable channel before being promoted."
	dev_desc    = "The Developer Channel is used for internal test builds."

	specs = map[string]channelSpec{
		"alpha": channelSpec{
			BaseURL:      "http://bincache.flatcar-linux.net/images",
			Boards:       []string{"amd64-usr", "arm64-usr"},
			Destinations: []storageSpec{},
			GCE:          newGceSpec("alpha", alpha_desc),
			Azure:        newAzureSpec(azureEnvironments, "publish", "Flatcar Alpha", "", alpha_desc),
			AzurePremium: newAzureSpec(azureEnvironments, "publish", "Flatcar Alpha", "", alpha_desc),
			AWS:          newAWSSpec(),
		},
		"beta": channelSpec{
			BaseURL:      "http://bincache.flatcar-linux.net/images",
			Boards:       []string{"amd64-usr", "arm64-usr"},
			Destinations: []storageSpec{},
			GCE:          newGceSpec("beta", beta_desc),
			Azure:        newAzureSpec(azureEnvironments, "publish", "Flatcar Beta", "", beta_desc),
			AzurePremium: newAzureSpec(azureEnvironments, "publish", "Flatcar Beta", "", beta_desc),
			AWS:          newAWSSpec(),
		},
		"stable": channelSpec{
			BaseURL:      "http://bincache.flatcar-linux.net/images",
			Boards:       []string{"amd64-usr", "arm64-usr"},
			Destinations: []storageSpec{},
			GCE:          newGceSpec("stable", stable_desc),
			Azure:        newAzureSpec(azureEnvironments, "publish", "Flatcar Stable", "", stable_desc),
			AzurePremium: newAzureSpec(azureEnvironments, "publish", "Flatcar Stable", "", stable_desc),
			AWS:          newAWSSpec(),
		},
		"edge": channelSpec{
			BaseURL:      "http://bincache.flatcar-linux.net/images",
			Boards:       []string{"amd64-usr", "arm64-usr"},
			Destinations: []storageSpec{},
			GCE:          newGceSpec("edge", edge_desc),
			Azure:        newAzureSpec(azureEnvironments, "publish", "Flatcar Edge", "", edge_desc),
			AzurePremium: newAzureSpec(azureEnvironments, "publish", "Flatcar Edge", "", edge_desc),
			AWS:          newAWSSpec(),
		},
		"lts": channelSpec{
			BaseURL:      "http://bincache.flatcar-linux.net/images",
			Boards:       []string{"amd64-usr", "arm64-usr"},
			Destinations: []storageSpec{},
			GCE:          newGceSpec("lts", lts_desc),
			Azure:        newAzureSpec(azureEnvironments, "publish", "Flatcar LTS", "", lts_desc),
			AzurePremium: newAzureSpec(azureEnvironments, "publish", "Flatcar LTS", "", lts_desc),
			AWS:          newAWSSpec(),
		},
		"developer": channelSpec{
			BaseURL:      "http://bincache.flatcar-linux.net/images",
			Boards:       []string{"amd64-usr", "arm64-usr"},
			Destinations: []storageSpec{},
			GCE:          gceSpec{},
			Azure:        newAzureSpec(azureEnvironments, "developer", "Flatcar Developer Channel", "", dev_desc),
			AzurePremium: newAzureSpec(azureEnvironments, "developer", "Flatcar Developer Channel", "", dev_desc),
			AWS:          newAWSSpec(),
		},
	}
)

func newGceSpec(channel, description string) gceSpec {
	return gceSpec{
		Project:     "kinvolk-public",
		Family:      fmt.Sprintf("flatcar-%s", channel),
		Description: description,
		Licenses:    []string{"flatcar-container-linux"},
		Image:       "flatcar_production_gce.tar.gz",
		Publish:     "",
		Limit:       10,
	}
}

func newAzureSpec(environments []azureEnvironmentSpec, container, label, category string, description string) azureSpec {
	return azureSpec{
		Offer:             "Flatcar",
		Image:             fmt.Sprintf("flatcar_production_azure%s_image.vhd.bz2", category),
		StorageAccount:    "flatcar",
		ResourceGroup:     "flatcar",
		Container:         container,
		Environments:      environments,
		Label:             label,
		Description:       description,
		RecommendedVMSize: "Medium",
		IconURI:           "coreos-globe-color-lg-100px.png",
		SmallIconURI:      "coreos-globe-color-lg-45px.png",
	}
}

func newAWSSpec() awsSpec {
	return awsSpec{
		BaseName:        "Flatcar",
		BaseDescription: "Flatcar Container Linux",
		Prefix:          "flatcar_production_ami_",
		Image:           "flatcar_production_ami_image.bin.bz2",
	}
}

func AddSpecFlags(flags *pflag.FlagSet) {
	board := sdk.DefaultBoard()
	channels := strings.Join(maps.SortedKeys(specs), " ")
	versions, _ := sdk.VersionsFromManifest()
	awsPartition := "default"
	flags.StringVarP(&specBoard, "board", "B",
		board, "target board")
	flags.StringVarP(&specChannel, "channel", "C",
		"user", "channels: "+channels)
	flags.StringVarP(&specVersion, "version", "V",
		versions.VersionID, "release version")
	flags.StringVarP(&specAwsPartition, "partition", "P",
		awsPartition, "aws partition")
	flags.BoolVarP(&specPrivateBucket, "private", "Z",
		false, "Private GCE Bucket")
}

func AmiNameArchTag() string {
	switch specBoard {
	case "amd64-usr":
		return ""
	case "arm64-usr":
		return "-arm64"
	default:
		plog.Fatalf("No AMI name architecture tag defined for board %q", specBoard)
		return "" // dummy
	}
}

func AzureBlobName() string {
	archTag := ""
	switch specBoard {
	case "amd64-usr":
		archTag = "amd64"
	case "arm64-usr":
		archTag = "arm64"
	}
	return fmt.Sprintf("flatcar-linux-%s-%s-%s.vhd", specVersion, specChannel, archTag)
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
	if specAwsPartition == "" {
		plog.Fatal("--partition is required")
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

	// For the developer channel, use the developer partition
	if specChannel == "developer" {
		specAwsPartition = "developer"
	}

	awsPartition, awsPartitionOk := awsPartitions[specAwsPartition]
	if !awsPartitionOk {
		plog.Fatalf("Unknown AWS Partition: %s", specAwsPartition)
	}
	spec.AWS.Partitions = []awsPartitionSpec{awsPartition}

	return spec
}

func (cs channelSpec) SourceURL() string {
	baseURL := cs.BaseURL
	if gceJSONKeyFile == "none" {
		baseURL = strings.Replace(baseURL, "gs://", "https://bucket.release.flatcar-linux.net/", 1)
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		panic(err)
	}

	// We conditionnally drop the '-usr' of the board based on the
	// URL scheme of the BaseURL.
	arch := specBoard
	if u.Scheme != "gs" {
		arch = strings.TrimSuffix(specBoard, "-usr")
	}

	u.Path = path.Join(u.Path, arch, specVersion)
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
