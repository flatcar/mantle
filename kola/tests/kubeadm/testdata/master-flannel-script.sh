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



{
    kubeadm config images pull
    kubeadm init --config kubeadm-config.yaml
    mkdir --parent "${HOME}"/.kube /home/core/.kube
    cp /etc/kubernetes/admin.conf "${HOME}"/.kube/config
    cp /etc/kubernetes/admin.conf /home/core/.kube/config
    chown -R core:core /home/core/.kube; chmod a+r /home/core/.kube/config;



    curl -sSfL https://raw.githubusercontent.com/flannel-io/flannel/v0.14.0/Documentation/kube-flannel.yml > kube-flannel.yml
    sed -i "s#10.244.0.0/16#192.168.0.0/17#" kube-flannel.yml
    kubectl apply -f kube-flannel.yml


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
