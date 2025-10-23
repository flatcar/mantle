#!/bin/bash
set -euo pipefail

# we get the node cgroup driver
# in order to pass the params to the
# kubelet config for both controller and worker
cgroup=$(docker info | awk '/Cgroup Driver/ { print $3}')



# we create the kubeadm config
# plugin-volume-dir and flex-volume-plugin-dir are required since /usr is read-only mounted
# etcd is also defined as external. The provided one has some issues with docker and selinux
# (permission denied with /var/lib/etcd) so it can't boot properly
cat << EOF > kubeadm-config.yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
cgroupDriver: ${cgroup}
---
apiVersion: kubeadm.k8s.io/v1beta4
kind: InitConfiguration
nodeRegistration:
  kubeletExtraArgs:
    - name: volume-plugin-dir
      value: "/opt/libexec/kubernetes/kubelet-plugins/volume/exec/"
timeouts:
  controlPlaneComponentHealthCheck: 30m0s
---
apiVersion: kubeadm.k8s.io/v1beta4
kind: ClusterConfiguration
etcd:
  external:
    endpoints:
      - http://1.2.3.4:2379
networking:
  podSubnet: 192.168.0.0/17
controllerManager:
  extraArgs:
    - name: flex-volume-plugin-dir
      value: "/opt/libexec/kubernetes/kubelet-plugins/volume/exec/"
EOF


cat << EOF > calico.yaml
# Source: https://raw.githubusercontent.com/projectcalico/calico/v3.30.4/manifests/custom-resources.yaml
# This section includes base Calico installation configuration.
# For more information, see: https://docs.tigera.io/calico/latest/reference/installation/api#operator.tigera.io/v1.Installation
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  # Use GH container registry to get rid of Docker limitation.
  registry: ghcr.io/
  imagePath: flatcar/calico
  # Configures Calico networking.
  calicoNetwork:
    ipPools:
    - name: default-ipv4-ippool
      blockSize: 26
      cidr: 192.168.0.0/17
      encapsulation: VXLANCrossSubnet
      natOutgoing: Enabled
      nodeSelector: all()
  flexVolumePath: /opt/libexec/kubernetes/kubelet-plugins/volume/exec/

---

# This section configures the Calico API server.
# For more information, see: https://docs.tigera.io/calico/latest/reference/installation/api#operator.tigera.io/v1.APIServer
apiVersion: operator.tigera.io/v1
kind: APIServer
metadata:
  name: default
spec: {}
---

# Configures the Calico Goldmane flow aggregator.
apiVersion: operator.tigera.io/v1
kind: Goldmane
metadata:
  name: default

---

# Configures the Calico Whisker observability UI.
apiVersion: operator.tigera.io/v1
kind: Whisker
metadata:
  name: default
EOF


{
    kubeadm config images pull
    kubeadm init --config kubeadm-config.yaml
    mkdir --parent "${HOME}"/.kube /home/core/.kube
    cp /etc/kubernetes/admin.conf "${HOME}"/.kube/config
    cp /etc/kubernetes/admin.conf /home/core/.kube/config
    chown -R core:core /home/core/.kube; chmod a+r /home/core/.kube/config;


    kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.30.4/manifests/tigera-operator.yaml
    # calico.yaml uses Installation and APIServer CRDs, so make sure that they are established.
    kubectl -n tigera-operator wait --for create --timeout=60s crd/installations.operator.tigera.io
    kubectl -n tigera-operator wait --for condition=established --timeout=60s crd/installations.operator.tigera.io
    kubectl -n tigera-operator wait --for create --timeout=60s crd/apiservers.operator.tigera.io
    kubectl -n tigera-operator wait --for condition=established --timeout=60s crd/apiservers.operator.tigera.io
    kubectl apply -f calico.yaml



} 1>&2


URL=$(kubectl config view -o jsonpath='{.clusters[0].cluster.server}')
prefix="https://"
short_url=${URL#"${prefix}"}
token=$(kubeadm token create)
certHashes=$(openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | openssl rsa -pubin -outform der 2>/dev/null | openssl dgst -sha256 -hex | sed 's/^.* //')

cat << EOF
apiVersion: kubeadm.k8s.io/v1beta4
kind: JoinConfiguration
nodeRegistration:
  kubeletExtraArgs:
    - name: volume-plugin-dir
      value: "/opt/libexec/kubernetes/kubelet-plugins/volume/exec/"
discovery:
  bootstrapToken:
    token: ${token}
    apiServerEndpoint: ${short_url}
    caCertHashes:
    - sha256:${certHashes}
timeouts:
  controlPlaneComponentHealthCheck: 30m0s
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
cgroupDriver: ${cgroup}
EOF
