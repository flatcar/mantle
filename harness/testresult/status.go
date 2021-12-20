// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package testresult

const (
	Fail TestResult = "FAIL"
	Skip TestResult = "SKIP"
	Pass TestResult = "PASS"
)

type TestResult string
