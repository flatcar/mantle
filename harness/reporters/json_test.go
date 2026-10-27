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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flatcar/mantle/harness/testresult"
)

func TestJSONReporterOutput(t *testing.T) {
	r := NewJSONReporter("results.json", "qemu", "1.2.3")
	r.ReportTest("test-a", testresult.Pass, 2*time.Second, []byte("ok"))
	r.ReportTest("test-b", testresult.Fail, time.Second, []byte("boom"))
	r.SetResult(testresult.Fail)

	dir := t.TempDir()
	if err := r.Output(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "results.json"))
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	var decoded jsonReporter
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decoding output: %v", err)
	}

	if decoded.Platform != "qemu" || decoded.Version != "1.2.3" {
		t.Errorf("unexpected context: platform=%q version=%q", decoded.Platform, decoded.Version)
	}
	if decoded.Result != testresult.Fail {
		t.Errorf("unexpected overall result: %v", decoded.Result)
	}
	if len(decoded.Tests) != 2 {
		t.Fatalf("expected 2 tests, got %d", len(decoded.Tests))
	}
	if decoded.Tests[0].Name != "test-a" || decoded.Tests[0].Result != testresult.Pass {
		t.Errorf("unexpected first test: %+v", decoded.Tests[0])
	}
	if decoded.Tests[1].Name != "test-b" || decoded.Tests[1].Result != testresult.Fail {
		t.Errorf("unexpected second test: %+v", decoded.Tests[1])
	}
}

func TestJSONReporterOutputInvalidPath(t *testing.T) {
	r := NewJSONReporter("results.json", "qemu", "1.2.3")
	if err := r.Output(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Error("expected an error writing to a nonexistent directory")
	}
}
