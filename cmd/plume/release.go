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
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"google.golang.org/api/compute/v1"
	gs "google.golang.org/api/storage/v1"

	"github.com/flatcar/mantle/platform/api/aws"
	"github.com/flatcar/mantle/platform/api/azure"
	"github.com/flatcar/mantle/platform/api/gcloud"
	"github.com/flatcar/mantle/sdk"
	"github.com/flatcar/mantle/storage"
	"github.com/flatcar/mantle/storage/index"
)

var (
	releaseDryRun bool
	cmdRelease    = &cobra.Command{
		Use:   "release [options]",
		Short: "Publish a new Flatcar release.",
		Run:   runRelease,
		Long:  `Publish a new Flatcar release.`,
	}
	gceReleaseKey string
)

func init() {
	cmdRelease.Flags().StringVar(&awsCredentialsFile, "aws-credentials", "", "AWS credentials file")
	cmdRelease.Flags().StringVar(&selectedDistro, "distro", "cl", "DEPRECATED - system to release")
	cmdRelease.Flags().StringVar(&azureProfile, "azure-profile", "", "Azure Profile json file")
	cmdRelease.Flags().StringVar(&azureAuth, "azure-auth", "", "Azure Credentials json file")
	cmdRelease.Flags().StringVar(&azureTestContainer, "azure-test-container", "", "Use test container instead of default")
	cmdRelease.Flags().BoolVar(&azureUseIdentity, "azure-identity", false, "Use VM managed identity for authentication (default false)")
	cmdRelease.Flags().StringVar(&gceReleaseKey, "gce-release-key", "", "GCE key file for releases")
	cmdRelease.Flags().BoolVarP(&releaseDryRun, "dry-run", "n", false,
		"perform a trial run, do not make changes")
	cmdRelease.Flags().BoolVarP(&publishMarketplace, "publish-marketplace", "", false,
		"publish on the AWS marketplace")
	cmdRelease.Flags().StringVar(&accessRoleARN, "access-role-arn", "", "ARN to give marketplace access to the AMI")
	cmdRelease.Flags().StringSliceVar(&productIDs, "product-ids", []string{}, "AWS Marketplace offer IDs")
	cmdRelease.Flags().StringVar(&awsMarketplaceCredentialsFile, "aws-marketplace-credentials", "", "AWS Marketplace credentials file")
	cmdRelease.Flags().StringVar(&username, "username", "core", "default username")
	AddSpecFlags(cmdRelease.Flags())
	root.AddCommand(cmdRelease)
}

func runRelease(cmd *cobra.Command, args []string) {
	if err := runCLRelease(cmd, args); err != nil {
		plog.Fatal(err)
	}
}

func runCLRelease(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		plog.Fatal("No args accepted")
	}

	spec := ChannelSpec()
	ctx := context.Background()
	client, err := getGoogleClient()
	if err != nil {
		plog.Fatalf("Authentication failed: %v", err)
	}

	src, err := storage.NewBucket(client, spec.SourceURL())
	if err != nil {
		plog.Fatal(err)
	}
	src.WriteDryRun(releaseDryRun)

	if err := src.Fetch(ctx); err != nil && !strings.HasPrefix(spec.SourceURL(), "http") {
		plog.Fatal(err)
	}

	// Sanity check!
	if vertxt := src.Object(src.Prefix() + "version.txt"); vertxt == nil {
		verurl := src.URL().String() + "version.txt"
		plog.Fatalf("File not found: %s", verurl)
	}

	// We do not provide yet ARM64 image for Google.
	if specBoard == "amd64-usr" {
		// Create a GCS bucket client to temporary upload the GCE image on GCS.
		gcs, err := storage.NewBucket(client, "gs://flatcar-jenkins")
		if err != nil {
			plog.Fatalf("creating GCE bucket client: %v", err)
		}

		doGCE(ctx, client, gcs, &spec)
	}

	// Make Azure images public.
	doAzure(ctx, client, src, &spec)

	// Make AWS images public.
	doAWS(ctx, client, src, &spec)

	for _, dSpec := range spec.Destinations {
		dst, err := storage.NewBucket(client, dSpec.BaseURL)
		if err != nil {
			plog.Fatal(err)
		}
		dst.WriteDryRun(releaseDryRun)

		// Fetch parent directories non-recursively to re-index it later.
		for _, prefix := range dSpec.ParentPrefixes() {
			if err := dst.FetchPrefix(ctx, prefix, false); err != nil {
				plog.Fatal(err)
			}
		}

		// Fetch and sync each destination directory.
		for _, prefix := range dSpec.FinalPrefixes() {
			if err := dst.FetchPrefix(ctx, prefix, true); err != nil {
				plog.Fatal(err)
			}

			sync := index.NewSyncIndexJob(src, dst)
			sync.DestinationPrefix(prefix)
			sync.DirectoryHTML(dSpec.DirectoryHTML)
			sync.IndexHTML(dSpec.IndexHTML)
			sync.Delete(true)
			if dSpec.Title != "" {
				sync.Name(dSpec.Title)
			}
			if err := sync.Do(ctx); err != nil {
				plog.Fatal(err)
			}
		}

		// Now refresh the parent directory indexes.
		for _, prefix := range dSpec.ParentPrefixes() {
			parent := index.NewIndexJob(dst)
			parent.Prefix(prefix)
			parent.DirectoryHTML(dSpec.DirectoryHTML)
			parent.IndexHTML(dSpec.IndexHTML)
			parent.Recursive(false)
			parent.Delete(true)
			if dSpec.Title != "" {
				parent.Name(dSpec.Title)
			}
			if err := parent.Do(ctx); err != nil {
				plog.Fatal(err)
			}
		}
	}

	return nil
}

