// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/sdk"
	"github.com/flatcar-linux/mantle/sdk/omaha"
)

var (
	buildCmd = &cobra.Command{
		Use:   "build [object]",
		Short: "Build something",
	}
	buildUpdateCmd = &cobra.Command{
		Use:   "update",
		Short: "Build an image update payload",
		Run:   runBuildUpdate,
	}
)

func init() {
	buildCmd.AddCommand(buildUpdateCmd)
	root.AddCommand(buildCmd)
}

func runBuildUpdate(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		plog.Fatalf("Unrecognized arguments: %v", args)
	}

	err := omaha.GenerateFullUpdate(sdk.BuildImageDir("", ""))
	if err != nil {
		plog.Fatalf("Building full update failed: %v", err)
	}
}
