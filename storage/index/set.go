// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package index

import (
	"strings"

	gs "google.golang.org/api/storage/v1"

	"github.com/flatcar-linux/mantle/storage"
)

type IndexSet map[string]struct{}

func NewIndexSet(bucket *storage.Bucket) IndexSet {
	is := IndexSet(make(map[string]struct{}))

	for _, prefix := range bucket.Prefixes() {
		is[prefix] = struct{}{}
		is[strings.TrimSuffix(prefix, "/")] = struct{}{}
		is[prefix+"index.html"] = struct{}{}
	}

	return is
}

func (is IndexSet) IsIndex(obj *gs.Object) bool {
	_, isIndex := is[obj.Name]
	return isIndex
}

func (is IndexSet) NotIndex(obj *gs.Object) bool {
	return !is.IsIndex(obj)
}
