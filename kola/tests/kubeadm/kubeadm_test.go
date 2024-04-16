// Copyright 2021 Kinvolk GmbH
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
// local package import and it's behavior as a package

package kubeadm

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTemplate(t *testing.T) {
	t.Run("SuccessWithBase64", func(t *testing.T) {
		res, err := render(
			"Hello, {{ .World }}",
			map[string]interface{}{
				"World": "world !",
			},
			true,
		)
		require.Nil(t, err)
		assert.Equal(t, "SGVsbG8sIHdvcmxkICE=", res.String())

	})
	t.Run("Success", func(t *testing.T) {
		res, err := render(
			"Hello, {{ .World }}",
			map[string]interface{}{
				"World": "world !",
			},
			false,
		)
		require.Nil(t, err)
		assert.Equal(t, "Hello, world !", res.String())

	})
	t.Run("SuccessMasterScript", func(t *testing.T) {
		for _, CNI := range CNIs {
			res, err := render(
				masterScript,
				map[string]interface{}{
					"HelmVersion":    "1.2.3",
					"CiliumVersion":  "v0.11.1",
					"FlannelVersion": "v0.14.0",
					"CNI":            CNI,
					"Endpoints":      []string{"http://1.2.3.4:2379"},
					"Params":         "amd64",
					"DownloadDir":    "/opt/bin",
					"PodSubnet":      "192.168.0.0/17",
				},
				false,
			)
			require.Nil(t, err)
			script, err := ioutil.ReadFile(fmt.Sprintf("testdata/master-%s-script.sh", CNI))
			require.Nil(t, err)
			assert.Equal(t, string(script), res.String())
		}
	})
	t.Run("SuccessMasterConfig", func(t *testing.T) {
		for _, arch := range []string{"amd64", "arm64"} {
			res, err := render(
				masterConfig,
				map[string]interface{}{
					"HelmVersion":      "1.2.3",
					"CiliumVersion":    "v0.11.1",
					"CNI":              "cilium",
					"CiliumCLIVersion": "v0.9.0",
					"Endpoints":        []string{"http://1.2.3.4:2379"},
					"Arch":             arch,
					"DownloadDir":      "/opt/bin",
					"PodSubnet":        "192.168.0.0/17",
					"Release":          "v1.29.2",
				},
				false,
			)
			require.Nil(t, err)
			script, err := ioutil.ReadFile(fmt.Sprintf("testdata/master-cilium-%s-config.yml", arch))
			require.Nil(t, err)
			assert.Equal(t, string(script), res.String())
		}
	})
}
