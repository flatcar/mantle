// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"os"
)

// IsSymlink checks if a path is a symbolic link.
func IsSymlink(path string) bool {
	st, err := os.Lstat(path)
	return err == nil && st.Mode()&os.ModeSymlink == os.ModeSymlink
}
