// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/sdk"
)

var (
	get = &cobra.Command{
		Use:   "get [url] [path]",
		Short: "download and verify a file from Google Storage",
		Run:   runGet,
	}

	gpgKeyFile      string
	keepSig, verify bool
)

func init() {
	bv := get.PersistentFlags().BoolVar
	sv := get.PersistentFlags().StringVar

	bv(&verify, "verify", true, "use GPG verification")
	sv(&gpgKeyFile, "verify-key", "", "PGP public key file to verify signatures, or blank for the default key built into the program")
	bv(&keepSig, "keep-sig", false, "keep the detached signature file on disk when successful")
	root.AddCommand(get)
}

func runGet(cmd *cobra.Command, args []string) {
	var output, source string

	if len(args) == 2 {
		source = args[0]
		output = args[1]
	} else if len(args) == 1 {
		source = args[0]
		output = "."
	} else {
		fmt.Fprintf(os.Stderr, "Expected one or two arguments\n")
		os.Exit(1)
	}

	// Perform some basic sanity checks on the options
	err := validateGSURL(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if output == "" {
		output = "."
	}

	// If the output path exists and is a directory, keep the file name
	if stat, err := os.Stat(output); err == nil && stat.IsDir() {
		output = path.Join(output, path.Base(source))
	}

	// Authenticate with Google
	client, err := getGoogleClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Download the file and verify it (unless disabled)
	if verify {
		err = sdk.UpdateSignedFile(output, source, client, gpgKeyFile)
		if err == nil && !keepSig {
			err = os.Remove(output + ".sig")
		}
	} else {
		err = sdk.UpdateFile(output, source, client)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
