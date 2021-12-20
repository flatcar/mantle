// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"fmt"
	"os"

	"github.com/flatcar-linux/mantle/sdk"
	"github.com/spf13/cobra"
)

var (
	cmdCreate = &cobra.Command{
		Use:   "create-image",
		Short: "Create image on OpenStack",
		Long: `Upload an image to OpenStack.

After a successful run, the final line of output will be the ID of the image.
`,
		RunE: runCreate,
	}

	path string
	name string
)

func init() {
	OpenStack.AddCommand(cmdCreate)
	cmdCreate.Flags().StringVar(&path, "file",
		sdk.BuildRoot()+"/images/amd64-usr/latest/coreos_production_openstack_image.img",
		"path to CoreOS image (build with: ./image_to_vm.sh --format=openstack ...)")
	cmdCreate.Flags().StringVar(&name, "name", "", "image name")
}

func runCreate(cmd *cobra.Command, args []string) error {
	id, err := API.UploadImage(name, path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't create image: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(id)
	return nil
}
