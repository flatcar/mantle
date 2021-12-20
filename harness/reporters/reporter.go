// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package reporters

import (
	"time"

	"github.com/flatcar-linux/mantle/harness/testresult"
)

type Reporters []Reporter

func (reps Reporters) ReportTest(name string, result testresult.TestResult, duration time.Duration, b []byte) {
	for _, r := range reps {
		r.ReportTest(name, result, duration, b)
	}
}

func (reps Reporters) Output(path string) error {
	for _, r := range reps {
		err := r.Output(path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (reps Reporters) SetResult(s testresult.TestResult) {
	for _, r := range reps {
		r.SetResult(s)
	}
}

type Reporter interface {
	ReportTest(string, testresult.TestResult, time.Duration, []byte)
	Output(string) error
	SetResult(testresult.TestResult)
}
