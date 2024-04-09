// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package scaleway

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	cmdGC = &cobra.Command{
		Use:   "gc",
		Short: "GC resources in Scaleway",
		Long:  `Delete instances and images created over the given duration ago`,
		RunE:  runGC,
	}

	gcDuration time.Duration
)

func init() {
	Scaleway.AddCommand(cmdGC)
	cmdGC.Flags().DurationVar(&gcDuration, "duration", 5*time.Hour, "how old resources must be before they're considered garbage")
}

func runGC(cmd *cobra.Command, args []string) error {
	if err := API.GC(context.Background(), gcDuration); err != nil {
		return fmt.Errorf("running garbage collection: %w", err)
	}

	return nil
}
