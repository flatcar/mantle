// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package brightbox

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cmdCreate = &cobra.Command{
		Use:   "create-image",
		Short: "Create image on Brightbox",
		Long: `Upload an image to Brigthbox.

After a successful run, the final line of output will be the ID of the image.
`,
		RunE: runCreate,
	}

	url, name string
)

func init() {
	Brightbox.AddCommand(cmdCreate)
	cmdCreate.Flags().StringVar(&url, "url",
		"https://stable.release.flatcar-linux.net/amd64-usr/current/flatcar_production_openstack_image.img",
		"Flatcar image URL")
	cmdCreate.Flags().StringVar(&name, "name", "", "image name")
}

func runCreate(cmd *cobra.Command, args []string) error {
	id, err := API.UploadImage(context.Background(), name, url)
	if err != nil {
		return fmt.Errorf("creating an image: %w", err)
	}

	fmt.Println(id)
	return nil
}
