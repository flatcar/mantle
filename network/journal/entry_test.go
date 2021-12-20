// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package journal

import (
	"testing"
	"time"
)

func TestEntryRealtime(t *testing.T) {
	for _, testcase := range []struct {
		name   string
		entry  Entry
		expect time.Time
	}{{
		name:   "zero",
		entry:  Entry{},
		expect: time.Time{},
	}, {
		name: "has_source",
		entry: Entry{
			FIELD_SOURCE_REALTIME_TIMESTAMP: []byte("1342540861416351"),
			FIELD_REALTIME_TIMESTAMP:        []byte("1342540861416409"),
		},
		expect: time.Unix(1342540861, 416351000),
	}, {
		name: "no_source",
		entry: Entry{
			FIELD_REALTIME_TIMESTAMP: []byte("1342540861416409"),
		},
		expect: time.Unix(1342540861, 416409000),
	}, {
		name: "invalid_source",
		entry: Entry{
			FIELD_SOURCE_REALTIME_TIMESTAMP: []byte("bob"),
			FIELD_REALTIME_TIMESTAMP:        []byte("1342540861416409"),
		},
		expect: time.Time{},
	}, {
		name: "invalid_both",
		entry: Entry{
			FIELD_REALTIME_TIMESTAMP: []byte("bob"),
		},
		expect: time.Time{},
	},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			result := testcase.entry.Realtime()
			if result != testcase.expect {
				t.Errorf("%s != %s", result, testcase.expect)
			}
		})
	}
}
