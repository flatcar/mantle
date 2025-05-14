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
	CmdTimeout           = time.Second * 300
	NvidiaSysextVersion  = "550-open"                         // NVIDIA drivers sysext version used in the template
	KubernetesVersion    = "v1.32.2"                          // Kubernetes version used in the template
	NvidiaRuntimeVersion = "v1.16.2"                          // NVIDIA runtime version used in the template
	GpuOperatorVersion   = "v24.9.2"                          // GPU operator version used for Helm install
	CudaSampleImageTag   = "vectoradd-cuda11.7.1-ubuntu20.04" // CUDA sample image tag
)

const nvidiaOperatorTemplate = `
variant: flatcar
version: 1.0.0

storage:
  files:
  - path: /opt/extensions/kubernetes-{{ .KubernetesVersion }}-{{ .ARCH_SUFFIX }}.raw
    contents:
      source: https://extensions.flatcar.org/extensions/kubernetes-{{ .KubernetesVersion }}-{{ .ARCH_SUFFIX }}.raw
  - path: /opt/extensions/nvidia-runtime-{{ .NvidiaRuntimeVersion }}-{{ .ARCH_SUFFIX }}.raw
    contents:
      source: https://extensions.flatcar.org/extensions/nvidia-runtime-{{ .NvidiaRuntimeVersion }}-{{ .ARCH_SUFFIX }}.raw
  links:
  - path: /etc/extensions/kubernetes.raw
    target: /opt/extensions/kubernetes-{{ .KubernetesVersion }}-{{ .ARCH_SUFFIX }}.raw
    hard: false
  - path: /etc/extensions/nvidia-runtime.raw
    target: /opt/extensions/nvidia-runtime-{{ .NvidiaRuntimeVersion }}-{{ .ARCH_SUFFIX }}.raw
    hard: false
`

var plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "kola/tests/misc")

func init() {
	register.Register(&register.Test{
		Name:          "cl.misc.nvidia",
		Run:           verifyNvidiaInstallation,
		ClusterSize:   0,
		Distros:       []string{"cl"},
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
		Platforms:     []string{"azure", "aws"},
		Architectures: []string{"amd64", "arm64"},
		Flags:         []register.Flag{register.NoEnableSelinux, register.NoEmergencyShellCheck},
		SkipFunc:      skipOnNonGpu,
	})
}

func skipOnNonGpu(_ semver.Version, _, arch, platform string) bool {
	// N stands for GPU instance obviously :)
	if platform == "azure" && strings.Contains(kola.AzureOptions.Size, "NC") {
		return false
	}
	if platform == "aws" && (strings.HasPrefix(kola.AWSOptions.InstanceType, "p") || strings.HasPrefix(kola.AWSOptions.InstanceType, "g")) {
		return false
	}
	return true
}

func runtimeSkipOnNonGpu(c cluster.TestCluster) {
	if skipOnNonGpu(semver.Version{}, "", kola.QEMUOptions.Board, string(c.Platform())) {
		c.Skip("wrong instance size")
	}
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
	runtimeSkipOnNonGpu(c)
	m, err := c.NewMachine(nil)
	if err != nil {
		c.Fatal(err)
	}
	if err := waitForNvidiaDriver(&c, &m); err != nil {
		c.Fatal(err)
	}
	out := c.MustSSH(m, "/opt/bin/nvidia-smi")
	c.Logf("nvidia-smi: %s", out)
}

func verifyNvidiaGpuOperator(c cluster.TestCluster) {
	runtimeSkipOnNonGpu(c)
	params := map[string]string{
		"KubernetesVersion":    KubernetesVersion,
		"NvidiaRuntimeVersion": NvidiaRuntimeVersion,
	}
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

	m, err := testsutil.NewMachineWithLargeDisk(c, "32G", conf.Butane(butane))
	if err != nil {
		c.Fatalf("Cluster.NewMachine: %s", err)
	}

	if err = waitForNvidiaDriver(&c, &m); err != nil {
		c.Fatal(err)
	}
	_ = c.MustSSH(m, "sudo systemctl cat nvidia.service")
	_ = c.MustSSH(m, "sudo systemd-sysext status")
	c.AssertCmdOutputContains(m, "sudo systemd-sysext status", "nvidia-runtime")
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
	_ = c.MustSSH(m, fmt.Sprintf(`/opt/bin/helm install --wait --generate-name \
	-n gpu-operator --create-namespace \
	--version %s \
	nvidia/gpu-operator \
	--set driver.enabled=false \
	--set toolkit.enabled=false \
	`, GpuOperatorVersion))
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
	_ = c.MustSSH(m, fmt.Sprintf(`kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: cuda-vectoradd
spec:
  restartPolicy: OnFailure
  containers:
  - name: cuda-vectoradd
    image: "nvcr.io/nvidia/k8s/cuda-sample:%s"
    resources:
      limits:
        nvidia.com/gpu: 1
EOF`, CudaSampleImageTag))
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
