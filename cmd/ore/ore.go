// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/cli"
)

var (
	root = &cobra.Command{
		Use:   "ore [command]",
		Short: "cloud image creation and upload tools",
	}
)

func main() {
	cli.Execute(root)
}
