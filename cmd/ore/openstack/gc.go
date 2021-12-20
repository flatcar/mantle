// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	cmdGC = &cobra.Command{
		Use:   "gc",
		Short: "GC resources in OpenStack",
		Long:  `Delete instances created over the given duration ago`,
		RunE:  runGC,
	}

	gcDuration time.Duration
)

func init() {
	OpenStack.AddCommand(cmdGC)
	cmdGC.Flags().DurationVar(&gcDuration, "duration", 5*time.Hour, "how old resources must be before they're considered garbage")
}

func runGC(cmd *cobra.Command, args []string) error {
	err := API.GC(gcDuration)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't gc: %v\n", err)
		os.Exit(1)
	}
	return nil
}
