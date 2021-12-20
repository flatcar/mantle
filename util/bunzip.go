// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"compress/bzip2"
	"io"
	"os"
)

// Bunzip2 does bunzip2 decompression from src to dst.
//
// It matches the signature of io.Copy.
func Bunzip2(dst io.Writer, src io.Reader) (written int64, err error) {
	bzr := bzip2.NewReader(src)
	return io.Copy(dst, bzr)
}

// Bunzip2File does bunzip2 decompression from src file into dst file.
func Bunzip2File(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}

	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	_, err = Bunzip2(out, in)
	if err != nil {
		os.Remove(dst)
	}
	return err
}
