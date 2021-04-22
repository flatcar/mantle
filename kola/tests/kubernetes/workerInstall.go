package kubernetes

// https://github.com/coreos/coreos-kubernetes/tree/master/multi-node/generic.
const workerInstallScript = `#!/bin/bash
export CNI_VERSION="v0.9.1"

# Download dir used to store the kubernetes
# related components
export DOWNLOAD_DIR=/opt/bin

# List of etcd servers (http://ip:port), comma separated
export ETCD_ENDPOINTS={{.ETCD_ENDPOINTS}}

# The endpoint the worker node should use to contact controller nodes (https://ip:port)
# In HA configurations this should be an external DNS record or loadbalancer in front of the control nodes.
# However, it is also possible to point directly to a single control node.
export CONTROLLER_ENDPOINT={{.CONTROLLER_ENDPOINT}}

# Specify the version (vX.Y.Z) of Kubernetes assets to deploy
export K8S_VER={{.K8S_VER}}

# Hyperkube image repository to use.
export HYPERKUBE_IMAGE_REPO={{.HYPERKUBE_IMAGE_REPO}}

# The IP address of the cluster DNS service.
# This must be the same DNS_SERVICE_IP used when configuring the controller nodes.
export DNS_SERVICE_IP=192.168.128.10

# Whether to use Calico for Kubernetes network policy.
export USE_CALICO=false

# Determines the container runtime for kubernetes to use. Accepts 'docker'.
export CONTAINER_RUNTIME={{.CONTAINER_RUNTIME}}

# The above settings can optionally be overridden using an environment file:
ENV_FILE=/run/coreos-kubernetes/options.env

# -------------

function init_config {
    local REQUIRED=( 'ADVERTISE_IP' 'ETCD_ENDPOINTS' 'CONTROLLER_ENDPOINT' 'DNS_SERVICE_IP' 'K8S_VER' 'HYPERKUBE_IMAGE_REPO' 'USE_CALICO' )

    if [ -f $ENV_FILE ]; then
        export $(cat $ENV_FILE | xargs)
    fi

    if [ -z $ADVERTISE_IP ]; then
        systemctl start coreos-metadata
        export ADVERTISE_IP=$(cat /run/metadata/flatcar | grep -v IPV6 | grep IP | grep -E '(PRIVATE|LOCAL)' | cut -d = -f 2)
    fi

    for REQ in "${REQUIRED[@]}"; do
        if [ -z "$(eval echo \$$REQ)" ]; then
            echo "Missing required config value: ${REQ}"
            exit 1
        fi
    done
}

function init_templates {
    local TEMPLATE=/etc/systemd/system/kubelet.service
    if [ ! -f $TEMPLATE ]; then
        echo "TEMPLATE: $TEMPLATE"
        mkdir -p $(dirname $TEMPLATE)
        cat << EOF > $TEMPLATE
[Service]
Requires=docker.service
After=docker.service
ExecStartPre=/usr/bin/docker pull ${HYPERKUBE_IMAGE_REPO}:${K8S_VER}
ExecStartPre=/usr/bin/mkdir -p /etc/kubernetes/manifests
ExecStart=/opt/bin/kubelet \
  --cni-conf-dir=/etc/kubernetes/cni/net.d \
  --network-plugin=cni \
  --container-runtime=${CONTAINER_RUNTIME} \
  --register-node=true \
  --pod-manifest-path=/etc/kubernetes/manifests \
  --hostname-override=${ADVERTISE_IP} \
  --kubeconfig=/etc/kubernetes/worker-kubeconfig.yaml \
  --tls-cert-file=/etc/kubernetes/ssl/worker.pem \
  --tls-private-key-file=/etc/kubernetes/ssl/worker-key.pem \
  --volume-plugin-dir=/opt/libexec/kubernetes/kubelet-plugins/volume/exec/
Restart=always
RestartSec=10
CPUAccounting=true
MemoryAccounting=true

[Install]
WantedBy=multi-user.target
EOF
    fi

    local TEMPLATE=/etc/systemd/system/calico-node.service
    if [ "${USE_CALICO}" = "true" ] && [ ! -f "${TEMPLATE}" ]; then
        echo "TEMPLATE: $TEMPLATE"
        mkdir -p $(dirname $TEMPLATE)
        cat << EOF > $TEMPLATE
[Unit]
Description=Calico per-host agent
Requires=network-online.target
After=network-online.target

[Service]
Slice=machine.slice
Environment=CALICO_DISABLE_FILE_LOGGING=true
Environment=HOSTNAME=${ADVERTISE_IP}
Environment=IP=${ADVERTISE_IP}
Environment=FELIX_FELIXHOSTNAME=${ADVERTISE_IP}
Environment=CALICO_NETWORKING=false
Environment=NO_DEFAULT_POOLS=true
Environment=ETCD_ENDPOINTS=${ETCD_ENDPOINTS}
ExecStart=/usr/bin/rkt run --inherit-env --stage1-from-dir=stage1-fly.aci \
--volume=modules,kind=host,source=/lib/modules,readOnly=false \
--mount=volume=modules,target=/lib/modules \
--trust-keys-from-https --insecure-options=image docker://quay.io/calico/node:v0.19.0
KillMode=mixed
Restart=always
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF
    fi

    local TEMPLATE=/etc/kubernetes/worker-kubeconfig.yaml
    if [ ! -f $TEMPLATE ]; then
        echo "TEMPLATE: $TEMPLATE"
        mkdir -p $(dirname $TEMPLATE)
        cat << EOF > $TEMPLATE
apiVersion: v1
kind: Config
clusters:
- name: local
  cluster:
    certificate-authority: /etc/kubernetes/ssl/ca.pem
    server: ${CONTROLLER_ENDPOINT}
    clusterDNS: ${DNS_SERVICE_IP}
    clusterDomain: cluster.local
users:
- name: kubelet
  user:
    client-certificate: /etc/kubernetes/ssl/worker.pem
    client-key: /etc/kubernetes/ssl/worker-key.pem
contexts:
- context:
    cluster: local
    user: kubelet
  name: kubelet-context
current-context: kubelet-context
EOF
    fi

    KUBE_PREFIX=""
    if [[ $K8S_VER > "v1.16" ]]; then
      KUBE_PREFIX="kube-"
    fi

    local TEMPLATE=/etc/kubernetes/manifests/kube-proxy.yaml
    if [ ! -f $TEMPLATE ]; then
        echo "TEMPLATE: $TEMPLATE"
        mkdir -p $(dirname $TEMPLATE)
        cat << EOF > $TEMPLATE
apiVersion: v1
kind: Pod
metadata:
  name: kube-proxy
  namespace: kube-system
spec:
  hostNetwork: true
  containers:
  - name: kube-proxy
    image: ${HYPERKUBE_IMAGE_REPO}:$K8S_VER
    command:
    - /hyperkube
    - ${KUBE_PREFIX}proxy
    - --master=${CONTROLLER_ENDPOINT}
    - --kubeconfig=/etc/kubernetes/worker-kubeconfig.yaml
    securityContext:
      privileged: true
    volumeMounts:
    - mountPath: /etc/ssl/certs
      name: "ssl-certs"
    - mountPath: /etc/kubernetes/worker-kubeconfig.yaml
      name: "kubeconfig"
      readOnly: true
    - mountPath: /etc/kubernetes/ssl
      name: "etc-kube-ssl"
      readOnly: true
    - mountPath: /var/run/dbus
      name: dbus
      readOnly: false
  volumes:
  - name: "ssl-certs"
    hostPath:
      path: "/usr/share/ca-certificates"
  - name: "kubeconfig"
    hostPath:
      path: "/etc/kubernetes/worker-kubeconfig.yaml"
  - name: "etc-kube-ssl"
    hostPath:
      path: "/etc/kubernetes/ssl"
  - hostPath:
      path: /var/run/dbus
    name: dbus
EOF
    fi

    local TEMPLATE=/run/flannel/options.env
    if [ ! -f $TEMPLATE ]; then
        echo "TEMPLATE: $TEMPLATE"
        mkdir -p $(dirname $TEMPLATE)
        cat << EOF > $TEMPLATE
FLANNELD_IFACE=$ADVERTISE_IP
FLANNELD_ETCD_ENDPOINTS=$ETCD_ENDPOINTS
EOF
    fi

    local TEMPLATE=/etc/systemd/system/docker.service.d/40-flannel.conf
    if [ ! -f $TEMPLATE ]; then
        echo "TEMPLATE: $TEMPLATE"
        mkdir -p $(dirname $TEMPLATE)
        cat << EOF > $TEMPLATE
[Service]
EnvironmentFile=/etc/kubernetes/cni/docker_opts_cni.env
EOF
    fi

    local TEMPLATE=/etc/kubernetes/cni/docker_opts_cni.env
    if [ ! -f $TEMPLATE ]; then
        echo "TEMPLATE: $TEMPLATE"
        mkdir -p $(dirname $TEMPLATE)
        cat << EOF > $TEMPLATE
DOCKER_OPT_BIP=""
DOCKER_OPT_IPMASQ=""
EOF
    fi

    local TEMPLATE=/etc/kubernetes/cni/net.d/10-calico.conf
    if [ "${USE_CALICO}" = "true" ] && [ ! -f "${TEMPLATE}" ]; then
        echo "TEMPLATE: $TEMPLATE"
        mkdir -p $(dirname $TEMPLATE)
        cat << EOF > $TEMPLATE
{
    "name": "calico",
    "cniVersion": "0.2.0",
    "type": "flannel",
    "delegate": {
        "type": "calico",
        "etcd_endpoints": "$ETCD_ENDPOINTS",
        "log_level": "none",
        "log_level_stderr": "info",
        "hostname": "${ADVERTISE_IP}",
        "policy": {
            "type": "k8s",
            "k8s_api_root": "${CONTROLLER_ENDPOINT}:443/api/v1/",
            "k8s_client_key": "/etc/kubernetes/ssl/worker-key.pem",
            "k8s_client_certificate": "/etc/kubernetes/ssl/worker.pem"
        }
    }
}
EOF
    fi

    local TEMPLATE=/etc/kubernetes/cni/net.d/10-flannel.conf
    if [ "${USE_CALICO}" = "false" ] && [ ! -f "${TEMPLATE}" ]; then
        echo "TEMPLATE: $TEMPLATE"
        mkdir -p $(dirname $TEMPLATE)
        cat << EOF > $TEMPLATE
{
    "name": "podnet",
    "cniVersion": "0.2.0",
    "type": "flannel",
    "delegate": {
        "isDefaultGateway": true
    }
}
EOF
    fi

}

mkdir --parent /opt/cni/bin
curl -sSL --remote-name-all https://storage.googleapis.com/kubernetes-release/release/${K8S_VER}/bin/linux/amd64/kubelet
curl -sSL "https://github.com/containernetworking/plugins/releases/download/${CNI_VERSION}/cni-plugins-linux-amd64-${CNI_VERSION}.tgz" | tar -C /opt/cni/bin -xz

chmod +x kubelet
mv kubelet $DOWNLOAD_DIR/

init_config
init_templates

systemctl stop update-engine; systemctl mask update-engine

systemctl daemon-reload

systemctl enable --now flanneld
systemctl enable --now kubelet

if [ $USE_CALICO = "true" ]; then
        systemctl enable calico-node; systemctl start calico-node
fi`
