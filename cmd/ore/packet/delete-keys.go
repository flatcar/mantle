// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package packet

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdDeleteKeys = &cobra.Command{
		Use:   "delete-keys <key>...",
		Short: "Delete Packet SSH keys",
		RunE:  runDeleteKeys,
	}
)

func init() {
	Packet.AddCommand(cmdDeleteKeys)
}

func runDeleteKeys(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Specify at least one key.\n")
		os.Exit(2)
	}

	labels := map[string]bool{}
	for _, arg := range args {
		labels[arg] = true
	}

	keys, err := API.ListKeys()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't list keys: %v\n", err)
		os.Exit(1)
	}

	exit := 0
	for _, key := range keys {
		if labels[key.Label] {
			if err := API.DeleteKey(key.ID); err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't delete key: %v\n", key.Label)
				exit = 1
			}
			delete(labels, key.Label)
		}
	}

	for label := range labels {
		fmt.Fprintf(os.Stderr, "No such key: %v\n", label)
		exit = 1
	}

	os.Exit(exit)
	return nil
}
