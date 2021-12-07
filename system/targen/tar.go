// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

// exec is extension of the standard os.exec package.
// Adds a handy dandy interface and assorted other features.

package targen

import (
	"archive/tar"
	"io"
	"os"

	"github.com/coreos/pkg/capnslog"
)

var plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "targen")

type TarGen struct {
	files    []string
	binaries []string
}

func New() *TarGen {
	return &TarGen{}
}

func (t *TarGen) AddFile(path string) *TarGen {
	plog.Tracef("adding file %q", path)
	t.files = append(t.files, path)
	return t
}

func (t *TarGen) AddBinary(path string) *TarGen {
	plog.Tracef("adding binary %q", path)
	t.binaries = append(t.binaries, path)
	return t
}

func tarWriteFile(tw *tar.Writer, file string) error {
	plog.Tracef("writing file %q", file)

	st, err := os.Stat(file)
	if err != nil {
		return err
	}
	hdr := &tar.Header{
		Name:    file,
		Size:    st.Size(),
		Mode:    int64(st.Mode()),
		ModTime: st.ModTime(),
	}

	if err = tw.WriteHeader(hdr); err != nil {
		return err
	}

	f, err := os.Open(file)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err := io.Copy(tw, f); err != nil {
		return err
	}

	return nil
}

func (t *TarGen) Generate(w io.Writer) error {
	tw := tar.NewWriter(w)

	// store processed files here so we skip duplicates.
	copied := make(map[string]struct{})

	for _, file := range t.files {
		if _, ok := copied[file]; ok {
			plog.Tracef("skipping duplicate file %q", file)
			continue
		}

		plog.Tracef("copying file %q", file)

		if err := tarWriteFile(tw, file); err != nil {
			return err
		}

		copied[file] = struct{}{}
	}

	for _, binary := range t.binaries {
		libs, err := ldd(binary)
		if err != nil {
			return err
		}

		for _, lib := range libs {
			if _, ok := copied[lib]; ok {
				plog.Tracef("skipping duplicate library %q", lib)

				continue
			}

			plog.Tracef("copying library %q", lib)

			if err := tarWriteFile(tw, lib); err != nil {
				return err
			}

			copied[lib] = struct{}{}
		}

		if _, ok := copied[binary]; ok {
			plog.Tracef("skipping duplicate binary %q", binary)
			continue
		}

		plog.Tracef("copying binary %q", binary)

		if err := tarWriteFile(tw, binary); err != nil {
			return err
		}

		copied[binary] = struct{}{}
	}

	if err := tw.Close(); err != nil {
		return err
	}

	return nil
}
