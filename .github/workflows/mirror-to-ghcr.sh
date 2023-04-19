#!/bin/bash

# This generic script aims to mirror an image from Docker hub to another registry.
# Authentication to the registry must be done before.

set -euo pipefail

function fail {
    echo "${*}" >&2
    exit 1
}

org=flatcar
ghcrtag=''
while [[ -n "${1:-}" ]]; do
    case "${1}" in
        -o)
            if [[ -z "${2:-}" ]]; then
                fail "No meaningful value for ${1}"
            fi
            org="${2}"
            shift 2
            ;;
        -t)
            if [[ -z "${2:-}" ]]; then
                fail "No meaningful value for ${1}"
            fi
            ghcrtag="${2}"
            shift 2
            ;;
        -*)
            echo 'invalid flag'
            exit 1
            ;;
        *)
            break
            ;;
    esac
done

image="${1:-}"
if [[ -z "${image}" ]]; then
    fail "Empty image name"
fi
imagetag="${2}"
if [[ -z "${imagetag}" ]]; then
    fail "Empty image tag"
fi
if [[ -z "${ghcrtag}" ]]; then
    ghcrtag="${imagetag}"
fi

# we want both arch for running tests
platforms=( amd64 arm64 )

# tags will hold the mirrored images
tags=()

srcname="${image}:${imagetag}"
dstname="ghcr.io/${org}/${image}:${ghcrtag}"

for platform in "${platforms[@]}"; do
    # we first fetch the image from Docker Hub
    var=$(docker pull "${srcname}" --platform="linux/${platform}" -q)
    # we prepare the image to be pushed into another registry
    tag="${dstname}-${platform}"
    # we tag the image to create the mirrored image
    docker tag "${var}" "${tag}"
    docker push "${tag}"
    tags+=( "${tag}" )
done

docker manifest create "${dstname}" "${tags[@]}"
# some images have bad arch specs in the individual image manifests :(
docker manifest annotate "${dstname}" "${dstname}-arm64" --arch arm64
docker manifest push --purge "${dstname}"
