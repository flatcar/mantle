// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package harness

import (
	"fmt"

	"github.com/flatcar-linux/mantle/lang/maps"
)

// Test is a single test function.
type Test func(*H)

// Tests is a set of test functions that can be given to a Suite.
type Tests map[string]Test

// Add inserts the given Test into the set, initializing Tests if needed.
// If a test with the given name already exists Add will panic.
func (ts *Tests) Add(name string, test Test) {
	if *ts == nil {
		*ts = make(Tests)
	} else if _, ok := (*ts)[name]; ok {
		panic(fmt.Errorf("harness: duplicate test %q", name))
	}
	(*ts)[name] = test
}

// List returns a sorted list of test names.
func (ts Tests) List() []string {
	return maps.NaturalKeys(ts)
}
