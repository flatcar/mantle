// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/coreos/pkg/capnslog"
	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/cli"
)

var plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "cork")
var root = &cobra.Command{
	Use:   "cork [command]",
	Short: "The CoreOS SDK Manager",
}

func main() {
	cli.Execute(root)
}
