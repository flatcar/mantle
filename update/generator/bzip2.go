// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"bytes"
	"io"
	"os"
	"os/exec"
)

type bzip2Writer struct {
	cmd *exec.Cmd
	in  io.WriteCloser
}

// NewBzip2Writer wraps a writer, compressing all data written to it.
func NewBzip2Writer(w io.Writer) (io.WriteCloser, error) {
	zipper, err := exec.LookPath("lbzip2")
	if err != nil {
		zipper = "bzip2"
	}

	cmd := exec.Command(zipper, "-c")
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	return &bzip2Writer{cmd, in}, cmd.Start()
}

func (bz *bzip2Writer) Write(p []byte) (n int, err error) {
	return bz.in.Write(p)
}

// Close stops the compressor, flushing out any remaining data.
// The underlying writer is not closed.
func (bz *bzip2Writer) Close() error {
	if err := bz.in.Close(); err != nil {
		return err
	}
	return bz.cmd.Wait()
}

// Bzip2 simplifies using a Bzip2Writer when working with in-memory buffers.
func Bzip2(data []byte) ([]byte, error) {
	buf := bytes.Buffer{}
	zip, err := NewBzip2Writer(&buf)
	if err != nil {
		return nil, err
	}

	if _, err := zip.Write(data); err != nil {
		return nil, err
	}

	if err := zip.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
