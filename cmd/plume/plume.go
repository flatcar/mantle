// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"io/ioutil"
	"net/http"

	"github.com/coreos/pkg/capnslog"
	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/auth"
	"github.com/flatcar-linux/mantle/cli"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "plume")
	root = &cobra.Command{
		Use:   "plume [command]",
		Short: "The Flatcar release utility",
	}

	gceJSONKeyFile string
)

func init() {
	root.PersistentFlags().StringVar(&gceJSONKeyFile, "gce-json-key", "", "use a JSON key for authentication (set to 'none' for unauthorized access)")
}

func getGoogleClient() (*http.Client, error) {
	if gceJSONKeyFile == "none" {
		return &http.Client{}, nil
	}

	if gceJSONKeyFile != "" {
		if b, err := ioutil.ReadFile(gceJSONKeyFile); err == nil {
			return auth.GoogleClientFromJSONKey(b)
		} else {
			return nil, err
		}
	}
	return auth.GoogleClient()
}

func main() {
	cli.Execute(root)
}
