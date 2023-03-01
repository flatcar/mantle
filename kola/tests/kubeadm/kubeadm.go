// Copyright 2021 Kinvolk GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/kola/tests/etcd"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/util"
)

// extraTest is a regular test except that the `runFunc` takes
// a kubernetes controller as parameter in order to run the test commands from the
// controller node.
type extraTest struct {
	// name is the name of the test.
	name string
	// runFunc is step to run in order to perform the actual test. Controller is the Kubernetes node
	// from where the commands are ran.
	runFunc func(m platform.Machine, p map[string]interface{}, c cluster.TestCluster)
}

var (
	// extraTests can be used to extend the common tests for a given supported CNI.
	extraTests = map[string][]extraTest{
		"cilium": []extraTest{
			extraTest{
				name: "IPSec encryption",
				runFunc: func(controller platform.Machine, params map[string]interface{}, c cluster.TestCluster) {
					_ = c.MustSSH(controller, "/opt/bin/cilium uninstall")
					version := params["CiliumVersion"].(string)
					cidr := params["PodSubnet"].(string)
					cmd := fmt.Sprintf("/opt/bin/cilium install --config enable-endpoint-routes=true --config cluster-pool-ipv4-cidr=%s --version=%s --encryption=ipsec --wait=false --restart-unmanaged-pods=false --rollback=false", cidr, version)
					_, _ = c.SSH(controller, cmd)
					patch := `/opt/bin/kubectl --namespace kube-system patch daemonset/cilium -p '{"spec":{"template":{"spec":{"containers":[{"name":"cilium-agent","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}}],"initContainers":[{"name":"mount-cgroup","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}},{"name":"apply-sysctl-overwrites","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}},{"name":"clean-cilium-state","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}}]}}}}'`
					_ = c.MustSSH(controller, patch)
					status := "/opt/bin/cilium status --wait --wait-duration 1m"
					_ = c.MustSSH(controller, status)
				},
			},
		},
	}

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
		"v1.26.0": map[string]interface{}{
			"MinMajorVersion": 3374,
			// from https://github.com/flannel-io/flannel/releases
			"FlannelVersion": "v0.20.2",
			// from https://github.com/cilium/cilium/releases
			"CiliumVersion": "1.12.5",
			// from https://github.com/cilium/cilium-cli/releases
			"CiliumCLIVersion": "v0.12.12",
			// from https://github.com/containernetworking/plugins/releases
			"CNIVersion": "v1.1.1",
			// from https://github.com/kubernetes-sigs/cri-tools/releases
			"CRIctlVersion": "v1.26.0",
			// from https://github.com/kubernetes/release/releases
			"ReleaseVersion": "v0.14.0",
			"DownloadDir":    "/opt/bin",
			"PodSubnet":      "192.168.0.0/17",
			"arm64": map[string]string{
				"KubeadmSum": "b4a4d206b3140ba907397e7ea9177262c6f6d06ec86855579d26f207d16f43ed603a4132076168fd4e2559d6abb60c50964e3884d69236b201d477cd180e56af",
				"KubeletSum": "b9a75c086d41c4cf2c8a6f9875f48a3e3da5a72863ffec28300448dbd7b2a1d42f128180494825eb506efac514df4f616f0052bd81ef70675b50734dc70d7949",
				"CRIctlSum":  "4c7e4541123cbd6f1d6fec1f827395cd58d65716c0998de790f965485738b6d6257c0dc46fd7f66403166c299f6d5bf9ff30b6e1ff9afbb071f17005e834518c",
				"CNISum":     "6b5df61a53601926e4b5a9174828123d555f592165439f541bc117c68781f41c8bd30dccd52367e406d104df849bcbcfb72d9c4bafda4b045c59ce95d0ca0742",
				"KubectlSum": "ced35f756bfbfb4edd15ee839c555b600032ebcf3caeb6fd639223de5a546103f99fead5c3bb6ed4a4b1ee3884980d6b73fa0cb21441767531bb30f11e0ea60e",
			},
			"amd64": map[string]string{
				"KubeadmSum": "934bf6176dd74e74ebc6056d3b65e741847923ca91f7b58d7f00df565c5034e3764c8b785cb39a7e0a9c779a4fe40ab5b0d123b23b2e531f34e99daf9abb3bc8",
				"KubeletSum": "b147e0f072577e3b13f8fea51bc69fe4149cb63056d9f89aee2ef3a74ddbdc3261d9ad91236ff2549b2e8b8a816ef409a183fe27dd9ac8b8b2242ff2922cb2a4",
				"CRIctlSum":  "a3a2c02a90b008686c20babaf272e703924db2a3e2a0d4e2a7c81d994cbc68c47458a4a354ecc243af095b390815c7f203348b9749351ae817bd52a522300449",
				"CNISum":     "4d0ed0abb5951b9cf83cba938ef84bdc5b681f4ac869da8143974f6a53a3ff30c666389fa462b9d14d30af09bf03f6cdf77598c572f8fb3ea00cecdda467a48d",
				"KubectlSum": "0be35f107a13bef00822586fa9ad7154d4149c2168f2835c3b9cb7218156bff549c83b5af6052836fc0b34c896f895943952926253cb4253405f7e528b835977",
			},
			"cgroupv1": false,
		},
		"v1.25.0": map[string]interface{}{
			"MinMajorVersion": 3033,
			// from https://github.com/flannel-io/flannel/releases
			"FlannelVersion": "v0.19.1",
			// from https://github.com/cilium/cilium/releases
			"CiliumVersion": "1.12.1",
			// from https://github.com/cilium/cilium-cli/releases
			"CiliumCLIVersion": "v0.12.2",
			// from https://github.com/containernetworking/plugins/releases
			"CNIVersion": "v1.1.1",
			// from https://github.com/kubernetes-sigs/cri-tools/releases
			"CRIctlVersion": "v1.24.2",
			// from https://github.com/kubernetes/release/releases
			"ReleaseVersion": "v0.14.0",
			"DownloadDir":    "/opt/bin",
			"PodSubnet":      "192.168.0.0/17",
			"arm64": map[string]string{
				"KubeadmSum": "2d59890912a7286c3597f50a88a3d19d47fc989847dd3832730ca1f4b782fb0deb7f7b5704042ab5836c3e87e2f466d14a0967afc3de06da697b0199f8fe60ad",
				"KubeletSum": "fb2bbfb0240cb40b75bf1279a176075725dc97a23f1fc399efb3da5cdf25edf1673957c132a5dfa00a6987bf93fde579e11098261a1df7a56c59f1e089669cc3",
				"CRIctlSum":  "ebd055e9b2888624d006decd582db742131ed815d059d529ba21eaf864becca98a84b20a10eec91051b9d837c6855d28d5042bf5e9a454f4540aec6b82d37e96",
				"CNISum":     "6b5df61a53601926e4b5a9174828123d555f592165439f541bc117c68781f41c8bd30dccd52367e406d104df849bcbcfb72d9c4bafda4b045c59ce95d0ca0742",
				"KubectlSum": "3b212169122b29afafa94c75bb066cb3205196a6fb184d97e41afbc112208dad1cf8e924357e3e4f6c02eba2fcc74e46089396091eba96f3fadad4c47121501e",
			},
			"amd64": map[string]string{
				"KubeadmSum": "dc0715bb8b33efc56cd531b44c423d24eb27d0d4a2e69041ec29267fa88480a62974bdfa40eeb07082e1d3663b5dd3e21cfe04522e4950ac72474710ed734458",
				"KubeletSum": "73f735394f1651bf95500627ccb264725a0c89fa4394103f272665aaecfefab89ecb1dfd91833bfd25c9e32d37640a0926cb030fb0dfa583555a8e4006602b8d",
				"CRIctlSum":  "961188117863ca9af5b084e84691e372efee93ad09daf6a0422e8d75a5803f394d8968064f7ca89f14e8973766201e731241f32538cf2c8d91f0233e786302df",
				"CNISum":     "4d0ed0abb5951b9cf83cba938ef84bdc5b681f4ac869da8143974f6a53a3ff30c666389fa462b9d14d30af09bf03f6cdf77598c572f8fb3ea00cecdda467a48d",
				"KubectlSum": "fac91d79079672954b9ae9f80b9845fbf373e1c4d3663a84cc1538f89bf70cb85faee1bcd01b6263449f4a2995e7117e1c85ed8e5f137732650e8635b4ecee09",
			},
			"cgroupv1": false,
		},
		"v1.24.1": map[string]interface{}{
			"MinMajorVersion":  3033,
			"FlannelVersion":   "v0.18.1",
			"CiliumVersion":    "1.12.1",
			"CiliumCLIVersion": "v0.12.2",
			"CNIVersion":       "v1.1.1",
			"CRIctlVersion":    "v1.24.2",
			"ReleaseVersion":   "v0.13.0",
			"DownloadDir":      "/opt/bin",
			"PodSubnet":        "192.168.0.0/17",
			"arm64": map[string]string{
				"KubeadmSum": "171ad33a0ffed8ae0bc78a48b12bd1575a03e221de4ca079e60d46689373c298a95bc95a156ea235f5e4f6c4fe714740277ee57e8f0f267b2eeb77f569039ad9",
				"KubeletSum": "f774044d65ebcf07143fb482fbe69c8836287d862c740c9e4804c92060d17867f77625fef0c1a0d2a358745520f5e67bd41b7fcf68756ab105ced9e26c84c881",
				"CRIctlSum":  "ebd055e9b2888624d006decd582db742131ed815d059d529ba21eaf864becca98a84b20a10eec91051b9d837c6855d28d5042bf5e9a454f4540aec6b82d37e96",
				"CNISum":     "6b5df61a53601926e4b5a9174828123d555f592165439f541bc117c68781f41c8bd30dccd52367e406d104df849bcbcfb72d9c4bafda4b045c59ce95d0ca0742",
				"KubectlSum": "ae4e316e1127b7189cdd08980729dea0e20946431c8caec07f79ea43dc34e4f161bb687c5cdf306fb032e6a3537597b9d31cfa416ad0bfc85abd0c0f8d11c66d",
			},
			"amd64": map[string]string{
				"KubeadmSum": "4a825ba96997bca7fc1b3a2a4867026632cf3298709685270333452b5d755176c4891c1cfdd589e162d8af0b43aaf956c71455e4cf886ff0d767196eadb9766e",
				"KubeletSum": "553695adcd0229f680f9edf6afcbbeefc051c77fba6c8ff82644852877c15d422801b5453a09e2fb7ddb4894c713dfe4755562711c302800f985a457a0cbb7c3",
				"CRIctlSum":  "961188117863ca9af5b084e84691e372efee93ad09daf6a0422e8d75a5803f394d8968064f7ca89f14e8973766201e731241f32538cf2c8d91f0233e786302df",
				"CNISum":     "4d0ed0abb5951b9cf83cba938ef84bdc5b681f4ac869da8143974f6a53a3ff30c666389fa462b9d14d30af09bf03f6cdf77598c572f8fb3ea00cecdda467a48d",
				"KubectlSum": "db7e24076f2cbc5bae9033c736048a87c820757af3473cbe583ef7831ad046a9ceeb9e40325e381d193f37ac28f4d6926c8e2fb36ff6029f661b090d8aa15470",
			},
			"cgroupv1": false,
		},
	}
	plog       = capnslog.NewPackageLogger("github.com/flatcar/mantle", "kola/tests/kubeadm")
	etcdConfig = conf.ContainerLinuxConfig(`
etcd:
  advertise_client_urls: http://{PRIVATE_IPV4}:2379
  listen_client_urls: http://0.0.0.0:2379`)
)

