// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"golang.org/x/net/context"
)

// Parallel executes a set of Workers and waits for them to finish.
func Parallel(ctx context.Context, workers ...Worker) error {
	wg := NewWorkerGroup(ctx, len(workers))
	for _, worker := range workers {
		if err := wg.Start(worker); err != nil {
			return wg.WaitError(err)
		}
	}
	return wg.Wait()
}
