// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package network

import (
	"fmt"
	"io/ioutil"

	"golang.org/x/crypto/ssh"
)

// NewJumpDialer initializes a RetryDialer with SSH Proxy Jump.
func NewJumpDialer(addr, user, keyfile string) (*RetryDialer, error) {
	key, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, fmt.Errorf("reading private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	cfg := &ssh.ClientConfig{
		User: user,
		// this is only used for testing - it's ok to live with that.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}

	addr = ensurePortSuffix(addr, defaultPort)
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating SSH client: %w", err)
	}

	return &RetryDialer{
		Dialer:  client,
		Retries: DefaultRetries,
	}, nil
}
