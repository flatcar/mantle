// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package reader

import (
	"io"
	"strings"
	"testing"
)

func TestAtReader(t *testing.T) {
	ra := strings.NewReader("this is a test")
	rd := atReader{ra, 0}

	buf := make([]byte, 4)
	_, err := rd.Read(buf)
	if err != nil {
		t.Error(err)
	}
	if string(buf) != "this" {
		t.Errorf("Unexpected: %q", string(buf))
	}

	_, err = rd.Read(buf)
	if err != nil {
		t.Error(err)
	}
	if string(buf) != " is " {
		t.Errorf("Unexpected: %q", string(buf))
	}

	r := AtReader(ra)
	switch typ := r.(type) {
	case *strings.Reader:
	default:
		t.Errorf("Unexpected type %T", typ)
	}

	var rn io.ReaderAt
	r = AtReader(rn)
	switch typ := r.(type) {
	case *atReader:
	default:
		t.Errorf("Unexpected type %T", typ)
	}
}
