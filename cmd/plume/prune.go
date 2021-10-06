// Copyright 2020 Kinvolk GmbH
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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	"github.com/flatcar-linux/mantle/platform/api/aws"
	"github.com/flatcar-linux/mantle/platform/api/azure"
)

var (
	days        int
	pruneDryRun bool
	cmdPrune    = &cobra.Command{
		Use:   "prune --channel CHANNEL [options]",
		Short: "Prune old release images for the given channel.",
		Run:   runPrune,
		Long:  `Prune old release images for the given channel.`,
	}
)

func init() {
	cmdPrune.Flags().IntVar(&days, "days", 30, "Minimum age in days for files to get deleted")
	cmdPrune.Flags().StringVar(&awsCredentialsFile, "aws-credentials", "", "AWS credentials file")
	cmdPrune.Flags().StringVar(&azureProfile, "azure-profile", "", "Azure Profile json file")
	cmdPrune.Flags().StringVar(&azureAuth, "azure-auth", "", "Azure Credentials json file")
	cmdPrune.Flags().StringVar(&azureTestContainer, "azure-test-container", "", "Use another container instead of the default")
	cmdPrune.Flags().BoolVarP(&pruneDryRun, "dry-run", "n", false,
		"perform a trial run, do not make changes")
	AddSpecFlags(cmdPrune.Flags())
	root.AddCommand(cmdPrune)
}

func runPrune(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		plog.Fatal("No args accepted")
	}

	// Override specVersion as it's not relevant for this command
	specVersion = "none"

	spec := ChannelSpec()
	ctx := context.Background()
	pruneAWS(ctx, &spec)
	pruneAzure(ctx, &spec)
}

func pruneAzure(ctx context.Context, spec *channelSpec) {
	if spec.Azure.StorageAccount == "" || azureProfile == "" {
		plog.Notice("Azure image pruning disabled, skipping.")
		return
	}

	for _, environment := range spec.Azure.Environments {
		api, err := azure.New(&azure.Options{
			AzureProfile:      azureProfile,
			AzureAuthLocation: azureAuth,
			AzureSubscription: environment.SubscriptionName,
		})
		if err != nil {
			plog.Fatalf("Failed to create Azure API: %v", err)
		}
		if err := api.SetupClients(); err != nil {
			plog.Fatalf("Failed to set up clients: %v", err)
		}

		plog.Printf("Fetching Azure storage credentials for %q in %q", spec.Azure.StorageAccount, spec.Azure.ResourceGroup)

		storageKey, err := api.GetStorageServiceKeysARM(spec.Azure.StorageAccount, spec.Azure.ResourceGroup)
		if err != nil {
			plog.Fatalf("Failed to fetch storage key: %v", err)
		}
		if storageKey.Keys == nil {
			plog.Fatalf("No storage service keys found")
		}

		container := spec.Azure.Container
		if azureTestContainer != "" {
			container = azureTestContainer
		}

		// Remove the compression extension from the filename, as Azure sets
		// the filename without the compression extension.
		specFileName := strings.TrimSuffix(spec.Azure.Image, filepath.Ext(spec.Azure.Image))

		for _, key := range *storageKey.Keys {
			blobs, err := api.ListBlobs(spec.Azure.StorageAccount, *key.Value, container, storage.ListBlobsParameters{})
			if err != nil {
				plog.Warningf("Error listing blobs: %v", err)
			}
			plog.Infof("Got %d blobs for container %q (key %v)", len(blobs), container, key)

			now := time.Now()
			for _, blob := range blobs {
				// Check that the blob's name includes the channel
				if !strings.Contains(blob.Name, specChannel) {
					plog.Infof("Blob's name %q doesn't include %q, skipping.", blob.Name, specChannel)
					continue
				}
				// Get the blob metadata and check that it's one of the release images
				var metadata map[string]map[string]interface{}
				json.Unmarshal([]byte(blob.Metadata["diskmetadata"]), &metadata)
				fileName := metadata["fileMetaData"]["fileName"]
				if fileName == nil {
					plog.Infof("No file name metadata for %q, skipping.", blob.Name)
					continue
				}
				if fileName != specFileName {
					plog.Infof("Blob's file name %q doesn't match %q, skipping.", fileName, specFileName)
					continue
				}
				// Get the last modified date and only delete obsolete blobs
				lastModifiedDate := time.Time(blob.Properties.LastModified)
				duration := now.Sub(lastModifiedDate)
				daysOld := int(duration.Hours() / 24)
				if daysOld < days {
					plog.Infof("Valid blob: %q: %d days old, skipping.", blob.Name, daysOld)
					continue
				}
				plog.Infof("Obsolete blob %q: %d days old", blob.Name, daysOld)
				if !pruneDryRun {
					plog.Infof("Deleting blob %q in container %q", blob.Name, container)
					err = api.DeleteBlob(spec.Azure.StorageAccount, *key.Value, container, blob.Name)
					if err != nil {
						plog.Warningf("Error deleting blob (%v): %v", blob.Name, err)
					}
				}
			}
		}
	}
}

func pruneAWS(ctx context.Context, spec *channelSpec) {
	if spec.AWS.Image == "" || awsCredentialsFile == "" {
		plog.Notice("AWS image pruning disabled.")
		return
	}

	// Iterate over all partitions and regions in the given channel and prune
	// images in each of them.
	for _, part := range spec.AWS.Partitions {
		for _, region := range part.Regions {
			if pruneDryRun {
				plog.Printf("Checking for images in %v...", part.Name)
			} else {
				plog.Printf("Pruning images in %v...", part.Name)
			}

			api, err := aws.New(&aws.Options{
				CredentialsFile: awsCredentialsFile,
				Profile:         part.Profile,
				Region:          region,
			})
			if err != nil {
				plog.Fatalf("Creating client for %v %v: %v", part.Name, region, err)
			}

			images, err := api.GetImagesByTag("Channel", specChannel)
			if err != nil {
				plog.Fatalf("Couldn't list images in channel %q: %v", specChannel, err)
			}

			plog.Infof("Got %d images with channel %q", len(images), specChannel)

			now := time.Now()
			for _, image := range images {
				creationDate, err := time.Parse(time.RFC3339Nano, *image.CreationDate)
				if err != nil {
					plog.Warningf("Error converting creation date (%v): %v", *image.CreationDate, err)
				}
				duration := now.Sub(creationDate)
				daysOld := int(duration.Hours() / 24)
				if daysOld < days {
					plog.Infof("Valid image %q: %d days old, skipping", *image.Name, daysOld)
					continue
				}
				plog.Infof("Obsolete image %q: %d days old", *image.Name, daysOld)
				if !pruneDryRun {
					// Construct the s3ObjectPath in the same manner it's constructed for upload
					arch := *image.Architecture
					if arch == "x86_64" {
						arch = "amd64"
					}
					board := fmt.Sprintf("%s-usr", arch)
					var version string
					for _, t := range image.Tags {
						if *t.Key == "Version" {
							version = *t.Value
						}
					}
					imageFileName := strings.TrimSuffix(spec.AWS.Image, filepath.Ext(spec.AWS.Image))
					s3ObjectPath := fmt.Sprintf("%s/%s/%s", board, version, imageFileName)

					// Remove -hvm from the name, as the snapshots don't include that.
					imageName := strings.TrimSuffix(*image.Name, "-hvm")

					err := api.RemoveImage(imageName, imageName, part.Bucket, s3ObjectPath, nil)
					if err != nil {
						plog.Fatalf("couldn't prune image %v: %v", *image.Name, err)
					}
				}
			}
		}
	}
}
