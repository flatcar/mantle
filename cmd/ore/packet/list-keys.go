// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package packet

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdListKeys = &cobra.Command{
		Use:   "list-keys",
		Short: "List Packet SSH keys",
		RunE:  runListKeys,
	}
)

func init() {
	Packet.AddCommand(cmdListKeys)
}

func runListKeys(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "Unrecognized args in packet list-keys cmd: %v\n", args)
		os.Exit(2)
	}

	keys, err := API.ListKeys()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't list keys: %v\n", err)
		os.Exit(1)
	}

	for _, key := range keys {
		fmt.Println(key.Label)
	}
	return nil
}
