// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package brightbox

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cmdRemoveServers = &cobra.Command{
		Use:   "remove-servers",
		Short: "Remove any remaining servers",
		Long:  `Remove left overs server from previous garbage collection`,
		RunE:  removeServers,
	}
)

func init() {
	Brightbox.AddCommand(cmdRemoveServers)
}

func removeServers(cmd *cobra.Command, args []string) error {
	if err := API.RemoveServers(context.Background()); err != nil {
		return fmt.Errorf("removing servers: %w", err)
	}

	return nil
}
