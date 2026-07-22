// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

import (
	"fmt"
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/cli"
	"github.com/flatcar/mantle/platform"
	oraclecloudapi "github.com/flatcar/mantle/platform/api/oraclecloud"
	"github.com/spf13/cobra"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "ore/oraclecloud")

	OracleCloud = &cobra.Command{
		Use:   "oraclecloud [command]",
		Short: "Oracle Cloud Infrastructure utilities",
	}

	api     *oraclecloudapi.API
	options oraclecloudapi.Options
)

func init() {
	cli.WrapPreRun(OracleCloud, preflightCheck)
	OracleCloud.PersistentFlags().StringVar(&options.Tenancy, "oraclecloud-tenancy", "", "Oracle Cloud tenancy")
	OracleCloud.PersistentFlags().StringVar(&options.User, "oraclecloud-user", "", "Oracle Cloud user")
	OracleCloud.PersistentFlags().StringVar(&options.Region, "oraclecloud-region", "us-ashburn-1", "Oracle Cloud fingerprint")
	OracleCloud.PersistentFlags().StringVar(&options.PrivateKey, "oraclecloud-private-key", "", "Oracle Cloud private key")
	OracleCloud.PersistentFlags().StringVar(&options.PrivateKeyPassphrase, "oraclecloud-private-key-passphrase", "", "Oracle Cloud private key passphrase")
	OracleCloud.PersistentFlags().StringVar(&options.Fingerprint, "oraclecloud-fingerprint", "", "Oracle Cloud fingerprint")
	OracleCloud.PersistentFlags().StringVar(&options.CompartmentID, "oraclecloud-compartment-id", "", "Oracle Cloud Infrastructure compartment OCID")
}

func preflightCheck(cmd *cobra.Command, args []string) error {
	options.Options = &platform.Options{}

	a, err := oraclecloudapi.New(&options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create Oracle Cloud Infrastructure API client: %v\n", err)
		os.Exit(1)
	}

	api = a
	return nil
}
