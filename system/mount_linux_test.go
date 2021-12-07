// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"syscall"
	"testing"
)

func TestSplitFlags(t *testing.T) {
	data := []struct {
		opts  string
		flags uintptr
		extra string
	}{
		{"", 0, ""},
		{"nodev,nosuid,mode=755", syscall.MS_NOSUID | syscall.MS_NODEV, "mode=755"},
		{"mode=755,other", 0, "mode=755,other"},
		{"mode=755,nodev,other", syscall.MS_NODEV, "mode=755,other"},
	}

	for _, d := range data {
		f, e := splitFlags(d.opts)
		if f != d.flags {
			t.Errorf("bad flags for %q, got 0x%x wanted 0x%x", d.opts, f, d.flags)
		}
		if e != d.extra {
			t.Errorf("bad extra for %q, got %q wanted %q", d.opts, e, d.extra)
		}
	}
}
