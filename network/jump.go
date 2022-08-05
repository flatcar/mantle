// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package network

import (
	"fmt"
	"io/ioutil"
	"net"

	"golang.org/x/crypto/ssh"
)

var _ Dialer = (*JumpDialer)(nil)

// JumpDialer is a RetryDialer from a proxy.
type JumpDialer struct {
	RetryDialer
	user    string
	addr    string
	keyfile string
}

// NewJumpDialer initializes a JumpDialer to establish a SSH Proxy Jump.
func NewJumpDialer(addr, user, keyfile string) *JumpDialer {
	return &JumpDialer{
		RetryDialer: RetryDialer{
			Dialer: &net.Dialer{
				Timeout:   DefaultTimeout,
				KeepAlive: DefaultKeepAlive,
			},
			Retries: DefaultRetries,
		},
		user:    user,
		addr:    addr,
		keyfile: keyfile,
	}
}

// Dial connects to a remote address, retrying on failure.
func (d *JumpDialer) Dial(network, address string) (c net.Conn, err error) {
	key, err := ioutil.ReadFile(d.keyfile)
	if err != nil {
		return nil, fmt.Errorf("reading private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	cfg := &ssh.ClientConfig{
		User: d.user,
		// this is only used for testing - it's ok to live with that.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}

	addr := ensurePortSuffix(d.addr, defaultPort)
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating SSH client: %w", err)
	}

	return client.Dial(network, address)
}
