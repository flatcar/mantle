// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package packet

import (
	"bytes"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

type console struct {
	pc   *cluster
	f    *os.File
	buf  bytes.Buffer
	done chan interface{}
	ssh  *ssh.Client
}

func (c *console) SSHClient(ip, user string) (*ssh.Client, error) {
	client, err := c.pc.UserSSHClient(ip, user)
	if err != nil {
		return nil, fmt.Errorf("getting SSH client: %w", err)
	}

	c.ssh = client
	return client, nil
}

func (c *console) CloseSSH() error {
	if c.ssh == nil {
		return nil
	}

	if err := c.ssh.Close(); err != nil {
		return fmt.Errorf("closing SSH client: %w", err)
	}

	return nil
}

func (c *console) Write(p []byte) (int, error) {
	c.buf.Write(p)
	return c.f.Write(p)
}

func (c *console) Close() error {
	close(c.done)
	return c.f.Close()
}

func (c *console) Output() string {
	<-c.done
	return c.buf.String()
}
