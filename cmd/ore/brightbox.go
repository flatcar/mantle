// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/flatcar/mantle/cmd/ore/brightbox"
)

func init() {
	root.AddCommand(brightbox.Brightbox)
}
