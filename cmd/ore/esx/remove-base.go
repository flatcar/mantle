// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package esx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdDeleteBase = &cobra.Command{
		Use:   "remove-base",
		Short: "Remove base vm on ESX",
		Long:  `Remove base vm on ESX server.`,
		RunE:  runBaseDelete,
	}

	vmName string
)

func init() {
	ESX.AddCommand(cmdDeleteBase)
	cmdDeleteBase.Flags().StringVar(&vmName, "name", "", "name of base VM")
}

func runBaseDelete(cmd *cobra.Command, args []string) error {
	err := API.TerminateDevice(vmName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't delete base VM: %v\n", err)
		os.Exit(1)
	}
	return nil
}
