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
package kubeadm

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/coreos/pkg/capnslog"

	"github.com/flatcar-linux/mantle/kola"
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/kola/tests/etcd"
	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/conf"
	"github.com/flatcar-linux/mantle/util"
)

var (
	// CNIs is the list of CNIs to deploy
	// in the cluster setup
	CNIs = []string{
		"calico",
		"flannel",
		"cilium",
	}
	// testConfig holds params for various kubernetes releases
	// and the nested params are used to render script templates
	testConfig = map[string]map[string]interface{}{
		"v1.22.0": map[string]interface{}{
			"CiliumCLIVersion": "v0.9.0",
			"CNIVersion":       "v0.8.7",
			"CRIctlVersion":    "v1.17.0",
			"ReleaseVersion":   "v0.4.0",
			"DownloadDir":      "/opt/bin",
			"PodSubnet":        "192.168.0.0/17",
			"arm64": map[string]string{
				"KubeadmSum": "bdc32d358eba328a16b515cb1b7b1fd76bb5951ed5c6e1ea845798f13e6415e040e5f150030bd446e6eb6096533136780aef263b2a9c38ba11536c4415212be0",
				"KubeletSum": "39953c3dce3dd579b1601859681ee81825b3bc3fdf764a097b31c01277bc8afc23693599ec4c1065d8844e1fed91f5edea558b33648794f084fc495efe623f88",
				"CRIctlSum":  "45ab5f2dccb6579b5d376c07dd8264dd714a56ead32744655e698f5919bb0e7934a88666cccfad9cedf30d5bb713394f359f5c6a50963da9a34ddb469dbee92a",
				"CNISum":     "d1fcb37c727c6aa328e1f51d2a06c93a43dbdee2b7f495e12725e6d60db664d6068a1e6e26025df6c4996d9431921855c71df60c227e62bacbf5c9d213a21f8d",
				"KubectlSum": "0912bf3f26eeb35cce3adf21dfe899ea6707c499bd50a19dbaad7c64e00a6c3cd33a21c5f67e5022cf6c4187b2994b7b9a1c9dd7d192fb0fbd3ac52fdb776f07",
			},
			"amd64": map[string]string{
				"KubeadmSum": "339e13ad840cbeab906e416f321467ab6c91cc4b66e5ad4db6f8d41a974146cf8226727edbcf686854a0803246e316158f028de7e753197cdcd2d99a604afbfd",
				"KubeletSum": "1b5d530e62f0198aa7af09371ba799d135b54b9a4513981fa09b786ca5fdc98819345112b5c3a68834f6171e9b4438075cf7ec77c2c575b8e3c56b8eb15d2a86",
				"CRIctlSum":  "e258f4607a89b8d44c700036e636dd42cc3e2ed27a3bb13beef736f80f64f10b7974c01259a66131d3f7b44ed0c61b1ca0ea91597c416a9c095c432de5112d44",
				"CNISum":     "8f2cbee3b5f94d59f919054dccfe99a8e3db5473b553d91da8af4763e811138533e05df4dbeab16b3f774852b4184a7994968f5e036a3f531ad1ac4620d10ede",
				"KubectlSum": "a93b2ca067629cb1fe9cbf1af1a195c12126488ed321e3652200d4dbfee9a577865647b7ef6bb673e1bdf08f03108b5dcb4b05812a649a0de5c7c9efc1407810",
			},
		},
		"v1.21.0": map[string]interface{}{
			"CiliumCLIVersion": "v0.9.0",
			"CNIVersion":       "v0.8.7",
			"CRIctlVersion":    "v1.17.0",
			"ReleaseVersion":   "v0.4.0",
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
				"KubeadmSum": "0673408403a3474c868ae86109f11f9114bca7ddce204be0d169316fb3ce0edefa4b2a472ba9b8308e423e6b927d4098ac36296405570f444f39551fb1c4bbb4",
				"KubeletSum": "530689c0cc32ef1830f7ae26ac10995f815043d48a905141e23a34a5e61522c4ee2ff46953648c47c5592d7c2ffa40ce90469a697f36f68475b8da5abd73f9f5",
				"CRIctlSum":  "e258f4607a89b8d44c700036e636dd42cc3e2ed27a3bb13beef736f80f64f10b7974c01259a66131d3f7b44ed0c61b1ca0ea91597c416a9c095c432de5112d44",
				"CNISum":     "8f2cbee3b5f94d59f919054dccfe99a8e3db5473b553d91da8af4763e811138533e05df4dbeab16b3f774852b4184a7994968f5e036a3f531ad1ac4620d10ede",
				"KubectlSum": "9557d298146ef62ffbcf05b3591bf1ce74f345628370447a4f614b5f64e367b5bfa8e397cc4755da9ea38f1ba04c95c65c313e735550ffc3b03c197e936c3e11",
			},
		},
	}
	plog       = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "kola/tests/kubeadm")
	etcdConfig = conf.ContainerLinuxConfig(`
etcd:
  advertise_client_urls: http://{PRIVATE_IPV4}:2379
  listen_client_urls: http://0.0.0.0:2379`)
)

