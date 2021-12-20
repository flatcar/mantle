// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package destructor

import (
	"io"

	"github.com/coreos/pkg/capnslog"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "lang/destructor")
)

// Destructor is a common interface for objects that need to be cleaned up.
type Destructor interface {
	Destroy()
}

// CloseDestructor wraps any Closer to provide the Destructor interface.
type CloserDestructor struct {
	io.Closer
}

func (c CloserDestructor) Destroy() {
	if err := c.Close(); err != nil {
		plog.Errorf("Close() returned error: %v", err)
	}
}

// MultiDestructor wraps multiple Destructors for easy cleanup.
type MultiDestructor []Destructor

func (m MultiDestructor) Destroy() {
	for _, d := range m {
		d.Destroy()
	}
}

func (m *MultiDestructor) AddCloser(closer io.Closer) {
	m.AddDestructor(CloserDestructor{closer})
}

func (m *MultiDestructor) AddDestructor(destructor Destructor) {
	*m = append(*m, destructor)
}
