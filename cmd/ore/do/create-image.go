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
	cmdCreateImage = &cobra.Command{
		Use:   "create-image [options]",
		Short: "Create image",
		Long:  `Create an image.`,
		RunE:  runCreateImage,
	}
)

func init() {
	DO.AddCommand(cmdCreateImage)
	cmdCreateImage.Flags().StringVar(&options.Region, "region", "sfo2", "region slug")
	cmdCreateImage.Flags().StringVarP(&imageName, "name", "n", "", "image name")
	cmdCreateImage.Flags().StringVarP(&imageURL, "url", "u", "", "image source URL (e.g. \"https://stable.release.flatcar-linux.net/amd64-usr/current/flatcar_production_digitalocean_image.bin.bz2\"")
}

func runCreateImage(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "Unrecognized args in do create-image cmd: %v\n", args)
		os.Exit(2)
	}

	if err := createImage(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	return nil
}

func createImage() error {
	if imageName == "" {
		return fmt.Errorf("Image name must be specified")
	}
	if imageURL == "" {
		return fmt.Errorf("Image URL must be specified")
	}
	ctx := context.Background()

	_, err := API.CreateImage(ctx, imageName, imageURL)

	return err
}
