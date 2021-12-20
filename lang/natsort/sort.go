// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package natsort

import (
	"sort"
)

// Less determines if a naturally comes before b. Unlike Compare it will
// fall back to a normal string comparison which considers spaces if the
// two would otherwise be equal. That helps ensure the stability of sorts.
func Less(a, b string) bool {
	if r := Compare(a, b); r == 0 {
		return a < b
	} else {
		return r < 0
	}
}

// Strings natural sorts a slice of strings.
func Strings(s []string) {
	sort.Slice(s, func(i, j int) bool {
		return Less(s[i], s[j])
	})
}

// StringsAreSorted tests whether a slice of strings is natural sorted.
func StringsAreSorted(s []string) bool {
	return sort.SliceIsSorted(s, func(i, j int) bool {
		return Less(s[i], s[j])
	})
}
