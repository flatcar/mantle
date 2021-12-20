// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package reader

import (
	"io"
)

// AtReader converts an io.ReaderAt into an io.Reader
func AtReader(ra io.ReaderAt) io.Reader {
	if rd, ok := ra.(io.Reader); ok {
		return rd
	}
	return &atReader{ReaderAt: ra}
}

type atReader struct {
	io.ReaderAt
	off int64
}

func (r *atReader) Read(p []byte) (n int, err error) {
	n, err = r.ReadAt(p, r.off)
	r.off += int64(n)
	return
}
