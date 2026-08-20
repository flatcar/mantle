// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	cmdGC = &cobra.Command{
		Use:   "gc",
		Short: "GC resources in Oracle Cloud Infrastructure",
		Long:  "Delete mantle-managed instances created over the given duration ago",
		RunE:  runGC,
	}

	gcDuration time.Duration
)

func init() {
	OracleCloud.AddCommand(cmdGC)
	cmdGC.Flags().DurationVar(&gcDuration, "duration", 5*time.Hour, "how old resources must be before they're considered garbage")
}

func runGC(cmd *cobra.Command, args []string) error {
	if options.CompartmentID == "" {
		return fmt.Errorf("--oraclecloud-compartment-id is required")
	}

	if err := api.GC(cmd.Context(), gcDuration); err != nil {
		return fmt.Errorf("running garbage collection: %w", err)
	}

	return nil
}
