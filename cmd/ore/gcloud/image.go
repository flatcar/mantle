// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package gcloud

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

var (
	cmdImage = &cobra.Command{
		Use:   "list-images --prefix=<prefix>",
		Short: "List images in GCE",
		Run:   runImage,
	}

	imagePrefix string
)

func init() {
	cmdImage.Flags().StringVar(&imagePrefix, "prefix", "", "prefix to filter list by")
	GCloud.AddCommand(cmdImage)
}

func runImage(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "Unrecognized args in plume list cmd: %v\n", args)
		os.Exit(2)
	}

	images, err := api.ListImages(context.Background(), imagePrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	for _, image := range images {
		fmt.Printf("%v\n", image.Name)
	}
}
