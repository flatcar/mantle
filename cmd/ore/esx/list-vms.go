// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package esx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdListVMs = &cobra.Command{
		Use:   "list-vms",
		Short: "List VMs on ESX",
		Long: `List all names of VMs on ESX

After a successful run, all names are written in one line each.
`,
		RunE: runListVMs,
	}

	patternToList string
)

func init() {
	ESX.AddCommand(cmdListVMs)
	cmdListVMs.Flags().StringVar(&patternToList, "pattern", "*", "Pattern to match for")
}

func runListVMs(cmd *cobra.Command, args []string) error {
	names, err := API.GetDevices(patternToList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't list VMs: %v\n", err)
		os.Exit(1)
	}
	for _, name := range names {
		fmt.Println(name)
	}
	return nil
}
