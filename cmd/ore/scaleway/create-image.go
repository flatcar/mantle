// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package scaleway

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const bucket = "flatcar-testing"

var (
	cmdCreate = &cobra.Command{
		Use:   "create-image",
		Short: "Create Scaleway image",
		RunE:  runCreate,
		Example: `IMAGE_ID=$(ore scaleway \
  --scaleway-access-key "${SCALEWAY_ACCESS_KEY}" \
  --scaleway-secret-key "${SCALEWAY_SECRET_KEY}" \
  --scaleway-organization-id "${SCALEWAY_ORGANIZATION_ID}" \
  create-image --channel beta)`,
	}
	channel string
	version string
	board   string
	file    string
)

func init() {
	Scaleway.AddCommand(cmdCreate)

	cmdCreate.Flags().StringVar(&channel, "channel", "stable", "Flatcar channel")
	cmdCreate.Flags().StringVar(&version, "version", "current", "Flatcar version")
	cmdCreate.Flags().StringVar(&board, "board", "amd64-usr", "board used for naming with default prefix and AMI architecture")
	cmdCreate.Flags().StringVar(&file, "file", "flatcar_production_scaleway_image.qcow2", "path to local Flatcar image (.qcow2)")
}

func runCreate(cmd *cobra.Command, args []string) error {
	if err := API.InitializeBucket(bucket); err != nil {
		return fmt.Errorf("creating bucket %s: %v", bucket, err)
	}

	f, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("opening Flatcar image file %s: %v", file, err)
	}

	defer f.Close()

	key := fmt.Sprintf("%s/%s/%s/%s", channel, version, board, filepath.Base(file))
	if err := API.UploadObject(f, bucket, key, true); err != nil {
		return fmt.Errorf("uploading Flatcar image file %s: %v", file, err)
	}

	ID, err := API.CreateSnapshot(context.Background(), bucket, key)
	if err != nil {
		return fmt.Errorf("creating Flatcar image: %v", err)
	}

	if err := API.DeleteObject(bucket, key); err != nil {
		return fmt.Errorf("deleting Flatcar image from s3 bucket: %s", fmt.Sprintf("s3://%s/%s", bucket, key))
	}

	fmt.Println(ID)

	return nil
}
