// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/auth"
	"github.com/flatcar-linux/mantle/cli"
)

var (
	root = &cobra.Command{
		Use:   "gangue",
		Short: "Google Storage download and verification tool",
	}

	jsonKeyFile string
	serviceAuth bool
)

func init() {
	root.PersistentFlags().BoolVar(&serviceAuth, "service-auth", false, "use non-interactive auth when running within GCE")
	root.PersistentFlags().StringVar(&jsonKeyFile, "json-key", "", "use a service account's JSON key for authentication")
}

func validateGSURL(rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsedURL.Scheme != "gs" {
		return fmt.Errorf("URL missing gs:// scheme: %v", rawURL)
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("URL missing bucket name %v", rawURL)
	}
	if parsedURL.Path == "" {
		return fmt.Errorf("URL missing file path %v", rawURL)
	}
	if parsedURL.Path[len(parsedURL.Path)-1] == '/' {
		return fmt.Errorf("URL must not be a directory path %v", rawURL)
	}
	return nil
}

func getGoogleClient() (*http.Client, error) {
	if serviceAuth {
		return auth.GoogleServiceClient(), nil
	} else if jsonKeyFile != "" {
		if b, err := ioutil.ReadFile(jsonKeyFile); err == nil {
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
