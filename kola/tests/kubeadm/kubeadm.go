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
	"math"
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
					patch := `{ grep -q svirt_lxc_file_t /etc/selinux/mcs/contexts/lxc_contexts && kubectl --namespace kube-system patch daemonset/cilium -p '{"spec":{"template":{"spec":{"containers":[{"name":"cilium-agent","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}}],"initContainers":[{"name":"mount-cgroup","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}},{"name":"apply-sysctl-overwrites","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}},{"name":"clean-cilium-state","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}}]}}}}'; } || true`
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
		"v1.34.1": map[string]interface{}{
			"HelmVersion":     "v3.17.3",
			"MinMajorVersion": 3374,
			// from https://github.com/flannel-io/flannel/releases
			"FlannelVersion": "v0.26.7",
			// from https://github.com/cilium/cilium/releases
			"CiliumVersion": "1.12.5",
			// from https://github.com/cilium/cilium-cli/releases
			"CiliumCLIVersion": "v0.12.12",
			"DownloadDir":      "/opt/bin",
			"PodSubnet":        "192.168.0.0/17",
			"cgroupv1":         false,
		},
		"v1.33.0": map[string]interface{}{
			"HelmVersion":     "v3.17.3",
			"MinMajorVersion": 3374,
			// from https://github.com/flannel-io/flannel/releases
			"FlannelVersion": "v0.26.7",
			// from https://github.com/cilium/cilium/releases
			"CiliumVersion": "1.12.5",
			// from https://github.com/cilium/cilium-cli/releases
			"CiliumCLIVersion": "v0.12.12",
			"DownloadDir":      "/opt/bin",
			"PodSubnet":        "192.168.0.0/17",
			"cgroupv1":         false,
		},
		"v1.32.4": map[string]interface{}{
			"HelmVersion":     "v3.17.0",
			"MinMajorVersion": 3374,
			// from https://github.com/flannel-io/flannel/releases
			"FlannelVersion": "v0.22.0",
			// from https://github.com/cilium/cilium/releases
			"CiliumVersion": "1.12.5",
			// from https://github.com/cilium/cilium-cli/releases
			"CiliumCLIVersion": "v0.12.12",
			"DownloadDir":      "/opt/bin",
			"PodSubnet":        "192.168.0.0/17",
			"cgroupv1":         false,
		},
	}
	plog       = capnslog.NewPackageLogger("github.com/flatcar/mantle", "kola/tests/kubeadm")
	etcdConfig = conf.ContainerLinuxConfig(`
etcd:
  version: 3.5.22
  advertise_client_urls: http://{PRIVATE_IPV4}:2379
  listen_client_urls: http://0.0.0.0:2379`)
)

func init() {
	testConfigCgroupV1 := map[string]map[string]interface{}{}
	testConfigCgroupV1["v1.32.4"] = map[string]interface{}{}
	for k, v := range testConfig["v1.32.4"] {
		testConfigCgroupV1["v1.32.4"][k] = v
	}
	testConfigCgroupV1["v1.32.4"]["cgroupv1"] = true

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
				var majorMinVersion int64 = 0
				var majorEndVersion int64 = math.MaxInt64
				if testParams["cgroupv1"].(bool) {
					cgroupSuffix = ".cgroupv1"
					majorMinVersion = 3140
					majorEndVersion = 4179
				}

				if CNI == "flannel" {
					flags = append(flags, register.NoEnableSelinux)
				}

				if mmvi, ok := testParams["MinMajorVersion"]; ok {
					mmv := (int64)(mmvi.(int))
					// Careful, so we don't lower
					// the min version too much.
					if mmv > majorMinVersion {
						majorMinVersion = mmv
					}
				}

				register.Register(&register.Test{
					Name:    fmt.Sprintf("kubeadm.%s.%s%s.base", version, CNI, cgroupSuffix),
					Distros: []string{"cl"},
					// This should run on all clouds as a good end-to-end test
					// Network config problems in qemu-unpriv
					// akamai: Tests are failing for Kubernetes because we don't use the expected disk size,
					// so we are running out of free space.
					ExcludePlatforms: []string{"qemu-unpriv", "akamai"},
					Run: func(c cluster.TestCluster) {
						kubeadmBaseTest(c, testParams)
					},
					MinVersion: semver.Version{Major: majorMinVersion},
					EndVersion: semver.Version{Major: majorEndVersion},
					Flags:      flags,
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
			out := c.MustSSH(kubectl, "kubectl get nodes | grep \" Ready\"| wc -l")
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
		if _, err := c.SSH(kubectl, "kubectl apply -f nginx.yaml"); err != nil {
			c.Fatalf("unable to deploy nginx: %v", err)
		}

		// Wait up to 3 min (36*5 = 180s) for nginx. The test can be flaky on overcommitted platforms.
		if err := util.Retry(36, 5*time.Second, func() error {
			out := c.MustSSH(kubectl, "kubectl get deployments -o json | jq '.items | .[] | .status.readyReplicas'")
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
		if _, err := c.SSH(kubectl, "kubectl apply -f nfs-pod.yaml -f nfs-pvc.yaml"); err != nil {
			c.Fatalf("unable to create NFS pod and pvc: %v", err)
		}

		// Wait up to 3 min (36*5 = 180s). The test can be flaky on overcommitted platforms.
		if err := util.Retry(36, 5*time.Second, func() error {
			out, err := c.SSH(kubectl, `kubectl get pod/test-pod-1 -o json | jq '.status.containerStatuses[] | select (.name == "test") | .ready'`)
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
		master, err = tutil.NewMachineWithLargeDisk(c, "5G", conf.Butane(masterCfg.String()))
		if err != nil {
			return nil, fmt.Errorf("unable to create master node with large disk: %w", err)
		}
	} else {
		master, err = c.NewMachine(conf.Butane(masterCfg.String()))
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
		worker, err = tutil.NewMachineWithLargeDisk(c, "5G", conf.Butane(workerCfg.String()))
		if err != nil {
			return nil, fmt.Errorf("unable to create worker node with large disk: %w", err)
		}
	} else {
		worker, err = c.NewMachine(conf.Butane(workerCfg.String()))
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
