// Copyright 2017 CoreOS, Inc.
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
