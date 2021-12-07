// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyRegularFile copies a file in place, updates are not atomic. If
// the destination doesn't exist it will be created with the same
// permissions as the original but umask is respected. If the
// destination already exists the permissions will remain as-is.
func CopyRegularFile(src, dest string) (err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}
	mode := srcInfo.Mode()
	if !mode.IsRegular() {
		return fmt.Errorf("Not a regular file: %s", src)
	}

	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() {
		e := destFile.Close()
		if err == nil {
			err = e
		}
	}()

	_, err = io.Copy(destFile, srcFile)
	return err
}

// InstallRegularFile copies a file, creating any parent directories.
func InstallRegularFile(src, dest string) error {
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	return CopyRegularFile(src, dest)
}
