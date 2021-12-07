// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"runtime"
)

func PortageArch() string {
	arch := runtime.GOARCH
	switch arch {
	case "386":
		arch = "x86"

	// Go and Portage agree for these.
	case "amd64":
	case "arm":
	case "arm64":
	case "ppc64":

	// Gentoo doesn't have a little-endian PPC port.
	case "ppc64le":
		fallthrough
	default:
		panic("No portage arch defined for " + arch)
	}
	return arch
}
