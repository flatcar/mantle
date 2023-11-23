// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package brightbox

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cmdRemoveCloudIPs = &cobra.Command{
		Use:   "remove-ips",
		Short: "Remove any remaining cloud IPs",
		Long:  `Remove left overs IP from previous garbage collection`,
		RunE:  removeCloudIPs,
	}
)

func init() {
	Brightbox.AddCommand(cmdRemoveCloudIPs)
}

func removeCloudIPs(cmd *cobra.Command, args []string) error {
	if err := API.RemoveCloudIPs(context.Background()); err != nil {
		return fmt.Errorf("removing cloud IPs: %w", err)
	}

	return nil
}
