#!/bin/bash

function list_all_images() {
	local rg_pattern="$1"
	for rg in $(az group list | jq -r '.[].name' | grep "$rg_pattern"); do
		az image list -g "$rg" | jq -r '.[].id'
	done
}
# --

function run_test() {
	local image="$1"
	local name="$2"
	local location="$3"
	local parallel="$4"

	bin/kola run -d --platform azure --azure-disk-uri $image \
			  --azure-location $location --parallel $parallel \
		| tee -a $name.log
}
# --

function main() {

	# for channel_to_region()
	source "$(dirname $BASH_SOURCE[0])/azure-prep-image.sh"

	local rg_pattern="flatcar-release-testing"
	local parallel=1

	images=$(list_all_images "$rg_pattern")

	echo " #### Running kola tests for images:"
	for i in $images; do
		echo " --> $i"
	done
	echo

	exit

	for i in $images; do
		local name="$(basename $i)"
		local region="$(channel_to_region $name)"
		echo "#############################################"
		echo " Running tests for $name"
		echo "---------------------------------------------"
		run_test "$i" "$name" "$region" "$parallel"
	done
}
# --

if [ "$(basename $0)" = "azure-run-kola.sh" ] ; then
	main $@
fi
