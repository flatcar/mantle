// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

// This example program illustrates how to create a custom test suite
// based on the harness package. main.go contains the test suite glue
// while tests.go contains example tests.
//
// The custom test suite adds a feature to give individual tests some
// data that can be overridden in the environment. When executed:
//
//	./example  -v
//	=== RUN   LogIt
//	--- PASS: LogIt (0.00s)
//	        tests.go:27: Got "something"
//	=== RUN   SkipIt
//	--- SKIP: SkipIt (0.00s)
//	        main.go:40: Missing "TEST_DATA_else" in environment.
//	PASS
//
package main
