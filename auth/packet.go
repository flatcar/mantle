// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

const PacketConfigPath = ".config/packet.json"

// PacketProfile represents a parsed Packet profile. This is a custom format
// specific to Mantle.
type PacketProfile struct {
	ApiKey  string `json:"api_key"`
	Project string `json:"project"`
}

// ReadPacketConfig decodes a Packet config file, which is a custom format
// used by Mantle to hold API keys.
//
// If path is empty, $HOME/.config/packet.json is read.
func ReadPacketConfig(path string) (map[string]PacketProfile, error) {
	if path == "" {
		user, err := user.Current()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(user.HomeDir, PacketConfigPath)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var profiles map[string]PacketProfile
	if err := json.NewDecoder(f).Decode(&profiles); err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, fmt.Errorf("Packet config %q contains no profiles", path)
	}

	return profiles, nil
}
