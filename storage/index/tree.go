// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package index

import (
	"strings"

	gs "google.golang.org/api/storage/v1"

	"github.com/flatcar-linux/mantle/lang/natsort"
	"github.com/flatcar-linux/mantle/storage"
)

type IndexTree struct {
	bucket   *storage.Bucket
	prefixes map[string]bool
	subdirs  map[string][]string
	objects  map[string][]*gs.Object
}

func NewIndexTree(bucket *storage.Bucket, includeEmpty bool) *IndexTree {
	t := &IndexTree{
		bucket:   bucket,
		prefixes: make(map[string]bool),
		subdirs:  make(map[string][]string),
		objects:  make(map[string][]*gs.Object),
	}

	for _, prefix := range bucket.Prefixes() {
		if includeEmpty {
			t.addDir(prefix)
		} else {
			t.prefixes[prefix] = false // initialize as empty
		}
	}

	indexes := NewIndexSet(bucket)
	for _, obj := range bucket.Objects() {
		if indexes.NotIndex(obj) {
			t.addObj(obj)
		}
	}

	for _, dirs := range t.subdirs {
		natsort.Strings(dirs)
	}

	for _, objs := range t.objects {
		storage.SortObjects(objs)
	}

	return t
}

func (t *IndexTree) addObj(obj *gs.Object) {
	prefix := storage.NextPrefix(obj.Name)
	t.objects[prefix] = append(t.objects[prefix], obj)
	t.addDir(prefix)
}

func (t *IndexTree) addDir(prefix string) {
	for !t.prefixes[prefix] {
		t.prefixes[prefix] = true // mark as not empty
		if prefix == "" {
			return
		}
		parent := storage.NextPrefix(prefix)
		t.subdirs[parent] = append(t.subdirs[parent], prefix)
		prefix = storage.NextPrefix(prefix)
	}
}

func (t *IndexTree) Prefixes(dir string) []string {
	prefixes := make([]string, 0, len(t.prefixes))
	for prefix := range t.prefixes {
		if strings.HasPrefix(prefix, dir) {
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes
}
