// Copyright 2026 The Flatcar Maintainers.
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

package platform

import "testing"

func TestSystemRunningState(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantState string
		wantKeep  bool
	}{
		{"running", "running", "running", true},
		{"running with trailing newline", "running\n", "running", true},
		{"initializing", "initializing", "initializing", true},
		{"starting", "starting", "starting", true},
		{"stopping", "stopping", "stopping", true},
		{"degraded", "degraded", "degraded", false},
		{"maintenance", "maintenance", "maintenance", false},
		{"offline", "offline", "offline", false},
		// A blank/empty SSH response (e.g. a transient connection hiccup)
		// must not be mistaken for an in-progress state.
		{"empty output", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotState, gotKeep := systemRunningState([]byte(tt.raw))
			if gotState != tt.wantState || gotKeep != tt.wantKeep {
				t.Errorf("systemRunningState(%q) = (%q, %v), want (%q, %v)",
					tt.raw, gotState, gotKeep, tt.wantState, tt.wantKeep)
			}
		})
	}
}
