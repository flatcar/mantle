// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"compress/bzip2"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Bunzip2 does bunzip2 decompression from src to dst.
//
// It matches the signature of io.Copy.
func Bunzip2(dst io.Writer, src io.Reader) (written int64, err error) {
	bzr := bzip2.NewReader(src)
	return io.Copy(dst, bzr)
}

func Bunzip2FileGo(dst, src string) error {
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

// Bunzip2File does bunzip2 decompression from src file into dst file.
func Bunzip2File(dst, src string) error {
	lbunzip2, err := exec.LookPath("lbunzip2")
	if err != nil {
		return Bunzip2FileGo(dst, src)
	}

	cmd := exec.Command(lbunzip2, "--stdout", "--decompress", src)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("setup stdout pipe: %w", err)
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		os.Remove(dst)
		return fmt.Errorf("start lbunzip2: %w", err)
	}
	_, err = io.Copy(out, stdout)
	if err != nil {
		os.Remove(dst)
		return fmt.Errorf("copy: %w", err)
	}
	err = cmd.Wait()
	if err != nil {
		os.Remove(dst)
		return fmt.Errorf("lunbzip2 returned: %w", err)
	}
	return nil
}
