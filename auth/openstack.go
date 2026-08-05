// Copyright 2018 Red Hat
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

	"gopkg.in/yaml.v3"
)

const OpenStackConfigPath = ".config/openstack.json"

type OpenStackProfile struct {
	AuthURL    string `json:"auth_url"`
	DomainID   string `json:"domain_id"`
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	Username   string `json:"username"`
	Password   string `json:"password"`

	//Optional
	Domain         string `json:"user_domain"`
	FloatingIPPool string `json:"floating_ip_pool"`
	Region         string `json:"region_name"`
}

// ReadOpenStackConfig decodes an OpenStack config file,
// which is a custom format used by Mantle to hold OpenStack
// server information.
//
// If path is empty, $HOME/.config/openstack.json is read.
func ReadOpenStackConfig(path string) (map[string]OpenStackProfile, error) {
	if path == "" {
		user, err := user.Current()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(user.HomeDir, OpenStackConfigPath)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var profiles map[string]OpenStackProfile
	if err := json.NewDecoder(f).Decode(&profiles); err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, fmt.Errorf("OpenStack config %q contains no profiles", path)
	}

	return profiles, nil
}

// FindCloudsYAML resolves the standard locations to find clouds.yaml
func FindCloudsYAML(customPath string) string {
	if customPath != "" {
		return customPath
	}
	if envPath := os.Getenv("OS_CLIENT_CONFIG_FILE"); envPath != "" {
		return envPath
	}
	paths := []string{
		"clouds.yaml",
		"clouds.yml",
	}
	home, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths,
			filepath.Join(home, ".config/openstack/clouds.yaml"),
			filepath.Join(home, ".config/openstack/clouds.yml"),
		)
	}
	paths = append(paths,
		"/etc/openstack/clouds.yaml",
		"/etc/openstack/clouds.yml",
	)
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// ReadFloatingIPPoolFromClouds parses a clouds.yaml file and retrieves the floating_ip_pool
// field for the specified cloud.
func ReadFloatingIPPoolFromClouds(path, cloudName string) (string, error) {
	if path == "" {
		return "", nil
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var config struct {
		Clouds map[string]struct {
			FloatingIPPool string `yaml:"floating_ip_pool"`
		} `yaml:"clouds"`
	}
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return "", err
	}
	if cloud, ok := config.Clouds[cloudName]; ok {
		return cloud.FloatingIPPool, nil
	}
	return "", nil
}

