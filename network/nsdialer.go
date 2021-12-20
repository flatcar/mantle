// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"net"

	"github.com/vishvananda/netns"

	"github.com/flatcar-linux/mantle/system/ns"
)

// NsDialer is a RetryDialer that can enter any network namespace.
type NsDialer struct {
	RetryDialer
	NsHandle netns.NsHandle
}

func NewNsDialer(ns netns.NsHandle) *NsDialer {
	return &NsDialer{
		RetryDialer: RetryDialer{
			Dialer: &net.Dialer{
				Timeout:   DefaultTimeout,
				KeepAlive: DefaultKeepAlive,
			},
			Retries: DefaultRetries,
		},
		NsHandle: ns,
	}
}

func (d *NsDialer) Dial(network, address string) (net.Conn, error) {
	nsExit, err := ns.Enter(d.NsHandle)
	if err != nil {
		return nil, err
	}
	defer nsExit()

	return d.RetryDialer.Dial(network, address)
}
