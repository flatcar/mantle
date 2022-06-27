// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package sshstorage

import (
	"bytes"
	"context"
	"fmt"

	"github.com/flatcar-linux/mantle/platform/api/equinixmetal/storage"

	"github.com/coreos/pkg/capnslog"
	"golang.org/x/crypto/ssh"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "platform/api/equinixmetal/storage/remote")
)

func New(client *ssh.Client, host, docRoot, protocol string) storage.Storage {
	return &Remote{
		client:   client,
		host:     host,
		docRoot:  docRoot,
		protocol: protocol,
	}
}

// Remote implements a Storage interface for remote server.
type Remote struct {
	client *ssh.Client

	// host for SSH connection and webserver
	host string

	// docRoot is the absolute path to the webserver document root (e.g /var/wwww)
	docRoot string

	// procol is the procol (http or https) for accessing the uploaded content
	protocol string
}

func (r *Remote) Upload(name, contentType string, data []byte) (string, string, error) {
	session, err := r.client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("creating SSH session: %w", err)
	}
	defer session.Close()

	// we define the file extension based on the contentType
	ext := "ign"
	if contentType == "application/octet-stream" {
		ext = "ipxe"
	}

	fName := fmt.Sprintf("mantle-%s.%s", name, ext)

	p := fmt.Sprintf("%s/%s", r.docRoot, fName)

	session.Stdin = bytes.NewReader(data)

	if err := session.Run(fmt.Sprintf("/bin/cat - > %s", p)); err != nil {
		return "", "", fmt.Errorf("writing data from standard input: %w", err)
	}

	plog.Debugf("%s uploaded to %s", fName, p)

	return p, fmt.Sprintf("%s://%s/%s", r.protocol, r.host, fName), nil
}

// Delete the uploaded data from remote storage.
func (r *Remote) Delete(ctx context.Context, name string) error {
	session, err := r.client.NewSession()
	if err != nil {
		return fmt.Errorf("creating SSH session: %w", err)
	}
	defer session.Close()

	if err := session.Run(fmt.Sprintf("/bin/rm --force %s", name)); err != nil {
		return fmt.Errorf("removing data from SSH session: %w", err)
	}

	plog.Debugf("%s deleted from remote storage", name)

	return nil
}

func (r *Remote) Close() error {
	if r.client != nil {
		if err := r.client.Close(); err != nil {
			return fmt.Errorf("closing SSH client: %w", err)
		}

		return nil
	}

	return nil
}
