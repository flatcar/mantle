// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"time"
)

// Retry calls function f until it has been called attemps times, or succeeds.
// Retry delays for delay between calls of f. If f does not succeed after
// attempts calls, the error from the last call is returned.
func Retry(attempts int, delay time.Duration, f func() error) error {
	return RetryConditional(attempts, delay, func(_ error) bool { return true }, f)
}

// RetryConditional calls function f until it has been called attemps times, or succeeds.
// Retry delays for delay between calls of f. If f does not succeed after
// attempts calls, the error from the last call is returned.
// If shouldRetry returns false on the error generated, RetryConditional stops immediately
// and returns the error
func RetryConditional(attempts int, delay time.Duration, shouldRetry func(err error) bool, f func() error) error {
	var err error

	for i := 0; i < attempts; i++ {
		err = f()
		if err == nil || !shouldRetry(err) {
			break
		}

		if i < attempts-1 {
			time.Sleep(delay)
		}
	}

	return err
}

func WaitUntilReady(timeout, delay time.Duration, checkFunction func() (bool, error)) error {
	after := time.After(timeout)
	for {
		select {
		case <-after:
			return fmt.Errorf("time limit exceeded")
		default:
		}

		time.Sleep(delay)

		done, err := checkFunction()
		if err != nil {
			return err
		}

		if done {
			break
		}
	}
	return nil
}
