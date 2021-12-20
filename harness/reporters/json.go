// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package reporters

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/flatcar-linux/mantle/harness/testresult"
)

type jsonReporter struct {
	Tests    []jsonTest            `json:"tests"`
	Result   testresult.TestResult `json:"result"`
	filename string

	// Context variables
	Platform string `json:"platform"`
	Version  string `json:"version"`
}

type jsonTest struct {
	Name     string                `json:"name"`
	Result   testresult.TestResult `json:"result"`
	Duration time.Duration         `json:"duration"`
	Output   string                `json:"output"`
}

func NewJSONReporter(filename, platform, version string) *jsonReporter {
	return &jsonReporter{
		Platform: platform,
		Version:  version,
		filename: filename,
	}
}

func (r *jsonReporter) ReportTest(name string, result testresult.TestResult, duration time.Duration, b []byte) {
	r.Tests = append(r.Tests, jsonTest{
		Name:     name,
		Result:   result,
		Duration: duration,
		Output:   string(b),
	})
}

func (r *jsonReporter) Output(path string) error {
	f, err := os.Create(filepath.Join(path, r.filename))
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(r)
}

func (r *jsonReporter) SetResult(result testresult.TestResult) {
	r.Result = result
}
