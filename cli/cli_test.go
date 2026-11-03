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

package cli

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

func TestWrapPreRunCallsWrappedPersistentPreRun(t *testing.T) {
	var called bool
	root := &cobra.Command{
		Use: "root",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			called = true
		},
	}

	WrapPreRun(root, func(cmd *cobra.Command, args []string) error {
		return nil
	})

	if root.PersistentPreRun != nil {
		t.Fatal("PersistentPreRun should have been replaced with PersistentPreRunE")
	}
	if err := root.PersistentPreRunE(root, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("wrapped PersistentPreRun was not called")
	}
}

func TestWrapPreRunCallsWrappedPersistentPreRunE(t *testing.T) {
	var called bool
	root := &cobra.Command{
		Use: "root",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			called = true
			return nil
		},
	}

	WrapPreRun(root, func(cmd *cobra.Command, args []string) error {
		return nil
	})

	if err := root.PersistentPreRunE(root, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("wrapped PersistentPreRunE was not called")
	}
}

func TestWrapPreRunStopsAtFuncError(t *testing.T) {
	var wrappedCalled bool
	root := &cobra.Command{
		Use: "root",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			wrappedCalled = true
		},
	}

	wantErr := errors.New("boom")
	WrapPreRun(root, func(cmd *cobra.Command, args []string) error {
		return wantErr
	})

	if err := root.PersistentPreRunE(root, nil); err != wantErr {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if wrappedCalled {
		t.Error("wrapped PersistentPreRun should not run after f returns an error")
	}
}
