// Copyright 2026 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reporters

import (
	"errors"
	"testing"
	"time"

	"github.com/flatcar/mantle/harness/testresult"
)

type fakeReporter struct {
	outputErr   error
	outputCalls int
	lastResult  testresult.TestResult
	lastTest    string
}

func (f *fakeReporter) ReportTest(name string, result testresult.TestResult, duration time.Duration, b []byte) {
	f.lastTest = name
}

func (f *fakeReporter) Output(path string) error {
	f.outputCalls++
	return f.outputErr
}

func (f *fakeReporter) SetResult(result testresult.TestResult) {
	f.lastResult = result
}

func TestReportersFanOut(t *testing.T) {
	a, b := &fakeReporter{}, &fakeReporter{}
	reps := Reporters{a, b}

	reps.ReportTest("some-test", testresult.Pass, time.Second, nil)
	reps.SetResult(testresult.Pass)

	if a.lastTest != "some-test" || b.lastTest != "some-test" {
		t.Error("ReportTest was not forwarded to every reporter")
	}
	if a.lastResult != testresult.Pass || b.lastResult != testresult.Pass {
		t.Error("SetResult was not forwarded to every reporter")
	}
}

func TestReportersOutputStopsAtFirstError(t *testing.T) {
	failing := &fakeReporter{outputErr: errors.New("disk full")}
	skipped := &fakeReporter{}
	reps := Reporters{failing, skipped}

	if err := reps.Output("/tmp"); err == nil {
		t.Fatal("expected an error from the failing reporter")
	}
	if skipped.outputCalls != 0 {
		t.Error("Output was called on a reporter after an earlier one failed")
	}
}

func TestReportersOutputCallsAllOnSuccess(t *testing.T) {
	a, b := &fakeReporter{}, &fakeReporter{}
	reps := Reporters{a, b}

	if err := reps.Output("/tmp"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.outputCalls != 1 || b.outputCalls != 1 {
		t.Error("Output was not called on every reporter")
	}
}
