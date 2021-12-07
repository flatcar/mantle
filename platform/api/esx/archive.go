// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package esx

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/vmware/govmomi/ovf"
)

type archive struct {
	path string
}

type archiveEntry struct {
	io.Reader
	f *os.File
}

func (t *archiveEntry) Close() error {
	return t.f.Close()
}

func (t *archive) readOvf(fpath string) ([]byte, error) {
	r, _, err := t.open(fpath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return ioutil.ReadAll(r)
}

func (t *archive) readEnvelope(fpath string) (*ovf.Envelope, error) {
	if fpath == "" {
		return nil, nil
	}

	r, _, err := t.open(fpath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	e, err := ovf.Unmarshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ovf: %s", err.Error())
	}

	return e, nil
}

func (t *archive) open(pattern string) (io.ReadCloser, int64, error) {
	f, err := os.Open(t.path)
	if err != nil {
		return nil, 0, err
	}

	r := tar.NewReader(f)

	for {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			f.Close()
			return nil, 0, err
		}

		matched, err := path.Match(pattern, path.Base(h.Name))
		if err != nil {
			f.Close()
			return nil, 0, err
		}

		if matched {
			return &archiveEntry{r, f}, h.Size, nil
		}
	}

	f.Close()
	return nil, 0, fmt.Errorf("couldn't find file in archive")
}
