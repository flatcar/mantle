// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package packet

import (
	"bytes"
	"os"

	"golang.org/x/crypto/ssh"
)

type console struct {
	pc   *cluster
	f    *os.File
	buf  bytes.Buffer
	done chan interface{}
}

func (c *console) SSHClient(ip, user string) (*ssh.Client, error) {
	return c.pc.UserSSHClient(ip, user)
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
