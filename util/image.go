// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/json"
	"os/exec"
)

type ImageInfo struct {
	Format      string `json:"format"`
	VirtualSize uint64 `json:"virtual-size"`
}

func GetImageInfo(path string) (*ImageInfo, error) {
	out, err := exec.Command("qemu-img", "info", "--output=json", path).Output()
	if err != nil {
		return nil, err
	}

	var info ImageInfo
	err = json.Unmarshal(out, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}
