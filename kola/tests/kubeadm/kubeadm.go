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

	"github.com/coreos/go-semver/semver"
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
		"v1.23.4": map[string]interface{}{
			"FlannelVersion":   "v0.16.3",
			"CiliumVersion":    "1.11.0",
			"CiliumCLIVersion": "v0.10.2",
			"CNIVersion":       "v1.0.1",
			"CRIctlVersion":    "v1.22.0",
			"ReleaseVersion":   "v0.4.0",
			"DownloadDir":      "/opt/bin",
			"PodSubnet":        "192.168.0.0/17",
			"arm64": map[string]string{
				"KubeadmSum": "a1d7d1dc0ee4598c53eedfac7a10ae4bf69613b352b0067f9ec5a8c4f5410b37a475afddf669f93646f830ab963f54046d22b101c385732c6252ba9c9ee78d4f",
				"KubeletSum": "209450f58a2e9de79903723b169197e968ee58dd5b1149e3366aff9042286b4f83692f1b69e792155a9879e656802b64d317fdcbd5e85da4ad6cc2cb4667a5fe",
				"CRIctlSum":  "f926c645e0d5f177c0589b1d052ffef4b4ed9d45b3d5b467473b6075ef767fb43b1f7ba5b525d57f021b6b8dc18d7efd27e03e1ec5b71a20f4e321c32456cdd9",
				"CNISum":     "616c4f493a560ecd1ecc60f758720bb2c3539c4261a63d2094f474380d59d88444637cee7fed124c53193f08de7feb65510fe95579b12306c112ad45a74e1536",
				"KubectlSum": "8e46340013faf76b7881314e1f2375b8cb13668994d09fe5037a65d9521b6fe99ce1011339aaee24ce211dc4eef7902c341ddb3d7b628038f060482e3349a7f7",
			},
			"amd64": map[string]string{
				"KubeadmSum": "f56614d98fe93990664477c5c6cddcd319fcde0e452373da3506618c42ff5113a39848f169e1c4c8347dfc8c3e5f525469bcc6333d5c1bc88e60bcba45d57ea9",
				"KubeletSum": "4306ef42564efc96ca7901a7fabe3231a3c660b83d935e78f8a06913cc9aa06b0777976bbc62de4fa5291b9bc2406970213e5d09390826da87cc05f365459c0e",
				"CRIctlSum":  "9ff93e9c15942c39c85dd4e8182b3e9cd47fcb15b1315b0fdfd0d73442a84111e6cf8bb74b586e34b1f382a71107eb7e7820544a98d2224ca6b6dee3ee576222",
				"CNISum":     "220ee0073e9b3708b8ec6159a6ee511b2fd9b88cbe74d48a9b823542e17acf53acec6215869a1d21826422d655eebdd53795fafcef70205d34bf9d8878b493d8",
				"KubectlSum": "12349ef989f85e99ae88bb1e20ad15aa1c0aea7050372b4ae56e9f89c270a176246c445cf350d1024bc91e3fd5955ed1c6035185d0f4217f4b99628e9c173d50",
			},
			"cgroupv1": false,
		},
		"v1.22.7": map[string]interface{}{
			"FlannelVersion":   "v0.16.3",
			"CiliumVersion":    "1.11.0",
			"CiliumCLIVersion": "v0.10.2",
			"CNIVersion":       "v1.0.1",
			"CRIctlVersion":    "v1.22.0",
			"ReleaseVersion":   "v0.4.0",
			"DownloadDir":      "/opt/bin",
			"PodSubnet":        "192.168.0.0/17",
			"arm64": map[string]string{
				"KubeadmSum": "2289516a4bc33d0aff0f85e0d50db00f9f4d211a9a48eabd491b9dee0b6662c7f339135570e9eaa65f7ce82490703b700e18dc663d94de2fa54a0b9cd944daf8",
				"KubeletSum": "62a91ee9b915cb5cc8270b75c3f3fbfdf3fbed71dc422d1d49cbf9f83a5886f327390facdfb3e1c62cd286f56d438101eb6e8101b6b6611dda56647340f013a3",
				"CRIctlSum":  "f926c645e0d5f177c0589b1d052ffef4b4ed9d45b3d5b467473b6075ef767fb43b1f7ba5b525d57f021b6b8dc18d7efd27e03e1ec5b71a20f4e321c32456cdd9",
				"CNISum":     "616c4f493a560ecd1ecc60f758720bb2c3539c4261a63d2094f474380d59d88444637cee7fed124c53193f08de7feb65510fe95579b12306c112ad45a74e1536",
				"KubectlSum": "1714b683a2da381cc9801f20758fc01b7178e8b0a5c1c1f906b7f1aa59f125e1668c9d64aa3b82ffcccf015ec88d519cdaad461bd269f8e437bab1ab0d1be211",
			},
			"amd64": map[string]string{
				"KubeadmSum": "48b5d66203e2da262b2526a3a0e33527a13443014692d60c27a8513b36bde23cdb438cfbbe8fbe884bd0a04b1eb97e95dae2b648713cdefc8ecef3dcd0ed5ade",
				"KubeletSum": "69c1953ecf40e7c171bc918b99fb0379d25bdcea5b88124c088a875c5d343b94b4064457542bc530de1203ef041808cf3b7e4155f777fdc10a462df65848543e",
				"CRIctlSum":  "9ff93e9c15942c39c85dd4e8182b3e9cd47fcb15b1315b0fdfd0d73442a84111e6cf8bb74b586e34b1f382a71107eb7e7820544a98d2224ca6b6dee3ee576222",
				"CNISum":     "220ee0073e9b3708b8ec6159a6ee511b2fd9b88cbe74d48a9b823542e17acf53acec6215869a1d21826422d655eebdd53795fafcef70205d34bf9d8878b493d8",
				"KubectlSum": "c34d3a8f09993036acbe21a580bb25eb95b27c03d2950844220afb1ebe35e8bc67f2cb7682adbe1e1a7f33f5dd34e5abb2c1d899abe2090b194dfdf7b9c2e509",
			},
			"cgroupv1": false,
		},
		"v1.21.10": map[string]interface{}{
			"FlannelVersion":   "v0.16.3",
			"CiliumVersion":    "1.11.0",
			"CiliumCLIVersion": "v0.10.2",
			"CNIVersion":       "v1.0.1",
			"CRIctlVersion":    "v1.22.0",
			"ReleaseVersion":   "v0.4.0",
			"DownloadDir":      "/opt/bin",
			"PodSubnet":        "192.168.0.0/17",
			"arm64": map[string]string{
				"KubeadmSum": "9acabdb85f4dada74dab909b6b21315c81fdeba6df2c156bbebf4b18b8d427a114a853e694d1b42c01c12ede23a43e04651292d65917f405486d8b0088fb2b46",
				"KubeletSum": "029e81f48efa98cbb35835c208540acd662fa16dabea687aff3d25208d7bfa77500d97ed0a756f926f06b989f22971e91baf124d1df330eab9dee90448192752",
				"CRIctlSum":  "f926c645e0d5f177c0589b1d052ffef4b4ed9d45b3d5b467473b6075ef767fb43b1f7ba5b525d57f021b6b8dc18d7efd27e03e1ec5b71a20f4e321c32456cdd9",
				"CNISum":     "616c4f493a560ecd1ecc60f758720bb2c3539c4261a63d2094f474380d59d88444637cee7fed124c53193f08de7feb65510fe95579b12306c112ad45a74e1536",
				"KubectlSum": "c9cd8ebd472d9d6652a307ae78c0753cd3bfbbdcb1ed137db06e16ad900ae454f1e2654d7c01db737b0e2dff3e07b4e18c911264afc94b584f8795baf4a92735",
			},
			"amd64": map[string]string{
				"KubeadmSum": "1319e29679f1d91faf4c9900662641614fd05101dc38618ce075c6819eb8af5a097f8cf416bdf9e2ee9bb92b6c4e3cf4d25df8de2ed6c50ef48a2da605e610b6",
				"KubeletSum": "ee16a40f5982fe459c919d396f7b407e358394cb27836cba8545a5841b3dcedf02ed4da45207b8e311a0707defe977bdcebbed130c7fefc6dd770ec3afd68718",
				"CRIctlSum":  "9ff93e9c15942c39c85dd4e8182b3e9cd47fcb15b1315b0fdfd0d73442a84111e6cf8bb74b586e34b1f382a71107eb7e7820544a98d2224ca6b6dee3ee576222",
				"CNISum":     "220ee0073e9b3708b8ec6159a6ee511b2fd9b88cbe74d48a9b823542e17acf53acec6215869a1d21826422d655eebdd53795fafcef70205d34bf9d8878b493d8",
				"KubectlSum": "5334a634ea72358f2ec303e82678160e260b8920aeb62c1953a43cb70bc919870568e67144f78c4949e0db1297ee2b5e604ff76a7433637c093919c855d189d1",
			},
			"cgroupv1": false,
		},
	}
	plog       = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "kola/tests/kubeadm")
	etcdConfig = conf.ContainerLinuxConfig(`
etcd:
  advertise_client_urls: http://{PRIVATE_IPV4}:2379
  listen_client_urls: http://0.0.0.0:2379`)
)

