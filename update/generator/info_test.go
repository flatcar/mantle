// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"bytes"
	"testing"
)

func TestEmptyInstallInfo(t *testing.T) {
	info, err := NewInstallInfo(bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatal(err)
	}

	if info.Size == nil {
		t.Error("InstallInfo.Size is nil")
	} else if *info.Size != 0 {
		t.Errorf("InstallInfo.Size should be 0, got %d", *info.Size)
	}

	if !bytes.Equal(info.Hash, testEmptyHash) {
		t.Errorf("InstallInfo.Hash should be %q, got %q", testEmptyHash, info.Hash)
	}
}

func TestOnesInstallInfo(t *testing.T) {
	info, err := NewInstallInfo(bytes.NewReader(testOnes))
	if err != nil {
		t.Fatal(err)
	}

	if info.Size == nil {
		t.Error("InstallInfo.Size is nil")
	} else if *info.Size != BlockSize {
		t.Errorf("InstallInfo.Size should be %d, got %d", BlockSize, *info.Size)
	}

	if !bytes.Equal(info.Hash, testOnesHash) {
		t.Errorf("InstallInfo.Hash should be %q, got %q", testOnesHash, info.Hash)
	}
}

func TestUnalignedInstallInfo(t *testing.T) {
	info, err := NewInstallInfo(bytes.NewReader(testUnaligned))
	if err != nil {
		t.Fatal(err)
	}

	if info.Size == nil {
		t.Error("InstallInfo.Size is nil")
	} else if *info.Size != BlockSize+1 {
		t.Errorf("InstallInfo.Size should be %d, got %d", BlockSize, *info.Size)
	}

	if !bytes.Equal(info.Hash, testUnalignedHash) {
		t.Errorf("InstallInfo.Hash should be %q, got %q", testUnalignedHash, info.Hash)
	}
}
