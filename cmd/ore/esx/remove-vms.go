// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package esx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdRemoveVMs = &cobra.Command{
		Use:   "remove-vms",
		Short: "Remove VMs on ESX",
		Long: `Remove all VMs on ESX that match a pattern

After a successful run, all names of deleted VMs are written in one line each.
`,
		RunE: runRemoveVMs,
	}

	patternToRemove string
)

func init() {
	ESX.AddCommand(cmdRemoveVMs)
	cmdRemoveVMs.Flags().StringVar(&patternToRemove, "pattern", "*", "Pattern that VMs to be removed should match")
}

func runRemoveVMs(cmd *cobra.Command, args []string) error {
	names, err := API.GetDevices(patternToRemove)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't list VMs: %v\n", err)
		os.Exit(1)
	}

	var failed bool
	for _, name := range names {
		err := API.TerminateDevice(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't delete VM %q: %v\n", name, err)
			failed = true
		}
		fmt.Println(name)
	}

	if failed {
		os.Exit(1)
	}
	return nil
}
