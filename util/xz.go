// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"io"
	"os"

	"github.com/ulikunitz/xz"
)

// XZ2File does xz decompression from src file into dst file
func XZ2File(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	reader, err := xz.NewReader(in)
	if err != nil {
		os.Remove(dst)
		return err
	}

	_, err = io.Copy(out, reader)
	if err != nil {
		os.Remove(dst)
		return err
	}
	return nil
}
