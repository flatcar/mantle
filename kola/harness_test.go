// Copyright 2026 The Flatcar Maintainers
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

package kola

import "testing"

func TestArchitecture(t *testing.T) {
	origBoard := QEMUOptions.Board
	defer func() { QEMUOptions.Board = origBoard }()

	tests := []struct {
		platform string
		board    string
		want     string
	}{
		{"qemu", "amd64-usr", "amd64"},
		{"qemu", "arm64-usr", "arm64"},
		// qemu-unpriv shares QEMUOptions with qemu, so it must resolve the
		// board's arch the same way (regression test for the wrong-arch kolet).
		{"qemu-unpriv", "amd64-usr", "amd64"},
		{"qemu-unpriv", "arm64-usr", "arm64"},
	}

	for _, tt := range tests {
		QEMUOptions.Board = tt.board
		if got := architecture(tt.platform); got != tt.want {
			t.Errorf("architecture(%q) with board %q = %q, want %q", tt.platform, tt.board, got, tt.want)
		}
	}
}
