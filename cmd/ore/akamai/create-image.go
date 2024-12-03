// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package akamai

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cmdCreate = &cobra.Command{
		Use:   "create-image",
		Short: "Create Akamai image",
		RunE:  runCreate,
		Example: `IMAGE_ID=$(ore akamai \
  --akamai-token "${AKAMAI_TOKEN}" \
  --akamai-region "${AKAMAI_REGION}" \
  create-image --name my-image --file /path/to/flatcar_production_akamai_image.bin.gz)`,
	}
	file string
	name string
)

func init() {
	Akamai.AddCommand(cmdCreate)

	cmdCreate.Flags().StringVar(&file, "file", "flatcar_production_akamai_image.bin.gz", "path to local Flatcar image (.bin.gz)")
	cmdCreate.Flags().StringVar(&name, "name", "flatcar-kola-test", "name of the image")
}

func runCreate(cmd *cobra.Command, args []string) error {
	ID, err := api.UploadImage(context.Background(), name, file)
	if err != nil {
		return fmt.Errorf("creating Flatcar image: %v", err)
	}

	fmt.Println(ID)

	return nil
}
