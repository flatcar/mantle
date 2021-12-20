// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cmdUnreplicateImage = &cobra.Command{
		Use:   "unreplicate-image image",
		Short: "Unreplicate an OS image in Azure",
		RunE:  runUnreplicateImage,
	}
)

func init() {
	Azure.AddCommand(cmdUnreplicateImage)
}

func runUnreplicateImage(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting 1 argument")
	}

	return api.UnreplicateImage(args[0])
}
