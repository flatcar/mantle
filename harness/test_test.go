// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package harness

import (
	"reflect"
	"testing"
)

func TestTestsAdd(t *testing.T) {
	var ts Tests
	ts.Add("test1", nil)
	ts.Add("test2", nil)
	expect := Tests(map[string]Test{"test1": nil, "test2": nil})
	if !reflect.DeepEqual(ts, expect) {
		t.Errorf("got %v wanted %v", ts, expect)
	}
}

func TestTestsAddDup(t *testing.T) {
	var ts Tests
	ts.Add("test1", nil)
	defer func() {
		if recover() == nil {
			t.Errorf("Add did not panic")
		}
	}()
	ts.Add("test1", nil)
}

func TestTestsList(t *testing.T) {
	ts := Tests(map[string]Test{"test01": nil, "test2": nil})
	list := ts.List()
	expect := []string{"test01", "test2"}
	if !reflect.DeepEqual(list, expect) {
		t.Errorf("got %v wanted %v", list, expect)
	}
}
