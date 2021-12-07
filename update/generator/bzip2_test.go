// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package generator

import (
	"bytes"
	"compress/bzip2"
	"io/ioutil"
	"testing"

	"github.com/flatcar-linux/mantle/system/exec"
)

func bunzip2(t *testing.T, z []byte) []byte {
	b, err := ioutil.ReadAll(bzip2.NewReader(bytes.NewReader(z)))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestBzip2(t *testing.T) {
	smallOnes, err := Bzip2(testOnes)
	if err != nil {
		if exec.IsCmdNotFound(err) {
			t.Skip(err)
		}

		t.Fatal(err)
	}

	bigOnes := bunzip2(t, smallOnes)
	if !bytes.Equal(bigOnes, testOnes) {
		t.Fatal("bzip2 corrupted the data")
	}
}
