// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package maps

import (
	"sort"
	"testing"

	"github.com/flatcar-linux/mantle/lang/natsort"
)

var (
	// some random goo
	testKeys = []string{
		"100uquie",
		"10ocheiv",
		"1hiexieh",
		"cheuzash",
		"ohbohmop",
		"oobeecoh",
		"ohxadupu",
		"yuilohsh",
		"oongoojo",
		"mielutao",
		"iriecier",
		"eisheiba",
		"ahsoogup",
		"aabeevie",
		"aeyaebek",
		"kaibahgh",
	}
)

func TestBadKeys(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("Keys did not panic")
		} else if r != "maps: keys must be strings" {
			panic(r)
		}
	}()
	Keys(map[int]int{})
}

func TestSortedKeys(t *testing.T) {
	testMap := make(map[string]bool)
	for _, k := range testKeys {
		testMap[k] = true
	}

	// test is pointless if map iterates in-order by random chance
	mapKeys := make([]string, 0, len(testMap))
	for k, _ := range testMap {
		mapKeys = append(mapKeys, k)
	}
	if sort.StringsAreSorted(mapKeys) {
		t.Skip("map is already iterating in order!")
	}

	sortedKeys := SortedKeys(testMap)
	if !sort.StringsAreSorted(sortedKeys) {
		t.Error("SortedKeys did not sort the keys!")
	}

	if len(sortedKeys) != len(testKeys) {
		t.Errorf("SortedKeys returned %d keys, not %d",
			len(sortedKeys), len(testKeys))
	}
}

func TestNaturalKeys(t *testing.T) {
	testMap := make(map[string]bool)
	for _, k := range testKeys {
		testMap[k] = true
	}

	// test is pointless if map iterates in-order by random chance
	mapKeys := make([]string, 0, len(testMap))
	for k, _ := range testMap {
		mapKeys = append(mapKeys, k)
	}
	if natsort.StringsAreSorted(mapKeys) {
		t.Skip("map is already iterating in order!")
	}

	sortedKeys := NaturalKeys(testMap)
	if !natsort.StringsAreSorted(sortedKeys) {
		t.Error("SortedKeys did not sort the keys!")
	}

	if len(sortedKeys) != len(testKeys) {
		t.Errorf("SortedKeys returned %d keys, not %d",
			len(sortedKeys), len(testKeys))
	}
}
