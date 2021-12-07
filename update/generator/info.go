// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"crypto/sha256"
	"io"
	"os"

	"github.com/golang/protobuf/proto"

	"github.com/flatcar-linux/mantle/update/metadata"
)

func NewInstallInfo(r io.ReadSeeker) (*metadata.InstallInfo, error) {
	sha := sha256.New()
	size, err := io.Copy(sha, r)
	if err != nil {
		return nil, err
	}

	if _, err := r.Seek(0, os.SEEK_SET); err != nil {
		return nil, err
	}

	return &metadata.InstallInfo{
		Hash: sha.Sum(nil),
		Size: proto.Uint64(uint64(size)),
	}, nil
}
