// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package esx

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vmware/govmomi/nfc"
	"github.com/vmware/govmomi/vim25"
)

type leaseUpdater struct {
	lease *nfc.Lease

	pos   int64 // Number of bytes
	total int64 // Total number of bytes

	done chan struct{} // When lease updater should stop

	wg sync.WaitGroup // Track when update loop is done
}

func newLeaseUpdater(client *vim25.Client, lease *nfc.Lease, items []ovfFileItem) *leaseUpdater {
	l := leaseUpdater{
		lease: lease,

		done: make(chan struct{}),
	}

	for _, item := range items {
		l.total += item.item.Size
		go l.waitForProgress(item)
	}

	// Kickstart update loop
	l.wg.Add(1)
	go l.run()

	return &l
}

func (l *leaseUpdater) waitForProgress(item ovfFileItem) {
	var pos, total int64

	total = item.item.Size

	for {
		select {
		case <-l.done:
			return
		case p, ok := <-item.ch:
			// Return in case of error
			if ok && p.Error() != nil {
				return
			}

			if !ok {
				// Last element on the channel, add to total
				atomic.AddInt64(&l.pos, total-pos)
				return
			}

			// Approximate progress in number of bytes
			x := int64(float32(total) * (p.Percentage() / 100.0))
			atomic.AddInt64(&l.pos, x-pos)
			pos = x
		}
	}
}

func (l *leaseUpdater) run() {
	defer l.wg.Done()

	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-l.done:
			return
		case <-tick.C:
			// From the vim api HttpNfcLeaseProgress(percent) doc, percent ==
			// "Completion status represented as an integer in the 0-100 range."
			// Always report the current value of percent, as it will renew the
			// lease even if the value hasn't changed or is 0.
			percent := int32(float32(100*atomic.LoadInt64(&l.pos)) / float32(l.total))
			err := l.lease.Progress(context.TODO(), percent)
			if err != nil {
				plog.Debugf("from lease updater: %s\n", err)
			}
		}
	}
}

func (l *leaseUpdater) Done() {
	close(l.done)
	l.wg.Wait()
}
