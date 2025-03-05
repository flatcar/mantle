package misc

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	testsutil "github.com/flatcar/mantle/kola/tests/util"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/util"
)

const (
	CmdTimeout = time.Second * 300
)

const nvidiaDriverVersionOverride = `
variant: flatcar
version: 1.0.0
storage:
  files:
  - path: /etc/flatcar/nvidia-metadata
    contents:
      inline: |
        {{ .NVIDIA_DRIVER_VERSION_LINE }}
`

const nvidiaOperatorTemplate = `
variant: flatcar
version: 1.0.0

storage:
  files:
  - path: /etc/flatcar/nvidia-metadata
    contents:
      inline: |
        NVIDIA_DRIVER_VERSION=570.86.15
  - path: /opt/extensions/kubernetes-v1.30.4-{{ .ARCH_SUFFIX }}.raw
    contents:
      source: https://github.com/flatcar/sysext-bakery/releases/download/latest/kubernetes-v1.30.4-{{ .ARCH_SUFFIX }}.raw
  - path: /opt/extensions/nvidia_runtime-v1.16.2-{{ .ARCH_SUFFIX }}.raw
    contents:
      source: https://github.com/flatcar/sysext-bakery/releases/download/latest/nvidia_runtime-v1.16.2-{{ .ARCH_SUFFIX }}.raw
  links:
  - path: /etc/extensions/kubernetes.raw
    target: /opt/extensions/kubernetes-v1.30.4-{{ .ARCH_SUFFIX }}.raw
    hard: false
  - path: /etc/extensions/nvidia_runtime.raw
    target: /opt/extensions/nvidia_runtime-v1.16.2-{{ .ARCH_SUFFIX }}.raw
    hard: false
`

var plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "kola/tests/misc")

func init() {
	register.Register(&register.Test{
		Name:          "cl.misc.nvidia",
		Run:           verifyNvidiaInstallation,
		ClusterSize:   0,
		Distros:       []string{"cl"},
		// This test is to test the NVIDIA installation, limited to AZURE for now
		Platforms:     []string{"azure", "aws"},
		Architectures: []string{"amd64", "arm64"},
		Flags:         []register.Flag{register.NoEnableSelinux},
		SkipFunc:      skipOnNonGpu,
	})

	register.Register(&register.Test{
		Name:          "cl.misc.nvidia-operator",
		Run:           verifyNvidiaGpuOperator,
		ClusterSize:   0,
		Distros:       []string{"cl"},
		// This test is to test the NVIDIA installation, limited to AZURE for now
		Platforms:     []string{"azure", "aws"},
		Architectures: []string{"amd64", "arm64"},
		Flags:         []register.Flag{register.NoEnableSelinux, register.NoEmergencyShellCheck},
		SkipFunc:      skipOnNonGpu,
	})
}

func skipOnNonGpu(version semver.Version, channel, arch, platform string) bool {
	// N stands for GPU instance obviously :)
	if platform == "azure" && strings.Contains(kola.AzureOptions.Size, "NC") {
		return false
	}
	if platform == "aws" && (strings.HasPrefix(kola.AWSOptions.InstanceType, "p") || strings.HasPrefix(kola.AWSOptions.InstanceType, "g")) {
		return false
	}
	return true
}

func waitForNvidiaDriver(c *cluster.TestCluster, m *platform.Machine) error {
	nvidiaStatusRetry := func() error {
		out, err := c.SSH(*m, "systemctl status nvidia.service")
		if !bytes.Contains(out, []byte("active (exited)")) {
			return fmt.Errorf("nvidia.service: %q: %v", out, err)
		}
		return nil
	}

	if err := util.Retry(40, 15*time.Second, nvidiaStatusRetry); err != nil {
		return err
	}
	return nil
}

func verifyNvidiaInstallation(c cluster.TestCluster) {
	params := map[string]string{}
	// Earlier driver versions have issue building on arm64 with kernel 6.6
	if kola.QEMUOptions.Board == "arm64-usr" {
		params["NVIDIA_DRIVER_VERSION_LINE"] = "NVIDIA_DRIVER_VERSION=570.86.15"
	} else {
		params["NVIDIA_DRIVER_VERSION_LINE"] = ""
	}
	butane, err := testsutil.ExecTemplate(nvidiaDriverVersionOverride, params)
	if err != nil {
		c.Fatalf("ExecTemplate: %s", err)
	}
	m, err := c.NewMachine(conf.Butane(butane))
	if err != nil {
		c.Fatalf("Cluster.NewMachine: %s", err)
	}
	if err := waitForNvidiaDriver(&c, &m); err != nil {
		c.Fatal(err)
	}
	c.AssertCmdOutputContains(m, "/opt/bin/nvidia-smi", "Tesla")
}

