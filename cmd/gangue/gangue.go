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
