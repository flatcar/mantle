// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdDelete = &cobra.Command{
		Use:   "delete-image",
		Short: "Delete image on OpenStack",
		Long:  `Delete an image from OpenStack.`,
		RunE:  runDelete,
	}

	id string
)

func init() {
	OpenStack.AddCommand(cmdDelete)
	cmdDelete.Flags().StringVar(&id, "id", "", "image ID")
}

func runDelete(cmd *cobra.Command, args []string) error {
	img, err := API.ResolveImage(id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't find image: %v\n", err)
		os.Exit(1)
	}
	err = API.DeleteImage(img)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't delete image: %v\n", err)
		os.Exit(1)
	}
	return nil
}
