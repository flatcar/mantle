// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/sdk"
)

var (
	downloadCmd = &cobra.Command{
		Use:   "download",
		Short: "Download the SDK tarball",
		Long:  "Download the current SDK tarball to a local cache.",
		Run:   runDownload,
	}
	URL                   string
	downloadUrl           string
	downloadVersion       string
	downloadVerifyKeyFile string
	// downloadJSONKeyFile is used to access a private GCS bucket.
	downloadJSONKeyFile string
)

func init() {
	downloadCmd.Flags().StringVar(&URL,
		"sdk-url", "mirror.release.flatcar-linux.net", "SDK URL")
	downloadCmd.Flags().StringVar(&downloadUrl,
		"sdk-url-path", "sdk", "SDK URL path")
	downloadCmd.Flags().StringVar(&downloadVersion,
		"sdk-version", "", "SDK version")
	downloadCmd.Flags().StringVar(&downloadImageVerifyKeyFile,
		"verify-key", "", "PGP public key to be used in verifing download signatures.  Defaults to CoreOS Buildbot (0412 7D0B FABE C887 1FFB  2CCE 50E0 8855 93D2 DCB4)")
	downloadCmd.Flags().StringVar(&downloadJSONKeyFile,
		"json-key", "", "Google service account key for use with private buckets")
	root.AddCommand(downloadCmd)
}

func runDownload(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		plog.Fatal("No args accepted")
	}

	if downloadVersion == "" {
		plog.Fatal("Missing --sdk-version=VERSION")
	}

	plog.Noticef("Downloading SDK version %s", downloadVersion)
	if err := sdk.DownloadSDK(URL, downloadUrl, downloadVersion, downloadVerifyKeyFile, downloadImageJSONKeyFile); err != nil {
		plog.Fatalf("Download failed: %v", err)
	}
}
