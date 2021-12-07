// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package ns

import (
	"runtime"

	"github.com/vishvananda/netns"
)

// NsEnter locks the current goroutine the OS thread and switches to a
// new network namespace. The returned function must be called in order
// to restore the previous state and unlock the thread.
func Enter(ns netns.NsHandle) (func() error, error) {
	runtime.LockOSThread()

	origns, err := netns.Get()
	if err != nil {
		runtime.UnlockOSThread()
		return nil, err
	}

	err = netns.Set(ns)
	if err != nil {
		origns.Close()
		runtime.UnlockOSThread()
		return nil, err
	}

	return func() error {
		defer runtime.UnlockOSThread()
		defer origns.Close()
		if err := netns.Set(origns); err != nil {
			return err
		}
		return nil
	}, nil
}

// NsCreate returns a handle to a new network namespace.
// NsEnter must be used to safely enter and exit the new namespace.
func Create() (netns.NsHandle, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origns, err := netns.Get()
	if err != nil {
		return netns.None(), err
	}
	defer origns.Close()
	defer netns.Set(origns)

	return netns.New()
}
