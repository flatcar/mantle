// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package packet

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	cmdGC = &cobra.Command{
		Use:   "gc",
		Short: "GC resources in Packet",
		Long:  `Delete devices created over the given duration ago.`,
		RunE:  runGC,
	}

	gcDuration time.Duration
)

func init() {
	Packet.AddCommand(cmdGC)
	cmdGC.Flags().DurationVar(&gcDuration, "duration", 5*time.Hour, "how old resources must be before they're considered garbage")
}

func runGC(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "Unrecognized args in packet gc cmd: %v\n", args)
		os.Exit(2)
	}

	if err := API.GC(gcDuration); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	return nil
}