func init() {
	testConfigCgroupV1 := map[string]map[string]interface{}{}
	testConfigCgroupV1["v1.24.1"] = map[string]interface{}{}
	for k, v := range testConfig["v1.24.1"] {
		testConfigCgroupV1["v1.24.1"][k] = v
	}
	testConfigCgroupV1["v1.24.1"]["cgroupv1"] = true

	registerTests := func(config map[string]map[string]interface{}) {
		for version, params := range config {
			for _, CNI := range CNIs {
				flags := []register.Flag{}
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

				if CNI == "flannel" || CNI == "cilium" {
					flags = append(flags, register.NoEnableSelinux)
				}

				if mmvi, ok := testParams["MinMajorVersion"]; ok {
					mmv := (int64)(mmvi.(int))
					// Careful, so we don't lower
					// the min version too much.
					if mmv > major {
						major = mmv
					}
				}

				register.Register(&register.Test{
					Name:    fmt.Sprintf("kubeadm.%s.%s%s.base", version, CNI, cgroupSuffix),
					Distros: []string{"cl"},
					// This should run on all clouds as a good end-to-end test
					// Network config problems in qemu-unpriv
					ExcludePlatforms: []string{"qemu-unpriv"},
					Run: func(c cluster.TestCluster) {
						kubeadmBaseTest(c, testParams)
					},
					MinVersion: semver.Version{Major: major},
					Flags:      flags,
					SkipFunc: func(version semver.Version, channel, arch, platform string) bool {
						// LTS (3033) does not have the network-kargs service pulled in:
						// https://github.com/flatcar/coreos-overlay/pull/1848/commits/9e04bc12c3c7eb38da05173dc0ff7beaefa13446
						// Let's skip this test for < 3034 on ESX.
						return version.LessThan(semver.Version{Major: 3034}) && platform == "esx"
					},
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

	// this should not fail, we always have the CNI present at this step.
	cni, ok := params["CNI"]
	if !ok {
		c.Fatalf("CNI is not available in the runtime params")
	}

	// based on the CNI, we fetch the list of extra tests to run.
	extras, ok := extraTests[cni.(string)]
	if ok {
		for _, extra := range extras {
			t := extra.runFunc
			c.Run(extra.name, func(c cluster.TestCluster) { t(kubectl, params, c) })
		}
	}
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
