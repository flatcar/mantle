// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/flatcar/mantle/cmd/ore/hetzner"
)

func init() {
	root.AddCommand(hetzner.Hetzner)
}