func verifyNvidiaGpuOperator(c cluster.TestCluster) {
	params := map[string]string{}
	// For amd64 the suffix is x86-64, for arm64 it's arm64
	if kola.QEMUOptions.Board == "arm64-usr" {
		params["ARCH_SUFFIX"] = "arm64"
	} else {
		params["ARCH_SUFFIX"] = "x86-64"
	}

	butane, err := testsutil.ExecTemplate(nvidiaOperatorTemplate, params)
	if err != nil {
		c.Fatalf("ExecTemplate: %s", err)
	}

	m, err := c.NewMachine(conf.Butane(butane))
	if err != nil {
		c.Fatalf("Cluster.NewMachine: %s", err)
	}

	if err = waitForNvidiaDriver(&c, &m); err != nil {
		c.Fatal(err)
	}
	_ = c.MustSSH(m, "sudo systemctl cat nvidia.service")
	_ = c.MustSSH(m, "sudo systemd-sysext status")
	c.AssertCmdOutputContains(m, "sudo systemd-sysext status", "nvidia_runtime")
	c.AssertCmdOutputContains(m, "sudo systemd-sysext status", "nvidia-driver")
	_ = c.MustSSH(m, `curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 \
	&& chmod 700 get_helm.sh \
	&& HELM_INSTALL_DIR=/opt/bin PATH=$PATH:/opt/bin ./get_helm.sh`)
	_ = c.MustSSH(m, "sudo kubeadm init --pod-network-cidr=10.244.0.0/16")
	_ = c.MustSSH(m, `mkdir -p $HOME/.kube
	sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
	sudo chown $(id -u):$(id -g) $HOME/.kube/config`)
	_ = c.MustSSH(m, "kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml")
	_ = c.MustSSH(m, "kubectl taint nodes --all node-role.kubernetes.io/control-plane-")
	_ = c.MustSSH(m, "kubectl describe nodes $HOSTNAME")
	err = util.Retry(5, 10*time.Second, func() error {
		out, err := c.SSH(m, "kubectl get nodes")
		if err != nil {
			return err
		}
		if strings.Contains(string(out), "NotReady") {
			return fmt.Errorf("nodes not ready: %s", string(out))
		}
		return nil
	})
	if err != nil {
		c.Fatalf("%v", err)
	}
	_ = c.MustSSH(m, "/opt/bin/helm repo add nvidia https://helm.ngc.nvidia.com/nvidia  && /opt/bin/helm repo update")
	_ = c.MustSSH(m, `/opt/bin/helm install --wait --generate-name \
	-n gpu-operator --create-namespace \
	--version v24.6.1 \
	nvidia/gpu-operator \
	--set driver.enabled=false \
	--set toolkit.enabled=false \
	`)
	_ = c.MustSSH(m, "/opt/bin/helm ls")
	err = util.Retry(10, 10*time.Second, func() error {
		out, err := c.SSH(m, "kubectl get pods --all-namespaces -o json | jq '.items[] | select(.status.phase != \"Running\" and .status.phase != \"Succeeded\") | .metadata.name'")
		if err != nil {
			return err
		}
		lines := strings.Split(string(out), "\n")
		if len(lines) > 0 && lines[0] != "" {
			return fmt.Errorf("pods not ready: %d: %v", len(lines), lines)
		}
		return nil
	})
	_ = c.MustSSH(m, "kubectl get pods --all-namespaces")
	if err != nil {
		c.Fatalf("%v", err)
	}
	_ = c.MustSSH(m, `kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: cuda-vectoradd
spec:
  restartPolicy: OnFailure
  containers:
  - name: cuda-vectoradd
    image: "nvcr.io/nvidia/k8s/cuda-sample:vectoradd-cuda11.7.1-ubuntu20.04"
    resources:
      limits:
        nvidia.com/gpu: 1
EOF`)
	// wait until pod/cuda-vectoradd is done
	err = util.Retry(3, 10*time.Second, func() error {
		out, err := c.SSH(m, "kubectl get pod cuda-vectoradd -o jsonpath='{.status.phase}'")
		if err != nil {
			return err
		}
		if !strings.Contains(string(out), "Succeeded") {
			return fmt.Errorf("pod not ready: %s", string(out))
		}
		return nil
	})
	out := c.MustSSH(m, "kubectl get pods")
	c.Logf("get pods: %s", out)
	out = c.MustSSH(m, "kubectl logs pod/cuda-vectoradd")
	c.Logf("cuda-vectoradd logs: %s", out)
	if err != nil {
		c.Fatalf("%v", err)
	}
}
