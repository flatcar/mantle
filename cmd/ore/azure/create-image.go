// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/platform/api/azure"
)

var (
	cmdCreateImage = &cobra.Command{
		Use:   "create-image",
		Short: "Create Azure image",
		Long:  "Create Azure image from a local VHD file",
		RunE:  runCreateImage,
	}

	// create image options
	md azure.OSImage
)

func today() string {
	return time.Now().Format("2006-01-02")
}

func init() {
	sv := cmdCreateImage.Flags().StringVar

	sv(&md.Name, "name", "", "image name")
	sv(&md.Label, "label", "", "image label")
	sv(&md.Description, "description", "", "image description")
	sv(&md.MediaLink, "blob", "", "source blob url")
	sv(&md.ImageFamily, "family", "", "image family")
	sv(&md.PublishedDate, "published-date", today(), "image published date, parsed as RFC3339")
	sv(&md.RecommendedVMSize, "recommended-vm-size", "Medium", "recommended VM size")
	sv(&md.IconURI, "icon-uri", "coreos-globe-color-lg-100px.png", "icon URI")
	sv(&md.SmallIconURI, "small-icon-uri", "coreos-globe-color-lg-45px.png", "small icon URI")

	Azure.AddCommand(cmdCreateImage)
}

func runCreateImage(cmd *cobra.Command, args []string) error {
	md.Category = "Public"
	md.OS = "Linux"
	return api.AddOSImage(&md)
}
