// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdCopyImage = &cobra.Command{
		Use:   "copy-image <dest-region...>",
		Short: "Copy AWS image between regions",
		Long: `Copy an AWS image to one or more regions.

After a successful run, the final line of output will be a line of JSON describing the resources created.
`,
		RunE: runCopyImage,
	}

	sourceImageID string
)

func init() {
	AWS.AddCommand(cmdCopyImage)
	cmdCopyImage.Flags().StringVar(&sourceImageID, "image", "", "source AMI")
}

func runCopyImage(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Specify one or more regions.\n")
		os.Exit(2)
	}

	amis, err := API.CopyImage(sourceImageID, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't copy images: %v\n", err)
		os.Exit(1)
	}

	err = json.NewEncoder(os.Stdout).Encode(amis)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't encode result: %v\n", err)
		os.Exit(1)
	}
	return nil
}
