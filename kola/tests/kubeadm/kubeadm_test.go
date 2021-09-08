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
					"CNI":            CNI,
					"Endpoints":      []string{"http://1.2.3.4:2379"},
					"Params":         "amd64",
					"CNIVersion":     "v0.8.7",
					"CRIctlVersion":  "v1.17.0",
					"ReleaseVersion": "v0.4.0",
					"Release":        "v1.21.0",
					"DownloadDir":    "/opt/bin",
					"PodSubnet":      "192.168.0.0/17",
					"KubeadmSum":     "0673408403a3474c868ae86109f11f9114bca7ddce204be0d169316fb3ce0edefa4b2a472ba9b8308e423e6b927d4098ac36296405570f444f39551fb1c4bbb4",
					"KubeletSum":     "530689c0cc32ef1830f7ae26ac10995f815043d48a905141e23a34a5e61522c4ee2ff46953648c47c5592d7c2ffa40ce90469a697f36f68475b8da5abd73f9f5",
					"CRIctlSum":      "e258f4607a89b8d44c700036e636dd42cc3e2ed27a3bb13beef736f80f64f10b7974c01259a66131d3f7b44ed0c61b1ca0ea91597c416a9c095c432de5112d44",
					"CNISum":         "8f2cbee3b5f94d59f919054dccfe99a8e3db5473b553d91da8af4763e811138533e05df4dbeab16b3f774852b4184a7994968f5e036a3f531ad1ac4620d10ede",
					"KubectlSum":     "9557d298146ef62ffbcf05b3591bf1ce74f345628370447a4f614b5f64e367b5bfa8e397cc4755da9ea38f1ba04c95c65c313e735550ffc3b03c197e936c3e11",
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
					"CNI":              "cilium",
					"CiliumCLIVersion": "v0.9.0",
					"Endpoints":        []string{"http://1.2.3.4:2379"},
					"Arch":             arch,
					"CNIVersion":       "v0.8.7",
					"CRIctlVersion":    "v1.17.0",
					"ReleaseVersion":   "v0.4.0",
					"Release":          "v1.21.0",
					"DownloadDir":      "/opt/bin",
					"PodSubnet":        "192.168.0.0/17",
					"arm64": map[string]string{
						"KubeadmSum": "96248c47e809f88675d932bd8479cc1c170abb958be204965812235fb0173e788a91c46760a274a43cc56af3de4133f8ea1f5daf4f431410dbba043836e775d5",
						"KubeletSum": "fc2a7e3ae6d44c0e384067f8e0bcd47b0db120d03d06cc8589c601f618792959ea894cf3325df8ab4902af23ded7fd875cf4fe718be0e67ad990a7559e4a8b1a",
						"CRIctlSum":  "45ab5f2dccb6579b5d376c07dd8264dd714a56ead32744655e698f5919bb0e7934a88666cccfad9cedf30d5bb713394f359f5c6a50963da9a34ddb469dbee92a",
						"CNISum":     "d1fcb37c727c6aa328e1f51d2a06c93a43dbdee2b7f495e12725e6d60db664d6068a1e6e26025df6c4996d9431921855c71df60c227e62bacbf5c9d213a21f8d",
						"KubectlSum": "b990b81d5a885a9d131aabcc3a5ca9c37dfaff701470f2beb896682a8643c7e0c833e479a26f21129b598ac981732bf52eecdbe73896fe0ff2d9c1ffd082d1fd",
					},
					"amd64": map[string]string{
						"KubeadmSum": "339e13ad840cbeab906e416f321467ab6c91cc4b66e5ad4db6f8d41a974146cf8226727edbcf686854a0803246e316158f028de7e753197cdcd2d99a604afbfd",
						"KubeletSum": "1b5d530e62f0198aa7af09371ba799d135b54b9a4513981fa09b786ca5fdc98819345112b5c3a68834f6171e9b4438075cf7ec77c2c575b8e3c56b8eb15d2a86",
						"CRIctlSum":  "e258f4607a89b8d44c700036e636dd42cc3e2ed27a3bb13beef736f80f64f10b7974c01259a66131d3f7b44ed0c61b1ca0ea91597c416a9c095c432de5112d44",
						"CNISum":     "8f2cbee3b5f94d59f919054dccfe99a8e3db5473b553d91da8af4763e811138533e05df4dbeab16b3f774852b4184a7994968f5e036a3f531ad1ac4620d10ede",
						"KubectlSum": "a93b2ca067629cb1fe9cbf1af1a195c12126488ed321e3652200d4dbfee9a577865647b7ef6bb673e1bdf08f03108b5dcb4b05812a649a0de5c7c9efc1407810",
					},
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