func sanitizeVersion() string {
	v := strings.Replace(specVersion, ".", "-", -1)
	return strings.Replace(v, "+", "-", -1)
}

func gceWaitForImage(pending *gcloud.Pending) {
	plog.Infof("Waiting for image creation to finish...")
	pending.Interval = 3 * time.Second
	pending.Progress = func(_ string, _ time.Duration, op *compute.Operation) error {
		status := strings.ToLower(op.Status)
		if op.Progress != 0 {
			plog.Infof("Image creation is %s: %s % 2d%%", status, op.StatusMessage, op.Progress)
		} else {
			plog.Infof("Image creation is %s. %s", status, op.StatusMessage)
		}
		return nil
	}
	if err := pending.Wait(); err != nil {
		plog.Fatal(err)
	}
	plog.Info("Success!")
}

func gceUploadImage(spec *channelSpec, api *gcloud.API, obj *gs.Object, name, desc string) string {
	plog.Noticef("Creating GCE image %s", name)
	// Overwrite is set
	op, pending, err := api.CreateImage(&gcloud.ImageSpec{
		SourceImage: obj.MediaLink,
		Family:      spec.GCE.Family,
		Name:        name,
		Description: desc,
		Licenses:    spec.GCE.Licenses,
	}, true)
	if err != nil {
		plog.Fatalf("GCE image creation failed: %v", err)
	}

	gceWaitForImage(pending)

	return op.TargetLink
}

