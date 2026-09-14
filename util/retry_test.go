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

package util

import (
	"errors"
	"testing"
	"time"
)

func TestRetrySucceedsImmediately(t *testing.T) {
	calls := 0
	err := Retry(3, time.Millisecond, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetrySucceedsAfterFailures(t *testing.T) {
	calls := 0
	err := Retry(3, time.Millisecond, func() error {
		calls++
		if calls < 3 {
			return errors.New("not yet")
		}
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryExhaustsAttempts(t *testing.T) {
	calls := 0
	wantErr := errors.New("always fails")
	err := Retry(3, time.Millisecond, func() error {
		calls++
		return wantErr
	})
	if err != wantErr {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryConditionalStopsWhenNotRetryable(t *testing.T) {
	calls := 0
	wantErr := errors.New("fatal")
	err := RetryConditional(5, time.Millisecond, func(err error) bool {
		return false
	}, func() error {
		calls++
		return wantErr
	})
	if err != wantErr {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestWaitUntilReadySucceeds(t *testing.T) {
	calls := 0
	err := WaitUntilReady(time.Second, time.Millisecond, func() (bool, error) {
		calls++
		return calls >= 3, nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestWaitUntilReadyPropagatesError(t *testing.T) {
	wantErr := errors.New("check failed")
	err := WaitUntilReady(time.Second, time.Millisecond, func() (bool, error) {
		return false, wantErr
	})
	if err != wantErr {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
}

func TestWaitUntilReadyTimesOut(t *testing.T) {
	err := WaitUntilReady(10*time.Millisecond, 2*time.Millisecond, func() (bool, error) {
		return false, nil
	})
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}
