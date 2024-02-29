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
	tutil "github.com/flatcar/mantle/kola/tests/util"
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
		"v1.29.2": map[string]interface{}{
			"HelmVersion":     "v3.13.2",
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
				"KubeadmSum": "3e6beeb7794aa002604f0be43af0255e707846760508ebe98006ec72ae8d7a7cf2c14fd52bbcc5084f0e9366b992dc64341b1da646f1ce6e937fb762f880dc15",
				"KubeletSum": "ded47d757fac0279b1b784756fb54b3a5cb0180ce45833838b00d6d7c87578a985e4627503dd7ff734e5f577cf4752ae7daaa2b68e5934fd4617ea15e995f91b",

				"CRIctlSum": "db062e43351a63347871e7094115be2ae3853afcd346d47f7b51141da8c3202c2df58d2e17359322f632abcb37474fd7fdb3b7aadbc5cfd5cf6d3bad040b6251",

				"CNISum":     "b2b7fb74f1b3cb8928f49e5bf9d4bc686e057e837fac3caf1b366d54757921dba80d70cc010399b274d136e8dee9a25b1ad87cdfdc4ffcf42cf88f3e8f99587a",
				"KubectlSum": "b303598f3a65bbc366a7bfb4632d3b5cdd2d41b8a7973de80a99f8b1bb058299b57dc39b17a53eb7a54f1a0479ae4e2093fec675f1baff4613e14e0ed9d65c21",
			},
			"amd64": map[string]string{
				"KubeadmSum": "4261cb0319688a0557b3052cce8df9d754abc38d5fc8e0eeeb63a85a2194895fdca5bad464f8516459ed7b1764d7bbb2304f5f434d42bb35f38764b4b00ce663",
				"KubeletSum": "d3fef1d4b99415179ecb94d4de953bddb74c0fb0f798265829b899bb031e2ab8c2b60037b79a66405a9b102d3db0d90e9257595f4b11660356de0e2e63744cd7",
				"CRIctlSum":  "aa622325bf05520939f9e020d7a28ab48ac23e2fae6f47d5a4e52174c88c1ebc31b464853e4fd65bd8f5331f330a6ca96fd370d247d3eeaed042da4ee2d1219a",
				"CNISum":     "5d0324ca8a3c90c680b6e1fddb245a2255582fa15949ba1f3c6bb7323df9d3af754dae98d6e40ac9ccafb2999c932df2c4288d418949a4915d928eb23c090540",
				"KubectlSum": "a2de71807eb4c41f4d70e5c47fac72ecf3c74984be6c08be0597fc58621baeeddc1b5cc6431ab007eee9bd0a98f8628dd21512b06daaeccfac5837e9792a98a7",
			},
			"cgroupv1": false,
		},
		"v1.28.1": map[string]interface{}{
			"HelmVersion":     "v3.13.2",
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
			"HelmVersion":     "v3.13.2",
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
	}
	plog       = capnslog.NewPackageLogger("github.com/flatcar/mantle", "kola/tests/kubeadm")
	etcdConfig = conf.ContainerLinuxConfig(`
etcd:
  advertise_client_urls: http://{PRIVATE_IPV4}:2379
  listen_client_urls: http://0.0.0.0:2379`)
)

func init() {
	testConfigCgroupV1 := map[string]map[string]interface{}{}
	testConfigCgroupV1["v1.27.2"] = map[string]interface{}{}
	for k, v := range testConfig["v1.27.2"] {
		testConfigCgroupV1["v1.27.2"][k] = v
	}
	testConfigCgroupV1["v1.27.2"]["cgroupv1"] = true

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

				cni := CNI

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
						// * LTS (3033) does not have the network-kargs service pulled in:
						// https://github.com/flatcar/coreos-overlay/pull/1848/commits/9e04bc12c3c7eb38da05173dc0ff7beaefa13446
						// Let's skip this test for < 3034 on ESX
						// * For Cilium Calico/CNI on Brightbox:
						// unprocessable_entity: User data is too long (maximum is 16384 characters)
						// Should be reenabled once we switch to Butane provisioning because of internal compression.
						return (version.LessThan(semver.Version{Major: 3034}) && platform == "esx") ||
							(platform == "brightbox" && (cni == "cilium" || cni == "calico"))
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
	params["Platform"] = c.Platform()
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

	c.Run("NFS deployment", func(c cluster.TestCluster) {
		if _, err := c.SSH(kubectl, "/opt/bin/helm repo add nfs-ganesha-server-and-external-provisioner https://kubernetes-sigs.github.io/nfs-ganesha-server-and-external-provisioner/"); err != nil {
			c.Fatalf("unable to add helm NFS repo: %v", err)
		}

		if _, err := c.SSH(kubectl, "/opt/bin/helm install nfs-server-provisioner nfs-ganesha-server-and-external-provisioner/nfs-server-provisioner --set 'storageClass.mountOptions={nfsvers=4.2}'"); err != nil {
			c.Fatalf("unable to install NFS Helm Chart: %v", err)
		}

		// Manifests have been deployed through Ignition
		if _, err := c.SSH(kubectl, "/opt/bin/kubectl apply -f nfs-pod.yaml -f nfs-pvc.yaml"); err != nil {
			c.Fatalf("unable to create NFS pod and pvc: %v", err)
		}

		// Wait up to 3 min (36*5 = 180s). The test can be flaky on overcommitted platforms.
		if err := util.Retry(36, 5*time.Second, func() error {
			out, err := c.SSH(kubectl, `/opt/bin/kubectl get pod/test-pod-1 -o json | jq '.status.containerStatuses[] | select (.name == "test") | .ready'`)
			if err != nil {
				return fmt.Errorf("getting container status: %v", err)
			}

			ready := string(out)
			if ready != "true" {
				return fmt.Errorf("'test' pod should be ready, got: %s", ready)
			}

			return nil
		}); err != nil {
			c.Fatalf("nginx pod with NFS is not deployed: %v", err)
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

	var master, worker platform.Machine
	p := c.Platform()
	isQemu := p == "qemu" || p == "qemu-unpriv"
	if isQemu {
		master, err = tutil.NewMachineWithLargeDisk(c, "5G", conf.ContainerLinuxConfig(masterCfg.String()))
		if err != nil {
			return nil, fmt.Errorf("unable to create master node with large disk: %w", err)
		}
	} else {
		master, err = c.NewMachine(conf.ContainerLinuxConfig(masterCfg.String()))
		if err != nil {
			return nil, fmt.Errorf("unable to create master node: %w", err)
		}
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

	if isQemu {
		worker, err = tutil.NewMachineWithLargeDisk(c, "5G", conf.ContainerLinuxConfig(workerCfg.String()))
		if err != nil {
			return nil, fmt.Errorf("unable to create worker node with large disk: %w", err)
		}
	} else {
		worker, err = c.NewMachine(conf.ContainerLinuxConfig(workerCfg.String()))
		if err != nil {
			return nil, fmt.Errorf("unable to create worker node: %w", err)
		}
	}

	out, err = c.SSH(worker, "sudo /home/core/install.sh")
	if err != nil {
		return nil, fmt.Errorf("unable to run worker script: %w", err)
	}

	return master, nil
}
