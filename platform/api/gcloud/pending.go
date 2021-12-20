// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package gcloud

import (
	"fmt"
	"time"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

type doable interface {
	Do(opts ...googleapi.CallOption) (*compute.Operation, error)
}

type Pending struct {
	Interval time.Duration
	Timeout  time.Duration // for default progress function
	Progress func(desc string, elapsed time.Duration, op *compute.Operation) error

	desc string
	do   doable
}

func (a *API) NewPending(desc string, do doable) *Pending {
	pending := &Pending{
		Interval: 10 * time.Second,
		Timeout:  10 * time.Minute,
		desc:     desc,
		do:       do,
	}
	pending.Progress = pending.defaultProgress
	return pending
}

func (p *Pending) Wait() error {
	var op *compute.Operation
	var err error
	failures := 0
	start := time.Now()
	for {
		op, err = p.do.Do()
		if err == nil {
			err := p.Progress(p.desc, time.Now().Sub(start), op)
			if err != nil {
				return err
			}
		} else {
			failures++
			if failures > 5 {
				return fmt.Errorf("Fetching %q status failed: %v", p.desc, err)
			}
		}
		if op != nil && op.Status == "DONE" {
			break
		}
		time.Sleep(p.Interval)
	}
	if op.Error != nil {
		if len(op.Error.Errors) > 0 {
			return fmt.Errorf("Operation %q failed: %+v", p.desc, op.Error.Errors)
		}
		return fmt.Errorf("Operation %q failed to start", p.desc)
	}
	return nil
}

func (p *Pending) defaultProgress(desc string, elapsed time.Duration, op *compute.Operation) error {
	var err error
	switch op.Status {
	case "PENDING", "RUNNING":
		err = fmt.Errorf("Operation %q is %q", desc, op.Status)
	case "DONE":
		return nil
	default:
		err = fmt.Errorf("Unknown operation status %q for %q: %+v", op.Status, desc, op)
	}

	if p.Timeout > 0 && elapsed > p.Timeout {
		return fmt.Errorf("Failed to wait for operation %q: %v", desc, err)
	}

	return nil
}
