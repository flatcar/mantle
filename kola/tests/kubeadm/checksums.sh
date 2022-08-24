#!/bin/bash

set -euo pipefail

fail() {
    IFS=' ' echo "$*" >&2
    exit 1
}

# no limit by default
limit=''
for arg; do
    case "${arg}" in
        -h|--help)
            echo 'Prints various checksums of binaries used in kubeadm tests.

Parameters:
  -h --help                 Print this help.
  -N, where N is a number   Print checksums only for the first N entries
                            in the testConfig map in kubeadm.go.

Workflow when updating a version of Kubernetes or other binaries:
- update versions in a chosen entry in the testConfig map in kubeadm.go
- run the script
- paste the checksums into appriopriate places in the modified entry in kubeadm.go

Workflow when adding a new Kubernetes version to test:
- copy the first entry in testConfig in kubeadm.go
- update the versions in the new entry
- leave the checksums as they are
- run the script
- paste the checksums into appriopriate places in the new entry in kubeadm.go
'
            exit 0
            ;;
        -*)
            if [[ ! "${1}" =~ ^-[0-9]+$ ]]; then
                fail 'Invalid flag to limit k8s versions, must be in form of -<number>.'
            fi
            limit="${1//-/}"
            ;;
        *)
            fail "Unknown argument '${arg}', call the script with -h to see the synopsis."
            ;;
    esac
done

tuples=(
    "KubeadmSum^https://storage.googleapis.com/kubernetes-release/release/%K8S_VER%/bin/linux/%ARCH%/kubeadm"
    "KubeletSum^https://storage.googleapis.com/kubernetes-release/release/%K8S_VER%/bin/linux/%ARCH%/kubelet"
    "CRIctlSum^https://github.com/kubernetes-sigs/cri-tools/releases/download/%CRICTL_VER%/crictl-%CRICTL_VER%-linux-%ARCH%.tar.gz"
    "CNISum^https://github.com/containernetworking/plugins/releases/download/%CNI_VER%/cni-plugins-linux-%ARCH%-%CNI_VER%.tgz"
    "KubectlSum^https://storage.googleapis.com/kubernetes-release/release/%K8S_VER%/bin/linux/%ARCH%/kubectl"
)

this_dir=$(dirname "${0}")
kubeadm_go="${this_dir}/kubeadm.go"
version_pattern='v[0-9]\+\.[0-9]\+\.[0-9]\+'

get_full_pattern() {
    string_type="\[string\]"
    if [[ "${1}" = '-F' ]]; then
        string_type="[string]"
        shift
    fi
    local version_part="${1}"; shift
    echo "\"${version_part}\": map${string_type}interface{}{"
}

full_pattern=$(get_full_pattern "${version_pattern}")

k8s_releases=( $(grep -e "${full_pattern}" "${kubeadm_go}" | grep -oe "${version_pattern}") )

do_subs() {
    local template="${1}"; shift
    local key
    local value
    local -a kva
    local result="${template}"

    for kv; do
        kva=( ${kv//:/ } )
        key="%${kva[0]}%"
        value="${kva[1]}"
        result="${result//${key}/${value}}"
    done

    echo "${result}"
}

counter=0
for k8s_release in "${k8s_releases[@]}"; do
    if [[ -n "${limit}" ]] && [[ ! ${counter} -lt ${limit} ]]; then
        break
    fi
    ((++counter))
    fixed_pattern=$(get_full_pattern -F "${k8s_release}")
    output=$(grep -F -A 14 -e "${fixed_pattern}" "${kubeadm_go}")
    cni_version=$(grep -F CNIVersion <<<"${output}" | sed -e 's/.*":\s\+"\([^"]\+\)".*/\1/')
    crictl_version=$(grep -F CRIctlVersion <<<"${output}" | sed -e 's/.*":\s\+"\([^"]\+\)".*/\1/')
    for arch in arm64 amd64; do
        echo "K8s ${k8s_release}, arch ${arch}"
        for tuple in "${tuples[@]}"; do
            tuple=( ${tuple//^/ } )
            key="${tuple[0]}"
            url_template="${tuple[1]}"
            url=$(do_subs "${url_template}" "K8S_VER:${k8s_release}" "ARCH:${arch}" "CNI_VER:${cni_version}" "CRICTL_VER:${crictl_version}")
            sum=$(curl -s -S -f -L "${url}" | sha512sum | cut -f1 -d' ')
            echo "\"${key}\": \"${sum}\","
        done
        echo
    done
done
