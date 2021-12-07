// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate protoc --go_out=import_path=$GOPACKAGE:. update_metadata.proto

package metadata

// Magic is the first four bytes of any update payload.
const Magic = "CrAU"

// Major version of the payload format.
const Version = 1

// DeltaArchiveHeader begins the payload file.
type DeltaArchiveHeader struct {
	Magic        [4]byte // "CrAU"
	Version      uint64  // 1
	ManifestSize uint64
}
