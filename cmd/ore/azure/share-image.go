// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cmdShareImage = &cobra.Command{
		Use:   "share-image image-name",
		Short: "Set permissions on an azure OS image",
		RunE:  runShareImage,
	}

	sharePermission string
)

func init() {
	sv := cmdShareImage.Flags().StringVar

	sv(&sharePermission, "permission", "public", "Image permission (one of: public, msdn, private)")

	Azure.AddCommand(cmdShareImage)
}

func runShareImage(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting 1 argument, got %d", len(args))
	}

	if sharePermission == "" {
		return fmt.Errorf("permission is required")
	}

	return api.ShareImage(args[0], sharePermission)
}
