// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	cmdCreateImage = &cobra.Command{
		Use:   "create-image",
		Short: "Create Oracle Cloud Infrastructure image",
		Long:  "Upload an image to Object Storage and import it as an OCI custom image",
		RunE:  runCreateImage,
		Example: `IMAGE_ID=$(ore oraclecloud \
  --oraclecloud-compartment-id "${ORACLE_COMPARTMENT_ID}" \
  create-image \
  --oraclecloud-bucket "${ORACLE_BUCKET}" \
  --board "${CIA_ARCH}-usr" \
  --name "${kola_test_basename}" \
  --file "${ORACLE_IMAGE_NAME}")`,
	}

	imageFile            string
	imageName            string
	imageBoard           string
	imageObjectName      string
	imageSourceImageType string
)

func init() {
	OracleCloud.AddCommand(cmdCreateImage)
	cmdCreateImage.Flags().StringVar(&options.Namespace, "oraclecloud-namespace", "", "Oracle Cloud Infrastructure Object Storage namespace (default: auto-detect)")
	cmdCreateImage.Flags().StringVar(&options.Bucket, "oraclecloud-bucket", "", "Oracle Cloud Infrastructure Object Storage bucket for image uploads")
	cmdCreateImage.Flags().StringVar(&imageFile, "file", "flatcar_production_oraclecloud_image.qcow2", "path to local Flatcar image (.qcow2 or .vmdk)")
	cmdCreateImage.Flags().StringVar(&imageName, "name", "flatcar-kola-test", "name of the image")
	cmdCreateImage.Flags().StringVar(&imageBoard, "board", "amd64-usr", "board of the image")
	cmdCreateImage.Flags().StringVar(&imageObjectName, "oraclecloud-object-name", "", "Object Storage object name to use for the upload (default: <board>/<basename of --file>)")
	cmdCreateImage.Flags().StringVar(&imageSourceImageType, "source-image-type", "QCOW2", "image import source type: QCOW2 or VMDK")
}

func runCreateImage(cmd *cobra.Command, args []string) error {
	if options.CompartmentID == "" {
		return fmt.Errorf("--oraclecloud-compartment-id is required")
	}
	if options.Bucket == "" {
		return fmt.Errorf("--oraclecloud-bucket is required")
	}

	objectName := imageObjectName
	if objectName == "" {
		objectName = fmt.Sprintf("%s/%s", imageBoard, filepath.Base(imageFile))
	}

	ID, err := api.UploadImage(cmd.Context(), imageName, imageFile, objectName, imageSourceImageType)
	if err != nil {
		return fmt.Errorf("creating Flatcar image: %w", err)
	}

	fmt.Println(ID)
	return nil
}
