#!/bin/bash -e
#
# Create an image from a page blob and copy it to a custom region (so as to
# cirumvent regional CPU limitationf when running tests)
#

# dummy "az" for testing
#function az() {
#	echo "EXEC: az $@"
#}

function channel_to_region() {
	local blob_name="$1"
	local channel=$(echo "$blob_name" \
						| sed 's/.*\(alpha\|beta\|stable\|edge\).*/\1/')

	case "$channel" in
		alpha)	echo "uksouth";;
		beta)	echo "germanywestcentral";;
		stable) echo "francecentral";;
		edge)	echo "ukwest";;
		*)		echo "ERROR: unknown channel '$channel'" >&2
				exit 1;;
	esac
}
# --

function disk_from_page_blob() {
	local blob_name="$1"
	local name="$2"
	local rg="$3"

	local out=$(mktemp)
	trap "rm -f $out" EXIT

	az disk create --name "$name" -g "$rg" \
		--source https://$rg.blob.core.windows.net/publish/"$blob_name".vhd \
		> "$out"
	
	# print disk id
	cat "$out" | jq -r '.id'

	rm -f "$out"
	trap - EXIT
}
# --

function image_from_disk() {
	local name="$1"
	local rg="$2"
	local disk_id="$3"

	az image create -g $rg --name "$name" \
						   --source "$disk_id" --os-type linux
}
# --

function copy_to_region() {
	local name="$1"
	local rg="$2"
	local dest_rg="$3"
	local dest_region="$4"

	az image copy --source-resource-group $rg \
				  --source-object-name "$name" \
				  --target-location "$dest_region" \
				  --target-resource-group "$dest_rg" \
				  --cleanup
}
# --

function cleanup() {
	local disk_id="$1"
	local rg="$2"
	local name="$3"
	az disk delete --no-wait --yes --ids "$disk_id"
	az image delete -g "$rg" --name "$name"
}
# --

function main() {
	local rg="flatcar"

	local blob_name="$1"
	[ -z "$blob_name" ] && {
			echo
			echo "Usage: $0 <page-blob-name-without-vhd-suffix>"
			echo ""
			echo "		e.g."
			echo "		$0 flatcar-linux-2430.0.0-alpha"
			echo ""
			exit 1
	}

	local image_name="$blob_name-release-testing"

	local dest_region="$(channel_to_region $blob_name)"
	local dest_rg="flatcar-release-testing-$dest_region"

	echo
	echo "Creating VM disk image $image_name from page blob $blob_name"
	echo " in resource group $rg"
	echo " and storing it in rg $dest_rg in region $dest_region."
	echo
	
	echo " #### 1. Creating disk from page blob"
	local disk_id=$(disk_from_page_blob "$blob_name" "$image_name" "$rg")
	echo " ----> Generated disk with ID $disk_id"
	
	echo " #### 2. Creating image from disk"
	image_from_disk "$image_name" "$rg" "$disk_id"

	echo " #### 3. Copying image to rg $dest_rg in region $dest_region"
	copy_to_region "$image_name" "$rg" "$dest_rg" "$dest_region"

	echo " #### 4. Cleanup: deleting disk + image in source region"
	cleanup "$disk_id" "$rg" "$image_name"

	echo " ====> All done."
}
# --

if [ "$(basename $0)" = "azure-prep-image.sh" ] ; then
	main $@
fi
