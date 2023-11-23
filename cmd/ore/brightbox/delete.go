// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package brightbox

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cmdDelete = &cobra.Command{
		Use:   "delete-image",
		Short: "Delete image on Brightbox",
		Long:  `Delete an image from Brightbox.`,
		RunE:  runDelete,
	}

	id string
)

func init() {
	Brightbox.AddCommand(cmdDelete)
	cmdDelete.Flags().StringVar(&id, "id", "", "image ID")
}

func runDelete(cmd *cobra.Command, args []string) error {
	if err := API.DeleteImage(context.Background(), id); err != nil {
		return fmt.Errorf("deleting image: %w", err)
	}

	return nil
}
