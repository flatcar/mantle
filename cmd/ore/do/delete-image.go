// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package do

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdDeleteImage = &cobra.Command{
		Use:   "delete-image [options]",
		Short: "Delete image",
		Long:  `Delete an image.`,
		RunE:  runDeleteImage,
	}
)

func init() {
	DO.AddCommand(cmdDeleteImage)
	cmdDeleteImage.Flags().StringVarP(&imageName, "name", "n", "", "image name")
}

func runDeleteImage(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "Unrecognized args in do delete-image cmd: %v\n", args)
		os.Exit(2)
	}

	if err := deleteImage(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	return nil
}

func deleteImage() error {
	if imageName == "" {
		return fmt.Errorf("Image name must be specified")
	}

	ctx := context.Background()

	image, err := API.GetUserImage(ctx, imageName, false)
	if err != nil {
		return err
	}

	if err := API.DeleteImage(ctx, image.ID); err != nil {
		return err
	}

	return nil
}
