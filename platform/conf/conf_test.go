// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package conf

import (
	"net"
	"strings"
	"testing"

	"github.com/flatcar-linux/mantle/network"
)

func TestConfCopyKey(t *testing.T) {
	agent, err := network.NewSSHAgent(&net.Dialer{})
	if err != nil {
		t.Fatalf("NewSSHAgent failed: %v", err)
	}

	keys, err := agent.List()
	if err != nil {
		t.Fatalf("agent.List failed: %v", err)
	}

	tests := []*UserData{
		ContainerLinuxConfig(""),
		Ignition(`{ "ignition": { "version": "2.2.0" } }`),
		Ignition(`{ "ignition": { "version": "2.1.0" } }`),
		Ignition(`{ "ignition": { "version": "2.0.0" } }`),
		Ignition(`{ "ignitionVersion": 1 }`),
		CloudConfig("#cloud-config"),
	}

	for i, tt := range tests {
		conf, err := tt.Render("")
		if err != nil {
			t.Errorf("failed to parse config %d: %v", i, err)
			continue
		}

		conf.CopyKeys(keys)

		str := conf.String()

		if !strings.Contains(str, "ssh-rsa ") || !strings.Contains(str, " core@default") {
			t.Errorf("ssh public key not found in config %d: %s", i, str)
			continue
		}
	}
}
