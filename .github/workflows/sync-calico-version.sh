#!/bin/bash

set -exuo pipefail

calico_version=$(curl \
                     -H 'Accept: application/vnd.github+json' \
                     'https://api.github.com/repos/projectcalico/calico/releases' | \
                     jq --raw-output '.[].tag_name' | \
                     sort --version-sort --reverse | \
                     head --lines 1)

# The ':!.github' obviously means "exclude .github". This is to avoid
# this action to mangle itself.
files=( $(git grep 'https://[^ ]\+tigera-operator.yaml' -- . ':!.github' | cut -d: -f1) )
new_link="https://raw.githubusercontent.com/projectcalico/calico/${calico_version}/manifests/tigera-operator.yaml"

sed \
    -i \
    -e "s@https://[^ ]\+tigera-operator.yaml@${new_link}@g" \
    "${files[@]}"

# handle our patched custom resources
# - find which files are using custom resources
# - download the new manifest
# - download the old version of manifest
# - for each of the using files do
#   - extract the manifest from file
#   - create a patch from the old version to extracted
#   - patch the new manifest
#   - paste the patched new manifest into file
mkdir throw-away
trap 'rm -rf throw-away' EXIT

# - find which files are using custom resources
files=( $(git grep 'Source: https://raw\.githubusercontent\.com/projectcalico/calico/[^ /]\+/manifests/custom-resources.yaml' -- . ':!.github' | cut -d: -f1) )
# - download the new manifest
new_link="https://raw.githubusercontent.com/projectcalico/calico/${calico_version}/manifests/custom-resources.yaml"
wget "${new_link}"
mv custom-resources.yaml throw-away/custom-resources-new.yaml
# - download the old version of manifest
old_link=$(git grep -nA1 -e 'cat << EOF > calico.yaml' -- kola/tests/kubeadm/templates.go | tail -n 1 | sed -e 's/^.*Source: //')
wget "${old_link}"
mv custom-resources.yaml throw-away/custom-resources-old.yaml
# - for each of the using files do
for file in "${files[@]}"; do
    #   - extract the manifest from file
    lineno=$(git grep -ne 'cat << EOF > calico.yaml' -- "${file}" | cut -d: -f2)
    ((lineno+=2)) # skip the "cat" line and the follow-up "Source:" line
    for endlineno in $(git grep -ne '^EOF$' -- "${file}" | cut -d: -f2); do
        if [[ ${endlineno} -gt ${lineno} ]]; then
            break
        fi
    done
    lines=$((endlineno - lineno))
    tail -n "+${lineno}" "${file}" | head -n "${lines}" >throw-away/custom-resources-modified.yaml
    #   - create a patch from the old version to extracted
    diff throw-away/custom-resources-old.yaml throw-away/custom-resources-modified.yaml >throw-away/cr.patch || : # diff returns 1 if files differ
    rm -f throw-away/custom-resources-modified.yaml
    #   - patch the new manifest
    cp -a throw-away/custom-resources-new.yaml throw-away/custom-resources-new-patched.yaml
    patch --quiet throw-away/custom-resources-new-patched.yaml throw-away/cr.patch
    rm -f throw-away/cr.patch
    #   - paste the patched new manifest into file
    ((lineno-=2)) # go back to the "cat" line
    head -n "${lineno}" "${file}" >throw-away/new-file
    echo "# Source: ${new_link}" >>throw-away/new-file
    cat throw-away/custom-resources-new-patched.yaml >>throw-away/new-file
    rm -f throw-away/custom-resources-new-patched.yaml
    tail -n "+${endlineno}" "${file}" >>throw-away/new-file
    rm "${file}"
    mv throw-away/new-file "${file}"
done

update_needed=0
if git status --porcelain | grep --quiet '^ M'; then
    update_needed=1
fi

echo "CALICO_VERSION=${calico_version}" >>"${GITHUB_OUTPUT}"
echo "UPDATE_NEEDED=${update_needed}" >>"${GITHUB_OUTPUT}"
