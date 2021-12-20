// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

// The package github.com/Azure/azure-sdk-for-go needs go 1.7 for TLS
// renegotiation, so only link in the ore subcommands if we build with go 1.7.

package main

import (
	"github.com/flatcar-linux/mantle/cmd/ore/azure"
)

func init() {
	root.AddCommand(azure.Azure)
}