func doGCE(ctx context.Context, client *http.Client, src *storage.Bucket, spec *channelSpec) {
	if spec.GCE.Project == "" || spec.GCE.Image == "" {
		plog.Notice("GCE image creation disabled.")
		return
	}

	if gceReleaseKey == "" {
		plog.Notice("No GCE Release key file defined, skipping.")
		return
	}

	api, err := gcloud.New(&gcloud.Options{
		Project:     spec.GCE.Project,
		JSONKeyFile: gceReleaseKey,
	})
	if err != nil {
		plog.Fatalf("GCE client failed: %v", err)
	}

	name := fmt.Sprintf("%s-%s", spec.GCE.Family, sanitizeVersion())

	// We extend the name with '-arm64' suffix to avoid conflicting image name.
	if specBoard == "arm64-usr" {
		name = name + "-arm64"
	}

	date := time.Now().UTC()
	desc := fmt.Sprintf("%s, %s, %s published on %s", spec.GCE.Description,
		specVersion, specBoard, date.Format("2006-01-02"))

	images, err := api.ListImages(ctx, spec.GCE.Family+"-")
	if err != nil {
		plog.Fatal(err)
	}

	var oldImages []*compute.Image
	for _, image := range images {
		if !strings.HasPrefix(image.Name, name) {
			oldImages = append(oldImages, image)
		}
	}
	sort.Slice(oldImages, func(i, j int) bool {
		getCreation := func(image *compute.Image) time.Time {
			stamp, err := time.Parse(time.RFC3339, image.CreationTimestamp)
			if err != nil {
				plog.Fatalf("Couldn't parse timestamp %q: %v", image.CreationTimestamp, err)
			}
			return stamp
		}
		return getCreation(oldImages[i]).After(getCreation(oldImages[j]))
	})

	// Prepare the URL to temporary store the downloaded GCE image.
	gsURL, err := url.Parse(src.Prefix())
	if err != nil {
		plog.Fatalf("parsing GCS prefix URL: %v", err)
	}

	gsURL = gsURL.JoinPath(specChannel, "boards", specBoard, specVersion, spec.GCE.Image)

	// Check for any with the same version but possibly different dates.
	var imageLink string
	if releaseDryRun {
		plog.Noticef("Would create GCE image %s", name)
		return
	}

	// Download the image from the webserver, to temporary upload it on GCS.
	// To create the Image on GCE, it's required to have a GCS URL.
	imgURL, err := url.Parse(spec.SourceURL())
	if err != nil {
		plog.Fatalf("parsing webserver source URL: %v", err)
	}

	imgURL = imgURL.JoinPath(spec.GCE.Image)

	// verify key is set to "" to use the embedded one.
	if err := sdk.DownloadSignedFile(spec.GCE.Image, imgURL.String(), &http.Client{}, ""); err != nil {
		plog.Fatalf("downloading GCE image from webserver: %v", err)
	}

	f, err := os.Open(spec.GCE.Image)
	if err != nil {
		plog.Fatalf("opening GCE image: %v", err)
	}

	defer f.Close()

	o := gs.Object{Name: gsURL.String()}

	// Required to overwrite an existing image.
	src.WriteAlways(true)

	if err := src.Upload(ctx, &o, f); err != nil {
		plog.Fatalf("uploading GCE image to GCS: %v", err)
	}

	obj := src.Object(gsURL.String())
	if obj == nil {
		plog.Fatalf("GCE image not found %s%s", src.URL(), spec.GCE.Image)
	}

	imageLink = gceUploadImage(spec, api, obj, name, desc)

	// Released images should be public
	fmt.Printf("Setting image to have public access: %v\n", name)
	err = api.SetImagePublic(name)
	if err != nil {
		plog.Fatalf("Marking GCE image with public ACLs failed: %v", err)
	}

	if spec.GCE.Publish != "" {
		obj := gs.Object{
			Name:        src.Prefix() + spec.GCE.Publish,
			ContentType: "text/plain",
		}
		media := strings.NewReader(
			fmt.Sprintf("projects/%s/global/images/%s\n",
				spec.GCE.Project, name))
		if err := src.Upload(ctx, &obj, media); err != nil {
			plog.Fatal(err)
		}
	} else {
		plog.Notice("GCE image name publishing disabled.")
	}

	var pendings []*gcloud.Pending
	for _, old := range oldImages {
		if old.Deprecated != nil && old.Deprecated.State != "" {
			continue
		}
		plog.Noticef("Deprecating old image %s", old.Name)
		pending, err := api.DeprecateImage(old.Name, gcloud.DeprecationStateDeprecated, imageLink)
		if err != nil {
			plog.Fatal(err)
		}
		pending.Interval = 1 * time.Second
		pending.Timeout = 0
		pendings = append(pendings, pending)
	}

	if spec.GCE.Limit > 0 && len(oldImages) > spec.GCE.Limit {
		plog.Noticef("Pruning %d GCE images.", len(oldImages)-spec.GCE.Limit)
		for _, old := range oldImages[spec.GCE.Limit:] {
			plog.Noticef("Deleting old image %s", old.Name)
			pending, err := api.DeleteImage(old.Name)
			if err != nil {
				plog.Fatal(err)
			}
			pending.Interval = 1 * time.Second
			pending.Timeout = 0
			pendings = append(pendings, pending)
		}
	}

	plog.Infof("Waiting on %d operations.", len(pendings))
	for _, pending := range pendings {
		err := pending.Wait()
		if err != nil {
			plog.Fatal(err)
		}
	}
}

