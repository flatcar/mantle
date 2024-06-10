// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package hetzner

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cmdCreate = &cobra.Command{
		Use:   "create-image",
		Short: "Create image on Hetzner",
		Long: `Upload an image to Hetzner.

After a successful run, the final line of output will be the ID of the image.
`,
		RunE: runCreate,
	}

	path  string
	name  string
	board string
)

func init() {
	Hetzner.AddCommand(cmdCreate)
	cmdCreate.Flags().StringVar(&path, "file",
		"https://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_production_hetzner_image.bin.bz2",
		"Flatcar image (can be an absolute path or an URL)")
	cmdCreate.Flags().StringVar(&name, "name", "", "image name")
	cmdCreate.Flags().StringVar(&board, "board", "amd64-usr", "board of the image")
}

func runCreate(cmd *cobra.Command, args []string) error {
	id, err := API.UploadImage(cmd.Context(), name, path, board)
	if err != nil {
		return fmt.Errorf("creating an image: %w", err)
	}
	fmt.Println(id)
	return nil
}
