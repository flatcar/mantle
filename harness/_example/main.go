// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/flatcar-linux/mantle/harness"
)

type X struct {
	*harness.H
	defaults map[string]string
}

func (x *X) Option(key string) string {
	env := "TEST_DATA_" + key
	if value := os.Getenv(env); value != "" {
		return value
	}

	if value, ok := x.defaults[key]; ok {
		return value
	}

	x.Skipf("Missing %q in environment.", env)
	return ""
}

type Test struct {
	Name     string
	Run      func(x *X)
	Defaults map[string]string
}

var tests harness.Tests

func Register(test Test) {
	// copy map to prevent surprises
	defaults := make(map[string]string)
	for k, v := range test.Defaults {
		defaults[k] = v
	}

	tests.Add(test.Name, func(h *harness.H) {
		test.Run(&X{H: h, defaults: defaults})
	})
}

func main() {
	opts := harness.Options{OutputDir: "_x_temp"}
	opts.FlagSet("", flag.ExitOnError).Parse(os.Args[1:])

	suite := harness.NewSuite(opts, tests)
	if err := suite.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Println("FAIL")
		os.Exit(1)
	}
	fmt.Println("PASS")
	os.Exit(0)
}
