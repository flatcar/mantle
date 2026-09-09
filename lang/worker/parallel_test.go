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

package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestParallelRunsAllWorkers(t *testing.T) {
	var count int32
	workers := make([]Worker, 5)
	for i := range workers {
		workers[i] = func(ctx context.Context) error {
			atomic.AddInt32(&count, 1)
			return nil
		}
	}

	if err := Parallel(context.Background(), workers...); err != nil {
		t.Fatalf("Parallel returned error: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5 workers to run, got %d", count)
	}
}

func TestParallelReturnsWorkerError(t *testing.T) {
	wantErr := errors.New("boom")
	workers := []Worker{
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return wantErr },
		func(ctx context.Context) error { return nil },
	}

	err := Parallel(context.Background(), workers...)
	if !hasError(err, wantErr) {
		t.Errorf("expected Parallel to return %v, got %v", wantErr, err)
	}
}

func TestParallelWithNoWorkers(t *testing.T) {
	if err := Parallel(context.Background()); err != nil {
		t.Errorf("expected no error for an empty worker list, got %v", err)
	}
}
