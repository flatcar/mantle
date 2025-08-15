package stackit

import (
	"fmt"
	"github.com/spf13/cobra"
	"time"
)

var (
	cmdGC = &cobra.Command{
		Use:   "gc",
		Short: "GC resources in STACKIT",
		Long:  "Delete instances and images created over the given duration ago",
		RunE:  runGC,
	}

	gcDuration time.Duration
)

func init() {
	STACKIT.AddCommand(cmdGC)
	cmdGC.Flags().DurationVar(&gcDuration, "duration", 5*time.Hour, "how old resources must be before they're considered garbage")
}

func runGC(cmd *cobra.Command, args []string) error {
	if err := API.GC(cmd.Context(), gcDuration); err != nil {
		return fmt.Errorf("running garbage collection: %w", err)
	}

	return nil
}
