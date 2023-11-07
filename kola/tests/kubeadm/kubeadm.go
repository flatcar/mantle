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
					patch := `{ grep -q svirt_lxc_file_t /etc/selinux/mcs/contexts/lxc_contexts && /opt/bin/kubectl --namespace kube-system patch daemonset/cilium -p '{"spec":{"template":{"spec":{"containers":[{"name":"cilium-agent","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}}],"initContainers":[{"name":"mount-cgroup","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}},{"name":"apply-sysctl-overwrites","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}},{"name":"clean-cilium-state","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}}]}}}}'; } || true`
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
		"v1.28.1": map[string]interface{}{
			"MinMajorVersion": 3374,
			// from https://github.com/flannel-io/flannel/releases
			"FlannelVersion": "v0.22.0",
			// from https://github.com/cilium/cilium/releases
			"CiliumVersion": "1.12.5",
			// from https://github.com/cilium/cilium-cli/releases
			"CiliumCLIVersion": "v0.12.12",
			// from https://github.com/containernetworking/plugins/releases
			"CNIVersion": "v1.3.0",
			// from https://github.com/kubernetes-sigs/cri-tools/releases
			"CRIctlVersion": "v1.27.0",
			// from https://github.com/kubernetes/release/releases
			"ReleaseVersion": "v0.15.1",
			"DownloadDir":    "/opt/bin",
			"PodSubnet":      "192.168.0.0/17",
			"arm64": map[string]string{
				"KubeadmSum": "5a08b81f9cc82d3cce21130856ca63b8dafca9149d9775dd25b376eb0f18209aa0e4a47c0a6d7e6fb1316aacd5d59dec770f26c09120c866949d70bc415518b3",
				"KubeletSum": "5a898ef543a6482895101ea58e33602e3c0a7682d322aaf08ac3dc8a5a3c8da8f09600d577024549288f8cebb1a86f9c79927796b69a3d8fe989ca8f12b147d6",
				"CRIctlSum":  "db062e43351a63347871e7094115be2ae3853afcd346d47f7b51141da8c3202c2df58d2e17359322f632abcb37474fd7fdb3b7aadbc5cfd5cf6d3bad040b6251",
				"CNISum":     "b2b7fb74f1b3cb8928f49e5bf9d4bc686e057e837fac3caf1b366d54757921dba80d70cc010399b274d136e8dee9a25b1ad87cdfdc4ffcf42cf88f3e8f99587a",
				"KubectlSum": "6a5c9c02a29126949f096415bb1761a0c0ad44168e2ab3d0409982701da58f96223bec354828ddf958e945ef1ce63c0ad41e77cbcbcce0756163e71b4fbae432",
			},
			"amd64": map[string]string{
				"KubeadmSum": "f4daad200c8378dfdc6cb69af28eaca4215f2b4a2dbdf75f29f9210171cb5683bc873fc000319022e6b3ad61175475d77190734713ba9136644394e8a8faafa1",
				"KubeletSum": "ce6ba764274162d38ac1c44e1fb1f0f835346f3afc5b508bb755b1b7d7170910f5812b0a1941b32e29d950e905bbd08ae761c87befad921db4d44969c8562e75",
				"CRIctlSum":  "aa622325bf05520939f9e020d7a28ab48ac23e2fae6f47d5a4e52174c88c1ebc31b464853e4fd65bd8f5331f330a6ca96fd370d247d3eeaed042da4ee2d1219a",
				"CNISum":     "5d0324ca8a3c90c680b6e1fddb245a2255582fa15949ba1f3c6bb7323df9d3af754dae98d6e40ac9ccafb2999c932df2c4288d418949a4915d928eb23c090540",
				"KubectlSum": "33cf3f6e37bcee4dff7ce14ab933c605d07353d4e31446dd2b52c3f05e0b150b60e531f6069f112d8a76331322a72b593537531e62104cfc7c70cb03d46f76b3",
			},
			"cgroupv1": false,
		},
		"v1.27.2": map[string]interface{}{
			"MinMajorVersion": 3374,
			// from https://github.com/flannel-io/flannel/releases
			"FlannelVersion": "v0.22.0",
			// from https://github.com/cilium/cilium/releases
			"CiliumVersion": "1.12.5",
			// from https://github.com/cilium/cilium-cli/releases
			"CiliumCLIVersion": "v0.12.12",
			// from https://github.com/containernetworking/plugins/releases
			"CNIVersion": "v1.3.0",
			// from https://github.com/kubernetes-sigs/cri-tools/releases
			"CRIctlVersion": "v1.27.0",
			// from https://github.com/kubernetes/release/releases
			"ReleaseVersion": "v0.15.1",
			"DownloadDir":    "/opt/bin",
			"PodSubnet":      "192.168.0.0/17",
			"arm64": map[string]string{
				"KubeadmSum": "45b3100984c979ba0f1c0df8f4211474c2d75ebe916e677dff5fc8e3b3697cf7a953da94e356f39684cc860dff6878b772b7514c55651c2f866d9efeef23f970",
				"KubeletSum": "71857ff499ae135fa478e1827a0ed8865e578a8d2b1e25876e914fd0beba03733801c0654bcd4c0567bafeb16887dafb2dbbe8d1116e6ea28dcd8366c142d348",
				"CRIctlSum":  "db062e43351a63347871e7094115be2ae3853afcd346d47f7b51141da8c3202c2df58d2e17359322f632abcb37474fd7fdb3b7aadbc5cfd5cf6d3bad040b6251",
				"CNISum":     "b2b7fb74f1b3cb8928f49e5bf9d4bc686e057e837fac3caf1b366d54757921dba80d70cc010399b274d136e8dee9a25b1ad87cdfdc4ffcf42cf88f3e8f99587a",
				"KubectlSum": "14be61ec35669a27acf2df0380afb85b9b42311d50ca1165718421c5f605df1119ec9ae314696a674051712e80deeaa65e62d2d62ed4d107fe99d0aaf419dafc",
			},
			"amd64": map[string]string{
				"KubeadmSum": "f40216b7d14046931c58072d10c7122934eac5a23c08821371f8b08ac1779443ad11d3458a4c5dcde7cf80fc600a9fefb14b1942aa46a52330248d497ca88836",
				"KubeletSum": "a283da2224d456958b2cb99b4f6faf4457c4ed89e9e95f37d970c637f6a7f64ff4dd4d2bfce538759b2d2090933bece599a285ef8fd132eb383fece9a3941560",
				"CRIctlSum":  "aa622325bf05520939f9e020d7a28ab48ac23e2fae6f47d5a4e52174c88c1ebc31b464853e4fd65bd8f5331f330a6ca96fd370d247d3eeaed042da4ee2d1219a",
				"CNISum":     "5d0324ca8a3c90c680b6e1fddb245a2255582fa15949ba1f3c6bb7323df9d3af754dae98d6e40ac9ccafb2999c932df2c4288d418949a4915d928eb23c090540",
				"KubectlSum": "857e67001e74840518413593d90c6e64ad3f00d55fa44ad9a8e2ed6135392c908caff7ec19af18cbe10784b8f83afe687a0bc3bacbc9eee984cdeb9c0749cb83",
			},
			"cgroupv1": false,
		},
		"v1.26.5": map[string]interface{}{
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
				"KubeadmSum": "46c9f489062bdb84574703f7339d140d7e42c9c71b367cd860071108a3c1d38fabda2ef69f9c0ff88f7c80e88d38f96ab2248d4c9a6c9c60b0a4c20fd640d0db",
				"KubeletSum": "0e4ee1f23bf768c49d09beb13a6b5fad6efc8e3e685e7c5610188763e3af55923fb46158b5e76973a0f9a055f9b30d525b467c53415f965536adc2f04d9cf18d",
				"CRIctlSum":  "4c7e4541123cbd6f1d6fec1f827395cd58d65716c0998de790f965485738b6d6257c0dc46fd7f66403166c299f6d5bf9ff30b6e1ff9afbb071f17005e834518c",
				"CNISum":     "6b5df61a53601926e4b5a9174828123d555f592165439f541bc117c68781f41c8bd30dccd52367e406d104df849bcbcfb72d9c4bafda4b045c59ce95d0ca0742",
				"KubectlSum": "3672fda0beebbbd636a2088f427463cbad32683ea4fbb1df61650552e63846b6a47db803ccb70c3db0a8f24746a23a5632bdc15a3fb78f4f7d833e7f86763c2a",
			},
			"amd64": map[string]string{
				"KubeadmSum": "1c324cd645a7bf93d19d24c87498d9a17878eb1cc927e2680200ffeab2f85051ddec47d85b79b8e774042dc6726299ad3d7caf52c060701f00deba30dc33f660",
				"KubeletSum": "40daf2a9b9e666c14b10e627da931bd79978628b1f23ef6429c1cb4fcba261f86ccff440c0dbb0070ee760fe55772b4fd279c4582dfbb17fa30bc94b7f00126b",
				"CRIctlSum":  "a3a2c02a90b008686c20babaf272e703924db2a3e2a0d4e2a7c81d994cbc68c47458a4a354ecc243af095b390815c7f203348b9749351ae817bd52a522300449",
				"CNISum":     "4d0ed0abb5951b9cf83cba938ef84bdc5b681f4ac869da8143974f6a53a3ff30c666389fa462b9d14d30af09bf03f6cdf77598c572f8fb3ea00cecdda467a48d",
				"KubectlSum": "97840854134909d75a1a2563628cc4ba632067369ce7fc8a8a1e90a387d32dd7bfd73f4f5b5a82ef842088e7470692951eb7fc869c5f297dd740f855672ee628",
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
	testConfigCgroupV1["v1.26.5"] = map[string]interface{}{}
	for k, v := range testConfig["v1.26.5"] {
		testConfigCgroupV1["v1.26.5"][k] = v
	}
	testConfigCgroupV1["v1.26.5"]["cgroupv1"] = true

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

				if CNI == "flannel" {
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
		// Wait up to 3 min (36*5 = 180s) for nginx. The test can be flaky on overcommitted platforms.
		if err := util.Retry(36, 5*time.Second, func() error {
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

		// Wait up to 3 min (36*5 = 180s) for nginx. The test can be flaky on overcommitted platforms.
		if err := util.Retry(36, 5*time.Second, func() error {
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

	v := string(c.MustSSH(etcdNode, `set -euo pipefail; grep -m 1 "^VERSION=" /usr/lib/os-release | cut -d = -f 2`))
	if v == "" {
		c.Fatalf("Assertion for version string failed")
	}

	version, err := semver.NewVersion(v)
	if err != nil {
		c.Fatalf("unable to create semver version from %s: %v", version, err)
	}

	// For Cilium CNI, we enforce SELinux only for version >= 3745 because the SELinux policies update (container_t/spc_t) is not yet
	// propagated through all the channels.
	// The etcd node will run with enforced SELinux anyway but we want to test SELinux on the worker / master nodes.
	cni, ok := params["CNI"]
	if !ok {
		c.Fatal("unable to get CNI value")
	}

	if cni == "cilium" && version.LessThan(semver.Version{Major: 3745}) {
		r := c.RuntimeConf()
		if r != nil {
			plog.Infof("Setting SELinux to permissive mode")
			r.NoEnableSelinux = true
		}
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
