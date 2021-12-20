// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package gcloud

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/platform/api/gcloud"
)

var (
	cmdDeleteImage = &cobra.Command{
		Use:   "delete-images <name>...",
		Short: "Delete GCE images",
		Run:   runDeleteImage,
	}
)

func init() {
	GCloud.AddCommand(cmdDeleteImage)
}

func runDeleteImage(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, "Specify image name(s).\n")
		os.Exit(2)
	}

	exit := 0
	pendings := map[string]*gcloud.Pending{}
	for _, name := range args {
		pending, err := api.DeleteImage(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			exit = 1
			continue
		}
		pendings[name] = pending
	}
	for name, pending := range pendings {
		if err := pending.Wait(); err != nil {
			fmt.Fprintf(os.Stderr, "Deleting %q failed: %v\n", name, err)
			exit = 1
		}
	}
	os.Exit(exit)
}