func init() {
	for version, params := range testConfig {
		for _, CNI := range CNIs {
			// ugly but required to remove the reference between params and the params
			// actually used by the test.
			testParams := make(map[string]interface{})
			for k, v := range params {
				testParams[k] = v
			}
			testParams["CNI"] = CNI
			testParams["Release"] = version

			architectures := []string{"amd64"}

			if CNI != "calico" {
				architectures = append(architectures, "arm64")
			}

			register.Register(&register.Test{
				Name:             fmt.Sprintf("kubeadm.%s.%s.base", version, CNI),
				Distros:          []string{"cl"},
				ExcludePlatforms: []string{"esx"},
				Run: func(c cluster.TestCluster) {
					kubeadmBaseTest(c, testParams)
				},
				Architectures: architectures,
			})
		}
	}
}

// kubeadmBaseTest asserts that the cluster is up and running
func kubeadmBaseTest(c cluster.TestCluster, params map[string]interface{}) {
	params["Arch"] = strings.SplitN(kola.QEMUOptions.Board, "-", 2)[0]
	kubectl, err := setup(c, params)
	if err != nil {
		c.Fatalf("unable to setup cluster: %v", err)
	}

	c.Run("node readiness", func(c cluster.TestCluster) {
		// we let some times to the cluster to be fully booted
		if err := util.Retry(10, 10*time.Second, func() error {
			// notice the extra space before "Ready", it's to not catch
			// "NotReady" nodes
			out := c.MustSSH(kubectl, "/opt/bin/kubectl get nodes | grep \" Ready\"| wc -l")
			readyNodesCnt := string(out)
			if readyNodesCnt != "2" {
				return fmt.Errorf("ready nodes should be equal to 2: %s", readyNodesCnt)
			}

			return nil
		}); err != nil {
			c.Fatalf("nodes are not ready: %v", err)
		}
	})
	c.Run("nginx deployment", func(c cluster.TestCluster) {
		// nginx manifest has been deployed through ignition
		if _, err := c.SSH(kubectl, "/opt/bin/kubectl apply -f nginx.yaml"); err != nil {
			c.Fatalf("unable to deploy nginx: %v", err)
		}

		if err := util.Retry(10, 10*time.Second, func() error {
			out := c.MustSSH(kubectl, "/opt/bin/kubectl get deployments -o json | jq '.items | .[] | .status.readyReplicas'")
			readyCnt := string(out)
			if readyCnt != "1" {
				return fmt.Errorf("ready replicas should be equal to 1: %s", readyCnt)
			}

			return nil
		}); err != nil {
			c.Fatalf("nginx is not deployed: %v", err)
		}
	})
}

// render takes care of template rendering
// using `b` parameter, we can render in a base64 encoded format
func render(s string, p map[string]interface{}, b bool) (*bytes.Buffer, error) {
	tmpl, err := template.New("install").Parse(s)
	if err != nil {
		return nil, fmt.Errorf("unable to parse script: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, p); err != nil {
		return nil, fmt.Errorf("unable to execute template: %w", err)
	}

	if b {
		b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
		buf.Reset()
		if _, err := buf.WriteString(b64); err != nil {
			return nil, fmt.Errorf("unable to write bas64 content to buffer: %w", err)
		}
	}

	return &buf, nil
}

// setup creates a cluster with kubeadm
// cluster is composed by etcd node, worker and master node
// it returns master node in order to have direct access on node
// with kubectl installed and setup
func setup(c cluster.TestCluster, params map[string]interface{}) (platform.Machine, error) {
	plog.Infof("creating etcd node")

	etcdNode, err := c.NewMachine(etcdConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create etcd node: %w", err)
	}

	if err := etcd.GetClusterHealth(c, etcdNode, 1); err != nil {
		return nil, fmt.Errorf("unable to get etcd node health: %w", err)
	}

	params["Endpoints"] = []string{fmt.Sprintf("http://%s:2379", etcdNode.PrivateIP())}

	plog.Infof("creating master node")

	mScript, err := render(masterScript, params, true)
	if err != nil {
		return nil, fmt.Errorf("unable to render master script: %w", err)
	}

	params["MasterScript"] = mScript.String()

	masterCfg, err := render(masterConfig, params, false)
	if err != nil {
		return nil, fmt.Errorf("unable to render container linux config for master: %w", err)
	}

	master, err := c.NewMachine(conf.ContainerLinuxConfig(masterCfg.String()))
	if err != nil {
		return nil, fmt.Errorf("unable to create master node: %w", err)
	}

	out, err := c.SSH(master, "sudo /home/core/install.sh")
	if err != nil {
		return nil, fmt.Errorf("unable to run master script: %w", err)
	}

	// "out" holds the worker config generated
	// by the master script install
	params["WorkerConfig"] = string(out)

	plog.Infof("creating worker node")
	wScript, err := render(workerScript, params, true)
	if err != nil {
		return nil, fmt.Errorf("unable to render worker script: %w", err)
	}

	params["WorkerScript"] = wScript.String()

	workerCfg, err := render(workerConfig, params, false)
	if err != nil {
		return nil, fmt.Errorf("unable to render container linux config for master: %w", err)
	}

	worker, err := c.NewMachine(conf.ContainerLinuxConfig(workerCfg.String()))
	if err != nil {
		return nil, fmt.Errorf("unable to create worker node: %w", err)
	}

	out, err = c.SSH(worker, "sudo /home/core/install.sh")
	if err != nil {
		return nil, fmt.Errorf("unable to run worker script: %w", err)
	}

	return master, nil
}
