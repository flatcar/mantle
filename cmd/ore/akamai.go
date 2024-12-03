// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/flatcar/mantle/cmd/ore/akamai"
)

func init() {
	root.AddCommand(akamai.Akamai)
}