func init() {
	testConfigCgroupV1 := map[string]map[string]interface{}{}
	testConfigCgroupV1["v1.21.10"] = map[string]interface{}{}
	for k, v := range testConfig["v1.21.10"] {
		testConfigCgroupV1["v1.21.10"][k] = v
	}
	testConfigCgroupV1["v1.21.10"]["cgroupv1"] = true

	registerTests := func(config map[string]map[string]interface{}) {
		for version, params := range config {
			for _, CNI := range CNIs {
				// ugly but required to remove the reference between params and the params
				// actually used by the test.
				testParams := make(map[string]interface{})
				for k, v := range params {
					testParams[k] = v
				}
				testParams["CNI"] = CNI
				testParams["Release"] = version

				cgroupSuffix := ""
				var major int64 = 0
				if testParams["cgroupv1"].(bool) {
					cgroupSuffix = ".cgroupv1"
					major = 3140
				}

				register.Register(&register.Test{
					Name:             fmt.Sprintf("kubeadm.%s.%s%s.base", version, CNI, cgroupSuffix),
					Distros:          []string{"cl"},
					ExcludePlatforms: []string{"esx"},
					Run: func(c cluster.TestCluster) {
						kubeadmBaseTest(c, testParams)
					},
					MinVersion: semver.Version{Major: major},
				})
			}
		}
	}
	registerTests(testConfig)
	registerTests(testConfigCgroupV1)
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
