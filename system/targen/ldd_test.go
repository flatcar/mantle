// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package targen

import (
	"testing"
)

func TestLdd(t *testing.T) {
	bin := "/bin/sh"
	deps, err := ldd(bin)
	if err != nil {
		t.Fatal(err)
	}

	if len(deps) == 0 {
		t.Fatalf("no deps of %q", bin)
	}

	t.Logf("%+v", deps)
}
