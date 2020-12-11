// Copyright 2020 Kinvolk GmbH
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

package aws

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cmdDelete = &cobra.Command{
		Use:   "delete",
		Short: "Delete AWS images",
		Long: `Delete a Flatcar image by deleting any file from S3, the EC2 snapshot, and the AMI.

An error code is returned when something was found but couldn't be removed. No error code is returned when nothing was found.
`,
		Example: `  ore aws delete --region=us-east-1 \
	  --ami-name="myimage-1.2.3" --name="myimage-1.2.3" --file=flatcar_production_ami_vmdk_image.vmdk --bucket s3://flatcar-dev/myimages/`,
		RunE: runDelete,
	}

	deleteBucket    string
	deleteImageName string
	deleteBoard     string
	deleteFile      string
	deleteAMIName   string
)

func init() {
	AWS.AddCommand(cmdDelete)
	cmdDelete.Flags().StringVar(&deleteBucket, "bucket", "", "s3://bucket/prefix/ (defaults to a regional bucket and prefix defaults to $USER/board/name)")
	cmdDelete.Flags().StringVar(&deleteImageName, "name", "", "name of image EC2 snapshot to delete")
	cmdDelete.Flags().StringVar(&deleteBoard, "board", "amd64-usr", "board used for naming with default prefix and AMI architecture")
	cmdDelete.Flags().StringVar(&deleteFile, "file", defaultUploadFile(), "name of image file object in the bucket")
	cmdDelete.Flags().StringVar(&deleteAMIName, "ami-name", "", "name of the AMI to create (default: Container-Linux-$USER-$VERSION)")
}

func runDelete(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "Unrecognized args in aws delete cmd: %v\n", args)
		os.Exit(2)
	}

	amiName := deleteAMIName

	switch deleteBoard {
	case "amd64-usr":
	case "arm64-usr":
		if !strings.HasSuffix(amiName, "-arm64") {
			amiName = amiName + "-arm64"
		}
	default:
		fmt.Fprintf(os.Stderr, "No AMI name suffix known for board %q\n", deleteBoard)
		os.Exit(1)
	}

	s3URL, err := defaultBucketURL(deleteBucket, deleteImageName, deleteBoard, deleteFile, region)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	plog.Debugf("S3 object: %v\n", s3URL)
	s3BucketName := s3URL.Host
	s3ObjectPath := strings.TrimPrefix(s3URL.Path, "/")

	err = API.RemoveImage(amiName, deleteImageName, s3BucketName, s3ObjectPath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to delete: %v\n", err)
		os.Exit(1)
	}

	return nil
}
