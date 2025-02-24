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
	workerConfig = `---
variant: flatcar
version: 1.0.0
systemd:
  units:
{{ if .cgroupv1 }}
    - name: containerd.service
      dropins:
      - name: 10-use-cgroupfs.conf
        contents: |
          [Service]
          Environment=CONTAINERD_CONFIG=/usr/share/containerd/config-cgroupfs.toml
{{ end }}
storage:
  links:
    - target: /opt/extensions/kubernetes/kubernetes-{{ .Release }}-{{ if eq .Arch "amd64" }}x86-64{{ else }}arm64{{ end }}.raw
      path: /etc/extensions/kubernetes.raw
      hard: false
  files:
{{ if .cgroupv1 }}
    - path: /etc/flatcar-cgroupv1
      mode: 0444
{{ end }}
    - path: /home/core/install.sh
      mode: 0755
      contents:
        source: "data:text/plain;base64,{{ .WorkerScript }}"
    - path: /opt/extensions/kubernetes/kubernetes-{{ .Release }}-{{ if eq .Arch "amd64" }}x86-64{{ else }}arm64{{ end }}.raw
      contents:
        source: https://github.com/flatcar/sysext-bakery/releases/download/latest/kubernetes-{{ .Release }}-{{ if eq .Arch "amd64"}}x86-64{{ else }}arm64{{ end }}.raw
`

	masterConfig = `---
variant: flatcar
version: 1.0.0
systemd:
  units:{{ if .cgroupv1 }}
  - name: containerd.service
    dropins:
    - name: 10-use-cgroupfs.conf
      contents: |
        [Service]
        Environment=CONTAINERD_CONFIG=/usr/share/containerd/config-cgroupfs.toml{{ end }}
  - name: prepare-helm.service
    enabled: true
    contents: |
      [Unit]
      Description=Unpack helm to /opt/bin
      ConditionPathExists=!/opt/bin/helm
      [Service]
      Type=oneshot
      RemainAfterExit=true
      Restart=on-failure
      ExecStartPre=/usr/bin/mkdir --parents "{{ .DownloadDir }}"
      ExecStartPre=/usr/bin/tar -v --extract --file "/opt/helm-{{ .HelmVersion }}-linux-{{ .Arch }}.tar.gz" --directory "{{ .DownloadDir }}" --strip-components=1 --no-same-owner
      ExecStart=/usr/bin/rm "/opt/helm-{{ .HelmVersion }}-linux-{{ .Arch }}.tar.gz"
      [Install]
      WantedBy=multi-user.target
storage:
  links:
    - target: /opt/extensions/kubernetes/kubernetes-{{ .Release }}-{{ if eq .Arch "amd64" }}x86-64{{ else }}arm64{{ end }}.raw
      path: /etc/extensions/kubernetes.raw
      hard: false
  files:{{ if .cgroupv1 }}
    - path: /etc/flatcar-cgroupv1
      mode: 0444{{ end }}
    - path: /opt/helm-{{ .HelmVersion }}-linux-{{ .Arch }}.tar.gz
      mode: 0755
      contents:
        source: https://get.helm.sh/helm-{{ .HelmVersion }}-linux-{{ .Arch }}.tar.gz
    - path: /opt/extensions/kubernetes/kubernetes-{{ .Release }}-{{ if eq .Arch "amd64" }}x86-64{{ else }}arm64{{ end }}.raw
      contents:
        source: https://github.com/flatcar/sysext-bakery/releases/download/latest/kubernetes-{{ .Release }}-{{ if eq .Arch "amd64"}}x86-64{{ else }}arm64{{ end }}.raw
  {{ if eq .CNI "cilium" }}
    - path: {{ .DownloadDir }}/cilium.tar.gz
      mode: 0755
      contents:
        source: https://github.com/cilium/cilium-cli/releases/download/{{ .CiliumCLIVersion }}/cilium-linux-{{ .Arch }}.tar.gz
  {{ end }}
    - path: /home/core/install.sh
      mode: 0755
      contents:
        source: "data:text/plain;base64,{{ .MasterScript }}"
    - path: /home/core/nginx.yaml
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
    - path: /home/core/nfs-pod.yaml
      mode: 0644
      contents:
        inline: |
          apiVersion: v1
          kind: Pod
          metadata:
            name: test-pod-1
          spec:
            containers:
              - name: test
                image: ghcr.io/flatcar/nginx
                volumeMounts:
                  - name: config
                    mountPath: /test
            volumes:
              - name: config
                persistentVolumeClaim:
                  claimName: test-dynamic-volume-claim
    - path: /home/core/nfs-pvc.yaml
      mode: 0644
      contents:
        inline: |
          kind: PersistentVolumeClaim
          apiVersion: v1
          metadata:
            name: test-dynamic-volume-claim
          spec:
            storageClassName: "nfs"
            accessModes:
              - ReadWriteMany
            resources:
              requests:
                storage: 100Mi
`

	masterScript = `#!/bin/bash
set -euo pipefail

# we get the node cgroup driver
# in order to pass the params to the
# kubelet config for both controller and worker
cgroup=$(docker info | awk '/Cgroup Driver/ { print $3}')

{{ if eq .Platform "do" }}
systemctl start --quiet coreos-metadata
ipv4=$(cat /run/metadata/flatcar | grep -v -E '(IPV6|GATEWAY)' | grep IP | grep -E '(PUBLIC|LOCAL|DYNAMIC)' | cut -d = -f 2)
{{ end }}

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
{{ if eq .Platform "do" }}
    # On Digital Ocean, the private node IP is not reachable from one node to the other - let's use the public one.
    node-ip: "${ipv4}"
{{ end }}
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
# Source: https://raw.githubusercontent.com/projectcalico/calico/v3.29.2/manifests/custom-resources.yaml
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
{{ if eq .Platform "do" }}
    # On Digital Ocean, there is two network interfaces: eth0 and eth1
    # We use the one with a public IP (eth0)
    nodeAddressAutodetectionV4:
      interface: eth0
{{ end }}
    ipPools:
    - name: default-ipv4-ippool
      blockSize: 26
      cidr: {{ .PodSubnet }}
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
EOF
{{ end }}

{
    kubeadm config images pull
    kubeadm init --config kubeadm-config.yaml
    mkdir --parent "${HOME}"/.kube /home/core/.kube
    cp /etc/kubernetes/admin.conf "${HOME}"/.kube/config
    cp /etc/kubernetes/admin.conf /home/core/.kube/config
    chown -R core:core /home/core/.kube; chmod a+r /home/core/.kube/config;

{{ if eq .CNI "calico" }}
    kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.29.2/manifests/tigera-operator.yaml
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
    { grep -q svirt_lxc_file_t /etc/selinux/mcs/contexts/lxc_contexts && kubectl --namespace kube-system patch daemonset/cilium -p '{"spec":{"template":{"spec":{"containers":[{"name":"cilium-agent","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}}],"initContainers":[{"name":"mount-cgroup","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}},{"name":"apply-sysctl-overwrites","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}},{"name":"clean-cilium-state","securityContext":{"seLinuxOptions":{"level":"s0","type":"unconfined_t"}}}]}}}}'; } || true
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

cat << EOF > worker-config.yaml
{{ .WorkerConfig }}
EOF

systemctl start --quiet coreos-metadata
ipv4=$(cat /run/metadata/flatcar | grep -v -E '(IPV6|GATEWAY)' | grep IP | grep -E '({{ if eq .Platform "do" }}PUBLIC{{ else }}PRIVATE{{ end }}|LOCAL|DYNAMIC)' | cut -d = -f 2)

kubeadm join --config worker-config.yaml --node-name "${ipv4}"
`
)
