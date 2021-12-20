// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdCreateImageARM = &cobra.Command{
		Use:   "create-image-arm",
		Short: "Create Azure image",
		Long:  "Create Azure image from a blob url",
		RunE:  runCreateImageARM,
	}

	imageName     string
	blobUrl       string
	resourceGroup string
)

func init() {
	sv := cmdCreateImageARM.Flags().StringVar

	sv(&imageName, "image-name", "", "image name")
	sv(&blobUrl, "image-blob", "", "source blob url")
	sv(&resourceGroup, "resource-group", "kola", "resource group name")

	Azure.AddCommand(cmdCreateImageARM)
}

func runCreateImageARM(cmd *cobra.Command, args []string) error {
	if err := api.SetupClients(); err != nil {
		fmt.Fprintf(os.Stderr, "setting up clients: %v\n", err)
		os.Exit(1)
	}
	img, err := api.CreateImage(imageName, resourceGroup, blobUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't create image: %v\n", err)
		os.Exit(1)
	}
	if img.ID == nil {
		fmt.Fprintf(os.Stderr, "received nil image\n")
		os.Exit(1)
	}
	err = json.NewEncoder(os.Stdout).Encode(&struct {
		ID       *string
		Location *string
	}{
		ID:       img.ID,
		Location: img.Location,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't encode result: %v\n", err)
		os.Exit(1)
	}
	return nil
}
