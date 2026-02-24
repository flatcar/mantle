#!/bin/bash

# Get latest version in Gentoo repo
BCC_GENTOO_VERSION=$(ls -1 gentoo/dev-util/bcc/bcc-*.ebuild | \
                         sed -e 's#^.*/bcc-\(.*\)\.ebuild#\1#' | \
                         sort -V -r | \
                         head -n1)
echo "Found version in Gentoo: ${BCC_GENTOO_VERSION}"

case ${ACCOUNT_TYPE} in
    'org'|'user') :;;
    '') echo "No account type passed through ACCOUNT_TYPE env var" >&2; exit 1;;
    *) echo "Wrong account type ${ACCOUNT_TYPE@Q}, should be either 'user' or 'org'" >&2; exit 1;;
esac

if [[ -z ${ACCOUNT} ]]; then
    echo "No account passed through ACCOUNT env var" >&2
    exit 1
fi

if [[ -z ${TOKEN} ]]; then
    echo "No token passed through TOKEN env var" >&2
    exit 1
fi

if [[ ${FORCE:-} != 'true' ]]; then
    FORCE=''
fi

if [[ -z ${FORCE} ]]; then
    URL="https://api.github.com/${ACCOUNT_TYPE}s/${ACCOUNT}/packages/container/bcc/versions"

    echo "Sending request to ${URL}"

    # Get latest version in ghcr.io
    curl_opts=(
        --location
        --request 'GET'
        --url "${URL}"
        --header "Authorization: Bearer ${TOKEN}"
        --header 'X-GitHub-Api-Version: 2022-11-28'
    )
    curl "${curl_opts[@]}" >json_output
    echo "Got reply:"
    cat json_output
    jq --raw-output '.[].metadata.container.tags.[]' json_output >versions
    echo "Found versions in ghcr:"
    cat versions
    BCC_GHCR_VERSION=$(grep --invert-match --fixed-strings latest versions | \
                           sort -V -r | \
                           head -n1)
    echo "Latest version in ghcr: ${BCC_GHCR_VERSION}"

    # Given the versions, check if we should build a new image.
    build_version=''
    if [[ -z ${BCC_GENTOO_VERSION} ]]; then
        echo "No dev-util/bcc ebuilds in Gentoo?"
    elif [[ -z ${BCC_GHCR_VERSION} ]]; then
        echo "No versioned images in ghcr, building an image"
        build_version=${BCC_GENTOO_VERSION}
    else
        greater_version=$(printf '%s\n' "${BCC_GENTOO_VERSION}" "${BCC_GHCR_VERSION}" | \
                              sort -V -r | \
                              head -n1)
        if [[ ${greater_version} != "${BCC_GHCR_VERSION}" ]]; then
            echo "Gentoo has a greater version of bcc, building an image"
            build_version=${BCC_GENTOO_VERSION}
        else
            echo "We have latest version available in ghcr"
        fi
    fi
else
    echo "Forcing the build of the image"
    build_version=${BCC_GENTOO_VERSION}
fi
echo "BUILD_VERSION=${build_version}" >>"${GITHUB_OUTPUT}"
