// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"bytes"
	"fmt"
	"net"
	"testing"

	"golang.org/x/crypto/ssh"
)

var (
	testHostKeyBytes = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIBOwIBAAJBALdGZxkXDAjsYk10ihwU6Id2KeILz1TAJuoq4tOgDWxEEGeTrcld
r/ZwVaFzjWzxaf6zQIJbfaSEAhqD5yo72+sCAwEAAQJBAK8PEVU23Wj8mV0QjwcJ
tZ4GcTUYQL7cF4+ezTCE9a1NrGnCP2RuQkHEKxuTVrxXt+6OF15/1/fuXnxKjmJC
nxkCIQDaXvPPBi0c7vAxGwNY9726x01/dNbHCE0CBtcotobxpwIhANbbQbh3JHVW
2haQh4fAG5mhesZKAGcxTyv4mQ7uMSQdAiAj+4dzMpJWdSzQ+qGHlHMIBvVHLkqB
y2VdEyF7DPCZewIhAI7GOI/6LDIFOvtPo6Bj2nNmyQ1HU6k/LRtNIXi4c9NJAiAr
rrxx26itVhJmcvoUhOjwuzSlP2bE5VHAvkGB352YBg==
-----END RSA PRIVATE KEY-----
`)
)

func TestEnsurePortSuffix(t *testing.T) {
	tests := map[string]string{
		"host":          "host:22",
		"host:9":        "host:9",
		"[host]":        "[host]:22",
		"[host]:9":      "[host]:9",
		"::1":           "[::1]:22",
		"[::1]:9":       "[::1]:9",
		"127.0.0.1":     "127.0.0.1:22",
		"127.0.0.1:9":   "127.0.0.1:9",
		"[127.0.0.1]":   "[127.0.0.1]:22",
		"[127.0.0.1]:9": "[127.0.0.1]:9",
	}

	for input, expect := range tests {
		output := ensurePortSuffix(input, defaultPort)
		if output != expect {
			t.Errorf("Got result %q, expected %q", output, expect)
		}
	}
}

func TestSSHNewClient(t *testing.T) {
	m, err := NewSSHAgent(&net.Dialer{})
	if err != nil {
		t.Fatalf("NewSSHAgent failed: %v", err)
	}

	keys, err := m.List()
	if err != nil {
		t.Fatalf("Keys failed: %v", err)
	}

	cfg := ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if conn.User() == "core" && bytes.Equal(key.Marshal(), keys[0].Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("pubkey rejected")
		},
	}

	hostKey, err := ssh.ParsePrivateKey(testHostKeyBytes)
	if err != nil {
		t.Fatalf("ParsePrivateKey failed: %v", err)
	}
	cfg.AddHostKey(hostKey)

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	// Oh god... I give up for now.
	t.Skip("Implementation incomplete")
}
