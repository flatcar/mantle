// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package maps

import (
	"reflect"
	"sort"

	"github.com/flatcar-linux/mantle/lang/natsort"
)

// Keys returns a map's keys as an unordered slice of strings.
func Keys(m interface{}) []string {
	mapValue := reflect.ValueOf(m)

	// Value.String() isn't sufficient to assert the keys are strings.
	if mapValue.Type().Key().Kind() != reflect.String {
		panic("maps: keys must be strings")
	}

	keyValues := mapValue.MapKeys()
	keys := make([]string, len(keyValues))
	for i, k := range keyValues {
		keys[i] = k.String()
	}

	return keys
}

// SortedKeys returns a map's keys as a sorted slice of strings.
func SortedKeys(m interface{}) []string {
	keys := Keys(m)
	sort.Strings(keys)
	return keys
}

// NaturalKeys returns a map's keys as a natural sorted slice of strings.
// See github.com/flatcar-linux/mantle/lang/natsort
func NaturalKeys(m interface{}) []string {
	keys := Keys(m)
	natsort.Strings(keys)
	return keys
}