func doAzure(ctx context.Context, client *http.Client, src *storage.Bucket, spec *channelSpec) {
	if spec.Azure.StorageAccount == "" {
		plog.Notice("Azure image creation disabled, skipping.")
		return
	}

	if azureProfile == "" {
		plog.Notice("No Azure profile defined, skipping.")
		return
	}

	blobName := AzureBlobName()

	for _, environment := range spec.Azure.Environments {
		api, err := azure.New(&azure.Options{
			AzureProfile:      azureProfile,
			AzureAuthLocation: azureAuth,
			AzureSubscription: environment.SubscriptionName,
			UseIdentity:       azureUseIdentity,
		})
		if err != nil {
			plog.Fatalf("failed to create Azure API: %v", err)
		}
		if err := api.SetupClients(); err != nil {
			plog.Fatalf("setting up clients: %v", err)
		}

		plog.Printf("Fetching Azure storage credentials for %q in %q", spec.Azure.StorageAccount, spec.Azure.ResourceGroup)

		storageKey, err := api.GetStorageServiceKeysARM(spec.Azure.StorageAccount, spec.Azure.ResourceGroup)
		if err != nil {
			plog.Fatalf("fetching storage key: %v", err)
		}
		if storageKey.Keys == nil {
			plog.Fatalf("No storage service keys found")
		}

		container := spec.Azure.Container
		if azureTestContainer != "" {
			container = azureTestContainer
		}

		plog.Printf("Signing %q in %q on %v...", blobName, container, environment.SubscriptionName)

		var url string
		for _, key := range *storageKey.Keys {
			blobExists, err := api.BlobExists(spec.Azure.StorageAccount, *key.Value, container, blobName)
			if err != nil {
				continue
			}
			if !blobExists {
				plog.Notice("Blob does not exist, skipping.")
				return
			}
			url, err = api.SignBlob(spec.Azure.StorageAccount, *key.Value, container, blobName)
			if err == nil {
				break
			}
		}
		if err != nil {
			plog.Fatalf("signing failed: %v", err)
		}
		plog.Noticef("Generated SAS: %q for %q", url, specChannel)
		plog.Noticef("Please update the SKU manually (or try to automate this step)!")
	}
}

func doAWS(ctx context.Context, client *http.Client, src *storage.Bucket, spec *channelSpec) {
	if spec.AWS.Image == "" || awsCredentialsFile == "" {
		plog.Notice("AWS image creation disabled.")
		return
	}

	awsImageMetadata, err := getSpecAWSImageMetadata(spec)
	if err != nil {
		return
	}

	imageName := awsImageMetadata["imageName"]

	for _, part := range spec.AWS.Partitions {
		for _, region := range part.Regions {
			if releaseDryRun {
				plog.Printf("Checking for images in %v %v...", part.Name, region)
			} else {
				plog.Printf("Publishing images in %v %v...", part.Name, region)
			}

			api, err := aws.New(&aws.Options{
				CredentialsFile: awsCredentialsFile,
				Profile:         part.Profile,
				Region:          region,
			})
			if err != nil {
				plog.Fatalf("creating client for %v %v: %v", part.Name, region, err)
			}

			publish := func(imageName string) error {
				imageID, err := api.FindImage(imageName)
				if err != nil {
					return fmt.Errorf("couldn't find image %q in %v %v: %v", imageName, part.Name, region, err)
				}

				if !releaseDryRun {
					err := api.PublishImage(imageID)
					if err != nil {
						return fmt.Errorf("couldn't publish image in %v %v: %v", part.Name, region, err)
					}
				}

				// Publish on AWS Marketplace AMIs in us-east-1.
				if publishMarketplace && region == "us-east-1" {
					// Create a new API client to consume the AWS Marketplace credentials.
					marketplace, err := aws.New(&aws.Options{
						CredentialsFile: awsMarketplaceCredentialsFile,
						Profile:         "default",
						Region:          "us-east-1",
					})
					if err != nil {
						return fmt.Errorf("creating API Marketplace client: %w", err)
					}

					// Define the launch instance type based on the arch.
					instanceType := "t3.medium"
					if specBoard == "arm64-usr" {
						instanceType = "m6g.medium"
					}

					for _, pid := range productIDs {
						if err := marketplace.UpdateProduct(imageID, accessRoleARN, username, specVersion, pid, instanceType, releaseDryRun); err != nil {
							return fmt.Errorf("updating product with ID %s: %w", pid, err)
						}
					}

				}

				return nil
			}

			if err := publish(imageName + "-hvm"); err != nil {
				plog.Fatalf("publishing AWS release: %v", err)
			}
		}
	}
}
