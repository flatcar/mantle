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

var (
	workerConfig = `systemd:
  units:
{{ if .cgroupv1 }}
    - name: containerd.service
      dropins:
      - name: 10-use-cgroupfs.conf
        contents: |
          [Service]
          Environment=CONTAINERD_CONFIG=/usr/share/containerd/config-cgroupfs.toml
{{ end }}
    - name: prepare-cni-plugins.service
      enabled: true
      contents: |
        [Unit]
        Description=Unpack CNI plugins to /opt/cni/bin
        ConditionPathExists=!/opt/cni/bin
        [Service]
        Type=oneshot
        RemainAfterExit=true
        Restart=on-failure
        Environment=CNI_VERSION={{ .CNIVersion }}
        ExecStartPre=/usr/bin/mkdir --parents /opt/cni/bin
        ExecStartPre=/usr/bin/tar -v --extract --file "/opt/cni-plugins-linux-{{ .Arch }}-${CNI_VERSION}.tgz" --directory /opt/cni/bin --no-same-owner
        ExecStartPre=/usr/bin/chcon -R /opt/cni -t svirt_lxc_file_t
        ExecStart=/usr/bin/rm "/opt/cni-plugins-linux-{{ .Arch }}-${CNI_VERSION}.tgz"
        [Install]
        WantedBy=multi-user.target
    - name: prepare-critools.service
      enabled: true
      contents: |
        [Unit]
        Description=Unpack CRI tools to /opt/bin
        ConditionPathExists=!/opt/bin/crictl
        [Service]
        Type=oneshot
        RemainAfterExit=true
        Restart=on-failure
        Environment=CRICTL_VERSION={{ .CRIctlVersion }}
        Environment=DOWNLOAD_DIR={{ .DownloadDir}}
        ExecStartPre=/usr/bin/mkdir --parents "${DOWNLOAD_DIR}"
        ExecStartPre=/usr/bin/tar -v --extract --file "/opt/crictl-${CRICTL_VERSION}-linux-{{ .Arch }}.tar.gz" --directory "${DOWNLOAD_DIR}" --no-same-owner
        ExecStart=/usr/bin/rm "/opt/crictl-${CRICTL_VERSION}-linux-{{ .Arch }}.tar.gz"
        [Install]
        WantedBy=multi-user.target
storage:
  files:
{{ if .cgroupv1 }}
    - path: /etc/flatcar-cgroupv1
      mode: 0444
{{ end }}
    - path: /opt/cni-plugins-linux-{{ .Arch }}-{{ .CNIVersion }}.tgz
      filesystem: root
      mode: 0644
      contents:
        remote:
          url: https://github.com/containernetworking/plugins/releases/download/{{ .CNIVersion }}/cni-plugins-linux-{{ .Arch }}-{{ .CNIVersion }}.tgz
          verification:
            hash:
              function: sha512
              sum: {{ index (index . .Arch) "CNISum" }}
    - path: /opt/crictl-{{ .CRIctlVersion }}-linux-{{ .Arch }}.tar.gz
      filesystem: root
      mode: 0644
      contents:
        remote:
          url: https://github.com/kubernetes-sigs/cri-tools/releases/download/{{ .CRIctlVersion }}/crictl-{{ .CRIctlVersion }}-linux-{{ .Arch }}.tar.gz
          verification:
            hash:
              function: sha512
              sum: {{ index (index . .Arch) "CRIctlSum" }}
    - path: {{ .DownloadDir }}/kubeadm
      filesystem: root
      mode: 0755
      contents:
        remote:
          url: https://storage.googleapis.com/kubernetes-release/release/{{ .Release }}/bin/linux/{{ .Arch }}/kubeadm
          verification:
            hash:
              function: sha512
              sum: {{ index (index . .Arch) "KubeadmSum" }}
    - path: {{ .DownloadDir }}/kubelet
      filesystem: root
      mode: 0755
      contents:
        remote:
          url: https://storage.googleapis.com/kubernetes-release/release/{{ .Release }}/bin/linux/{{ .Arch }}/kubelet
          verification:
            hash:
              function: sha512
              sum: {{ index (index . .Arch) "KubeletSum" }}
    - path: /home/core/install.sh
      filesystem: root
      mode: 0755
      contents:
        remote:
          url: "data:text/plain;base64,{{ .WorkerScript }}"
    - path: /etc/docker/daemon.json
      filesystem: root
      mode: 0644
      contents:
        inline: |
          {
              "log-driver": "journald"
          }
`

	masterConfig = `systemd:
  units:{{ if .cgroupv1 }}
    - name: containerd.service
      dropins:
      - name: 10-use-cgroupfs.conf
        contents: |
          [Service]
          Environment=CONTAINERD_CONFIG=/usr/share/containerd/config-cgroupfs.toml{{ end }}
    - name: prepare-cni-plugins.service
      enabled: true
      contents: |
        [Unit]
        Description=Unpack CNI plugins to /opt/cni/bin
        ConditionPathExists=!/opt/cni/bin
        [Service]
        Type=oneshot
        RemainAfterExit=true
        Restart=on-failure
        Environment=CNI_VERSION={{ .CNIVersion }}
        ExecStartPre=/usr/bin/mkdir --parents /opt/cni/bin
        ExecStartPre=/usr/bin/tar -v --extract --file "/opt/cni-plugins-linux-{{ .Arch }}-${CNI_VERSION}.tgz" --directory /opt/cni/bin --no-same-owner
        ExecStartPre=/usr/bin/chcon -R /opt/cni -t svirt_lxc_file_t
        ExecStart=/usr/bin/rm "/opt/cni-plugins-linux-{{ .Arch }}-${CNI_VERSION}.tgz"
        [Install]
        WantedBy=multi-user.target
    - name: prepare-critools.service
      enabled: true
      contents: |
        [Unit]
        Description=Unpack CRI tools to /opt/bin
        ConditionPathExists=!/opt/bin/crictl
        [Service]
        Type=oneshot
        RemainAfterExit=true
        Restart=on-failure
        Environment=CRICTL_VERSION={{ .CRIctlVersion }}
        Environment=DOWNLOAD_DIR={{ .DownloadDir}}
        ExecStartPre=/usr/bin/mkdir --parents "${DOWNLOAD_DIR}"
        ExecStartPre=/usr/bin/tar -v --extract --file "/opt/crictl-${CRICTL_VERSION}-linux-{{ .Arch }}.tar.gz" --directory "${DOWNLOAD_DIR}" --no-same-owner
        ExecStart=/usr/bin/rm "/opt/crictl-${CRICTL_VERSION}-linux-{{ .Arch }}.tar.gz"
        [Install]
        WantedBy=multi-user.target
storage:
  files:{{ if .cgroupv1 }}
    - path: /etc/flatcar-cgroupv1
      mode: 0444{{ end }}
    - path: /opt/cni-plugins-linux-{{ .Arch }}-{{ .CNIVersion }}.tgz
      filesystem: root
      mode: 0644
      contents:
        remote:
          url: https://github.com/containernetworking/plugins/releases/download/{{ .CNIVersion }}/cni-plugins-linux-{{ .Arch }}-{{ .CNIVersion }}.tgz
          verification:
            hash:
              function: sha512
              sum: {{ index (index . .Arch) "CNISum" }}
    - path: /opt/crictl-{{ .CRIctlVersion }}-linux-{{ .Arch }}.tar.gz
      filesystem: root
      mode: 0644
      contents:
        remote:
          url: https://github.com/kubernetes-sigs/cri-tools/releases/download/{{ .CRIctlVersion }}/crictl-{{ .CRIctlVersion }}-linux-{{ .Arch }}.tar.gz
          verification:
            hash:
              function: sha512
              sum: {{ index (index . .Arch) "CRIctlSum" }}
    - path: {{ .DownloadDir }}/kubeadm
      filesystem: root
      mode: 0755
      contents:
        remote:
          url: https://storage.googleapis.com/kubernetes-release/release/{{ .Release }}/bin/linux/{{ .Arch }}/kubeadm
          verification:
            hash:
              function: sha512
              sum: {{ index (index . .Arch) "KubeadmSum" }}
    - path: {{ .DownloadDir }}/kubelet
      filesystem: root
      mode: 0755
      contents:
        remote:
          url: https://storage.googleapis.com/kubernetes-release/release/{{ .Release }}/bin/linux/{{ .Arch }}/kubelet
          verification:
            hash:
              function: sha512
              sum: {{ index (index . .Arch) "KubeletSum" }}
    - path: {{ .DownloadDir }}/kubectl
      filesystem: root
      mode: 0755
      contents:
        remote:
          url: https://storage.googleapis.com/kubernetes-release/release/{{ .Release }}/bin/linux/{{ .Arch }}/kubectl
          verification:
            hash:
              function: sha512
              sum: {{ index (index . .Arch) "KubectlSum" }}
    - path: /etc/docker/daemon.json
      filesystem: root
      mode: 0644
      contents:
        inline: |
          {
              "log-driver": "journald"
          }
{{ if eq .CNI "cilium" }}
    - path: {{ .DownloadDir }}/cilium.tar.gz
      filesystem: root
      mode: 0755
      contents:
        remote:
          url: https://github.com/cilium/cilium-cli/releases/download/{{ .CiliumCLIVersion }}/cilium-linux-{{ .Arch }}.tar.gz
{{ end }}
    - path: /home/core/install.sh
      filesystem: root
      mode: 0755
      contents:
        remote:
          url: "data:text/plain;base64,{{ .MasterScript }}"
    - path: /home/core/nginx.yaml
      filesystem: root
      mode: 0644
      contents:
        inline: |
          apiVersion: apps/v1
          kind: Deployment
          metadata:
            name: nginx-deployment
            labels:
              app: nginx
          spec:
            replicas: 1
            selector:
              matchLabels:
                app: nginx
            template:
              metadata:
                labels:
                  app: nginx
              spec:
                containers:
                - name: nginx
                  image: ghcr.io/flatcar/nginx
                  ports:
                  - containerPort: 80
`

	masterScript = `#!/bin/bash
set -euo pipefail

export RELEASE_VERSION={{ .ReleaseVersion }}
export DOWNLOAD_DIR={{ .DownloadDir }}
export PATH="${PATH}:${DOWNLOAD_DIR}"

# create the required directory
mkdir --parent \
    /etc/systemd/system/kubelet.service.d \
    ${HOME}/.kube \
    /home/core/.kube

# we download and install the various requirements:
# * kubelet service and kubeadm dropin
    
curl --retry-delay 1 \
    --retry 60 \
    --retry-connrefused \
    --retry-max-time 60 \
    --connect-timeout 20 \
    --fail \
    -sSL \
    "https://raw.githubusercontent.com/kubernetes/release/${RELEASE_VERSION}/cmd/kubepkg/templates/latest/deb/kubelet/lib/systemd/system/kubelet.service" |
    sed "s:/usr/bin:${DOWNLOAD_DIR}:g" > /etc/systemd/system/kubelet.service
    
curl --retry-delay 1 \
    --retry 60 \
    --retry-connrefused \
    --retry-max-time 60 \
    --connect-timeout 20 \
    --fail \
    -sSL \
    "https://raw.githubusercontent.com/kubernetes/release/${RELEASE_VERSION}/cmd/kubepkg/templates/latest/deb/kubeadm/10-kubeadm.conf" |
    sed "s:/usr/bin:${DOWNLOAD_DIR}:g" > /etc/systemd/system/kubelet.service.d/10-kubeadm.conf


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
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
nodeRegistration:
  kubeletExtraArgs:
    volume-plugin-dir: "/opt/libexec/kubernetes/kubelet-plugins/volume/exec/"
---
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
apiServer:
  timeoutForControlPlane: 30m0s
networking:
  podSubnet: {{ .PodSubnet }}
controllerManager:
  extraArgs:
    flex-volume-plugin-dir: "/opt/libexec/kubernetes/kubelet-plugins/volume/exec/"
etcd:
  external:
    endpoints:
    {{ range $endpoint := .Endpoints }}
      - {{ $endpoint }}
    {{ end }}
EOF

{{ if eq .CNI "calico" }}
cat << EOF > calico.yaml
# Source: https://raw.githubusercontent.com/projectcalico/calico/v3.25.0/manifests/custom-resources.yaml
# This section includes base Calico installation configuration.
# For more information, see: https://projectcalico.docs.tigera.io/master/reference/installation/api#operator.tigera.io/v1.Installation
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
    # Note: The ipPools section cannot be modified post-install.
    ipPools:
    - blockSize: 26
      cidr: {{ .PodSubnet }}
      encapsulation: VXLANCrossSubnet
      natOutgoing: Enabled
      nodeSelector: all()
  flexVolumePath: /opt/libexec/kubernetes/kubelet-plugins/volume/exec/

---

# This section configures the Calico API server.
# For more information, see: https://projectcalico.docs.tigera.io/master/reference/installation/api#operator.tigera.io/v1.APIServer
apiVersion: operator.tigera.io/v1
kind: APIServer
metadata:
  name: default
spec: {}
EOF
{{ end }}

{
    systemctl enable --quiet --now kubelet
    kubeadm config images pull
    kubeadm init --config kubeadm-config.yaml
    cp /etc/kubernetes/admin.conf $HOME/.kube/config
    cp /etc/kubernetes/admin.conf /home/core/.kube/config
    chown -R core:core /home/core/.kube; chmod a+r /home/core/.kube/config;

{{ if eq .CNI "calico" }}
    kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.25.0/manifests/tigera-operator.yaml
    # calico.yaml uses Installation and APIServer CRDs, so make sure that they are established.
    kubectl -n tigera-operator wait --for condition=established --timeout=60s crd/installations.operator.tigera.io
    kubectl -n tigera-operator wait --for condition=established --timeout=60s crd/apiservers.operator.tigera.io
    kubectl apply -f calico.yaml
{{ end }}
{{ if eq .CNI "flannel" }}
    curl -sSfL https://raw.githubusercontent.com/flannel-io/flannel/{{ .FlannelVersion }}/Documentation/kube-flannel.yml > kube-flannel.yml
    sed -i "s#10.244.0.0/16#{{ .PodSubnet }}#" kube-flannel.yml
    kubectl apply -f kube-flannel.yml
{{ end }}
{{ if eq .CNI "cilium" }}
    # iconv transforms the output to valid ascii so that jenkins TAP parser accepts it
    sudo tar -xf {{ .DownloadDir }}/cilium.tar.gz -C {{ .DownloadDir }}
    /opt/bin/cilium install \
        --config enable-endpoint-routes=true \
        --config cluster-pool-ipv4-cidr={{ .PodSubnet }} \
        --version={{ .CiliumVersion }} 2>&1 | iconv --from-code utf-8 --to-code ascii//TRANSLIT
    kubectl --namespace kube-system patch daemonset/cilium -p '{"spec":{"template":{"spec":{"containers":[{"name":"cilium-agent","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}}],"initContainers":[{"name":"mount-cgroup","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}},{"name":"apply-sysctl-overwrites","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}},{"name":"clean-cilium-state","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}}]}}}}'
    # --wait will wait for status to report success
    /opt/bin/cilium status --wait 2>&1 | iconv --from-code utf-8 --to-code ascii//TRANSLIT
{{ end }}
} 1>&2


URL=$(kubectl config view -o jsonpath='{.clusters[0].cluster.server}')
prefix="https://"
short_url=${URL#"${prefix}"}
token=$(kubeadm token create)
certHashes=$(openssl x509 -pubkey -in /etc/kubernetes/pki/ca.crt | openssl rsa -pubin -outform der 2>/dev/null | openssl dgst -sha256 -hex | sed 's/^.* //')

cat << EOF
apiVersion: kubeadm.k8s.io/v1beta3
kind: JoinConfiguration
discovery:
  bootstrapToken:
    apiServerEndpoint: ${short_url}
    token: ${token}
    caCertHashes:
    - sha256:${certHashes}
controlPlane:
nodeRegistration:
  kubeletExtraArgs:
    volume-plugin-dir: "/opt/libexec/kubernetes/kubelet-plugins/volume/exec/"
---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
cgroupDriver: ${cgroup}
EOF
`

	workerScript = `#!/bin/bash
set -euo pipefail

export RELEASE_VERSION={{ .ReleaseVersion }}
export DOWNLOAD_DIR={{ .DownloadDir }}
export PATH="${PATH}:${DOWNLOAD_DIR}"

# create the required directory
mkdir --parent \
    /opt/cni/bin \
    /etc/systemd/system/kubelet.service.d

# we download and install the various requirements
# * kubelet service and kubeadm dropin

curl --retry-delay 1 \
    --retry 60 \
    --retry-connrefused \
    --retry-max-time 60 \
    --connect-timeout 20 \
    --fail \
    -sSL \
    "https://raw.githubusercontent.com/kubernetes/release/${RELEASE_VERSION}/cmd/kubepkg/templates/latest/deb/kubelet/lib/systemd/system/kubelet.service" |
    sed "s:/usr/bin:${DOWNLOAD_DIR}:g" |
    tee /etc/systemd/system/kubelet.service

curl --retry-delay 1 \
    --retry 60 \
    --retry-connrefused \
    --retry-max-time 60 \
    --connect-timeout 20 \
    --fail \
    -sSL \
    "https://raw.githubusercontent.com/kubernetes/release/${RELEASE_VERSION}/cmd/kubepkg/templates/latest/deb/kubeadm/10-kubeadm.conf" |
    sed "s:/usr/bin:${DOWNLOAD_DIR}:g" |
    tee /etc/systemd/system/kubelet.service.d/10-kubeadm.conf

systemctl enable --now kubelet

cat << EOF > worker-config.yaml
{{ .WorkerConfig }}
EOF

systemctl start --quiet coreos-metadata
ipv4=$(cat /run/metadata/flatcar | grep -v -E '(IPV6|GATEWAY)' | grep IP | grep -E '(PRIVATE|LOCAL|DYNAMIC)' | cut -d = -f 2)

kubeadm join --config worker-config.yaml --node-name "${ipv4}"
`
)
