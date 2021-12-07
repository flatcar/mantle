// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

func init() {
	defaults := map[string]string{
		"something": "something",
	}

	Register(Test{
		Name:     "LogIt",
		Defaults: defaults,
		Run: func(x *X) {
			s := x.Option("something")
			x.Logf("Got %q", s)
		},
	})

	Register(Test{
		Name:     "SkipIt",
		Defaults: defaults,
		Run: func(x *X) {
			s := x.Option("else")
			x.Errorf("Got %q", s)
		},
	})
}
