// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"os"
	"path"

	"github.com/flatcar-linux/mantle/platform/conf"
)

// MakeConfigDrive creates a config drive directory tree under outputDir
// and returns the path to the top level directory.
func MakeConfigDrive(userdata *conf.Conf, outputDir string) (string, error) {
	drivePath := path.Join(outputDir, "config-2")
	userPath := path.Join(drivePath, "openstack/latest/user_data")

	if err := os.MkdirAll(path.Dir(userPath), 0777); err != nil {
		os.RemoveAll(drivePath)
		return "", err
	}

	if err := userdata.WriteFile(userPath); err != nil {
		os.RemoveAll(drivePath)
		return "", err
	}

	return drivePath, nil
}
