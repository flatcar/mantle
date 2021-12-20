// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/json"
	"fmt"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/platform"
)

// rpmOstreeDeployment represents some of the data of an rpm-ostree deployment
type rpmOstreeDeployment struct {
	Booted            bool     `json:"booted"`
	Checksum          string   `json:"checksum"`
	Origin            string   `json:"origin"`
	Osname            string   `json:"osname"`
	Packages          []string `json:"packages"`
	RequestedPackages []string `json:"requested-packages"`
	Timestamp         int64    `json:"timestamp"`
	Unlocked          string   `json:"unlocked"`
	Version           string   `json:"version"`
}

// simplifiedRpmOstreeStatus contains deployments from rpm-ostree status
type simplifiedRpmOstreeStatus struct {
	Deployments []rpmOstreeDeployment
}

// GetRpmOstreeStatusJSON returns an unmarshal'ed JSON object that contains
// a limited representation of the output of `rpm-ostree status --json`
func GetRpmOstreeStatusJSON(c cluster.TestCluster, m platform.Machine) (simplifiedRpmOstreeStatus, error) {
	target := simplifiedRpmOstreeStatus{}
	rpmOstreeJSON, err := c.SSH(m, "rpm-ostree status --json")
	if err != nil {
		return target, fmt.Errorf("Could not get rpm-ostree status: %v", err)
	}

	err = json.Unmarshal(rpmOstreeJSON, &target)
	if err != nil {
		return target, fmt.Errorf("Couldn't umarshal the rpm-ostree status JSON data: %v", err)
	}

	return target, nil
}
