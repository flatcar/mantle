// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/flatcar-linux/mantle/cmd/ore/packet"
)

func init() {
	root.AddCommand(packet.Packet)
}
