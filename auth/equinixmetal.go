// Copyright 2017 CoreOS, Inc.
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

package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

const EquinixMetalConfigPath = ".config/equinixmetal.json"

// EquinixMetalProfile represents a parsed EquinixMetal profile. This is a custom format
// specific to Mantle.
type EquinixMetalProfile struct {
	ApiKey  string `json:"api_key"`
	Project string `json:"project"`
}

// ReadEquinixMetalConfig decodes a EquinixMetal config file, which is a custom format
// used by Mantle to hold API keys.
//
// If path is empty, $HOME/.config/equinixmetal.json is read.
func ReadEquinixMetalConfig(path string) (map[string]EquinixMetalProfile, error) {
	if path == "" {
		user, err := user.Current()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(user.HomeDir, EquinixMetalConfigPath)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var profiles map[string]EquinixMetalProfile
	if err := json.NewDecoder(f).Decode(&profiles); err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, fmt.Errorf("EquinixMetal config %q contains no profiles", path)
	}

	return profiles, nil
}
