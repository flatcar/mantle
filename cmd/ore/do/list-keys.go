// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package do

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdListKeys = &cobra.Command{
		Use:   "list-keys",
		Short: "List DigitalOcean SSH keys",
		RunE:  runListKeys,
	}
)

func init() {
	DO.AddCommand(cmdListKeys)
}

func runListKeys(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "Unrecognized args in do list-keys cmd: %v\n", args)
		os.Exit(2)
	}

	ctx := context.Background()

	keys, err := API.ListKeys(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't list keys: %v\n", err)
		os.Exit(1)
	}

	for _, key := range keys {
		fmt.Println(key.Name)
	}
	return nil
}
