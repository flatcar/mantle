#!/bin/bash

# Compares the AWS regions listed in the "default" partition of
# cmd/plume/containerlinux.go against the regions AWS publishes in its
# public IP ranges file, and adds any missing ones to the list.
#
# Regions that are commented out in the list (e.g. `// "ap-northeast-3",`)
# count as known and will not be proposed again, so that is the way to
# permanently skip a region.

set -euo pipefail


# Fetch all the regions served via AWS, then filter only the commerical regions
# We would not be publishing to any soverign region
aws_regions=( $(curl --fail --silent --show-error 'https://ip-ranges.amazonaws.com/ip-ranges.json' | \
                    jq --raw-output '[.prefixes[].region, .ipv6_prefixes[].region] | unique | .[]' | \
                    grep -vE '^GLOBAL$|^(cn|us-gov|us-iso[a-z]*|eusc)-') )

file=cmd/plume/containerlinux.go
known_regions=( $(awk '
    /"default": awsPartitionSpec\{/ { in_default = 1 }
    in_default && /Regions: \[\]string\{/ { in_regions = 1; next }
    in_regions && /^\t*\},?$/ { exit }
    in_regions { if (match($0, /"[a-z0-9-]+"/)) print substr($0, RSTART + 1, RLENGTH - 2) }
' "${file}") )

if [[ ${#known_regions[@]} -eq 0 ]]; then
    echo "Failed to parse the region list from ${file}" >&2
    exit 1
fi

new_regions=( $(comm -23 <(printf '%s\n' "${aws_regions[@]}" | sort -u) \
                         <(printf '%s\n' "${known_regions[@]}" | sort -u)) )

if [[ ${#new_regions[@]} -eq 0 ]]; then
    echo "No new AWS regions."
    echo "UPDATE_NEEDED=0" >>"${GITHUB_OUTPUT}"
    exit 0
fi

echo "New AWS regions: ${new_regions[*]}"

# Insert the new regions at the end of the "default" partition's list.
insert=$(printf '\t\t\t\t"%s",\n' "${new_regions[@]}")
awk -v insert="${insert}" '
    /"default": awsPartitionSpec\{/ { in_default = 1 }
    in_default && /Regions: \[\]string\{/ { in_regions = 1 }
    in_regions && /^\t*\},?$/ { print insert; in_regions = 0; in_default = 0 }
    { print }
' "${file}" >"${file}.new"
mv "${file}.new" "${file}"

joined=$(IFS=,; echo "${new_regions[*]}")
joined=${joined//,/, }

{
    echo "## New AWS regions"
    echo
    echo "AWS has published the following regions, which are not yet in the \`default\` partition of \`${file}\`:"
    echo
    printf '- `%s`\n' "${new_regions[@]}"
    echo
    echo "## Before merging"
    echo
    echo "New AWS regions are opt-in. Publishing to them will fail until they are enabled in the Flatcar AWS account."
    echo
    for region in "${new_regions[@]}"; do
        echo "- [ ] Enable \`${region}\`: \`aws account enable-region --region-name ${region}\`"
    done
    echo
    echo "Enabling a region can take a while. Check that each region shows \`ENABLED\` with:"
    echo "\`aws account get-region-opt-status --region-name $REGION\`"
    echo
} >"${RUNNER_TEMP}/aws-regions-pr-body.md"

{
    echo "UPDATE_NEEDED=1"
    echo "NEW_REGIONS=${joined}"
    echo "PR_BODY_PATH=${RUNNER_TEMP:-/tmp}/aws-regions-pr-body.md"
} >>"${GITHUB_OUTPUT}"
