// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package oracle

import (
	"fmt"
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/cli"
	"github.com/flatcar/mantle/platform"
	oracleapi "github.com/flatcar/mantle/platform/api/oracle"
	"github.com/spf13/cobra"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "ore/oracle")

	Oracle = &cobra.Command{
		Use:   "oracle [command]",
		Short: "Oracle Cloud Infrastructure utilities",
	}

	api     *oracleapi.API
	options oracleapi.Options
)

func init() {
	cli.WrapPreRun(Oracle, preflightCheck)
	Oracle.PersistentFlags().StringVar(&options.ConfigFile, "oracle-config-file", "~/.oci/config", "Oracle Cloud Infrastructure config file")
	Oracle.PersistentFlags().StringVar(&options.Profile, "oracle-profile", "DEFAULT", "Oracle Cloud Infrastructure config profile")
	Oracle.PersistentFlags().StringVar(&options.CompartmentID, "oracle-compartment-id", "", "Oracle Cloud Infrastructure compartment OCID")
	Oracle.PersistentFlags().StringVar(&options.Namespace, "oracle-namespace", "", "Oracle Cloud Infrastructure Object Storage namespace (default: auto-detect)")
	Oracle.PersistentFlags().StringVar(&options.Bucket, "oracle-bucket", "", "Oracle Cloud Infrastructure Object Storage bucket for image uploads")
}

func preflightCheck(cmd *cobra.Command, args []string) error {
	options.Options = &platform.Options{}

	a, err := oracleapi.New(&options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create Oracle Cloud Infrastructure API client: %v\n", err)
		os.Exit(1)
	}

	api = a
	return nil
}
