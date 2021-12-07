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

const ESXConfigPath = ".config/esx.json"

// ESXProfile represents a parsed ESX profile. This is a custom format
// specific to Mantle.
type ESXProfile struct {
	Server                 string `json:"server"`
	User                   string `json:"user"`
	Password               string `json:"password"`
	StaticIPs              int    `json:"static_ips,omitempty"`
	FirstStaticIp          string `json:"first_static_ip,omitempty"`
	FirstStaticIpPrivate   string `json:"first_static_ip_private,omitempty"`
	StaticGatewayIp        string `json:"gateway,omitempty"`
	StaticGatewayIpPrivate string `json:"gateway_private,omitempty"`
	StaticSubnetSize       int    `json:"subnet_size,omitempty"`
}

// ReadESXConfig decodes a ESX config file, which is a custom format
// used by Mantle to hold ESX server information.
//
// If path is empty, $HOME/.config/esx.json is read.
func ReadESXConfig(path string) (map[string]ESXProfile, error) {
	if path == "" {
		user, err := user.Current()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(user.HomeDir, ESXConfigPath)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var profiles map[string]ESXProfile
	if err := json.NewDecoder(f).Decode(&profiles); err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, fmt.Errorf("ESX config %q contains no profiles", path)
	}

	return profiles, nil
}
