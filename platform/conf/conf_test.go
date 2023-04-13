// Copyright 2015 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package conf

import (
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/flatcar/mantle/network"
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
		Ignition(`{ "ignition": { "version": "3.0.0" } }`),
		Ignition(`{ "ignition": { "version": "3.1.0" } }`),
		Ignition(`{ "ignition": { "version": "3.2.0" } }`),
		Ignition(`{ "ignition": { "version": "3.3.0" } }`),
		Ignition(`{ "ignitionVersion": 1 }`),
		CloudConfig("#cloud-config"),
		Butane("variant: flatcar\nversion: 1.0.0"),
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

func TestConfAddUserToGroups(t *testing.T) {
	tests := []struct {
		u *UserData
		e error
	}{
		{
			CloudConfig("#cloud-config"),
			errors.New("missing addUserToGroups implementation for this config type"),
		},
		{
			Ignition(`{ "ignitionVersion": 1 }`),
			errors.New("missing addUserToGroups implementation for this config type"),
		},
		{
			Ignition(`{ "ignition": { "version": "2.2.0" } }`),
			errors.New("missing addUserToGroups implementation for this config type"),
		},
		{
			Ignition(`{ "ignition": { "version": "2.1.0" } }`),
			errors.New("missing addUserToGroups implementation for this config type"),
		},
		{
			Ignition(`{ "ignition": { "version": "2.0.0" } }`),
			errors.New("missing addUserToGroups implementation for this config type"),
		},
		{
			Ignition(`{ "ignition": { "version": "3.0.0" } }`),
			nil,
		},
		{
			Ignition(`{ "ignition": { "version": "3.1.0" } }`),
			nil,
		},
		{
			Ignition(`{ "ignition": { "version": "3.2.0" } }`),
			nil,
		},
		{
			Ignition(`{ "ignition": { "version": "3.3.0" } }`),
			nil,
		},
		{
			Butane("variant: flatcar\nversion: 1.0.0"),
			nil,
		},
	}

	for i, tt := range tests {
		conf, err := tt.u.Render("")
		if err != nil {
			t.Errorf("failed to parse config %d: %v", i, err)
			continue
		}

		err = conf.AddUserToGroups("test", []string{"sudo"})
		if tt.e == nil && err != nil {
			t.Errorf("should get nil error, got: %v", err)
		} else if tt.e != nil && err == nil {
			t.Errorf("should get an error, got a nil error")
		}
	}
}
