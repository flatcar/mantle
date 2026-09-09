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
	"time"

	"github.com/coreos/pkg/multierror"
)

func hasError(err, want error) bool {
	if errors.Is(err, want) {
		return true
	}
	me, ok := err.(multierror.Error)
	if !ok {
		return false
	}
	for _, e := range me {
		if errors.Is(e, want) {
			return true
		}
	}
	return false
}

func TestWorkerGroupRunsAllWorkers(t *testing.T) {
	wg := NewWorkerGroup(context.Background(), 4)

	var count int32
	for i := 0; i < 10; i++ {
		err := wg.Start(func(ctx context.Context) error {
			atomic.AddInt32(&count, 1)
			return nil
		})
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	}

	if err := wg.Wait(); err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}
	if count != 10 {
		t.Errorf("expected 10 workers to run, got %d", count)
	}
}

func TestWorkerGroupEnforcesLimit(t *testing.T) {
	wg := NewWorkerGroup(context.Background(), 2)

	var running int32
	var maxRunning int32
	release := make(chan struct{})
	started := make(chan struct{})

	for i := 0; i < 5; i++ {
		go func() {
			started <- struct{}{}
			if err := wg.Start(func(ctx context.Context) error {
				n := atomic.AddInt32(&running, 1)
				for {
					old := atomic.LoadInt32(&maxRunning)
					if n <= old || atomic.CompareAndSwapInt32(&maxRunning, old, n) {
						break
					}
				}
				<-release
				atomic.AddInt32(&running, -1)
				return nil
			}); err != nil {
				t.Error(err)
			}
		}()
	}
	for i := 0; i < 5; i++ {
		<-started
	}

	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(&maxRunning); got > 2 {
		t.Errorf("expected at most 2 concurrent workers, saw %d", got)
	}

	close(release)
	if err := wg.Wait(); err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}
}

func TestWorkerGroupCapturesError(t *testing.T) {
	wg := NewWorkerGroup(context.Background(), 1)
	wantErr := errors.New("boom")

	if err := wg.Start(func(ctx context.Context) error {
		return wantErr
	}); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	err := wg.Wait()
	if err == nil {
		t.Fatal("expected Wait to return an error")
	}
	if !hasError(err, wantErr) {
		t.Errorf("expected error to wrap %v, got %v", wantErr, err)
	}
}

func TestWorkerGroupCancelsRemainingWorkOnError(t *testing.T) {
	wg := NewWorkerGroup(context.Background(), 1)
	wantErr := errors.New("boom")

	if err := wg.Start(func(ctx context.Context) error {
		return wantErr
	}); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	deadline := time.After(time.Second)
	for {
		err := wg.Start(func(ctx context.Context) error {
			t.Error("worker started after group was cancelled")
			return nil
		})
		if err != nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("group was never cancelled after a worker failed")
		default:
		}
	}

	if err := wg.Wait(); !hasError(err, wantErr) {
		t.Errorf("expected Wait to return %v, got %v", wantErr, err)
	}
}

func TestWaitErrorPrefersWaitFailure(t *testing.T) {
	wg := NewWorkerGroup(context.Background(), 1)
	waitErr := errors.New("wait failed")

	if err := wg.Start(func(ctx context.Context) error {
		return waitErr
	}); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	startErr := errors.New("start failed")
	err := wg.WaitError(startErr)
	if !hasError(err, waitErr) {
		t.Errorf("expected WaitError to return %v, got %v", waitErr, err)
	}
}

func TestWaitErrorReturnsGivenErrorWhenNoFailure(t *testing.T) {
	wg := NewWorkerGroup(context.Background(), 1)

	if err := wg.Start(func(ctx context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	startErr := errors.New("start failed")
	err := wg.WaitError(startErr)
	if !errors.Is(err, startErr) {
		t.Errorf("expected WaitError to return %v, got %v", startErr, err)
	}
}
