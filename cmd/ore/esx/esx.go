// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package esx

import (
	"fmt"
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar-linux/mantle/auth"
	"github.com/flatcar-linux/mantle/cli"
	"github.com/flatcar-linux/mantle/platform/api/esx"
	"github.com/spf13/cobra"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "ore/esx")

	ESX = &cobra.Command{
		Use:   "esx [command]",
		Short: "esx image and vm utilities",
	}

	API     *esx.API
	options esx.Options
)

func init() {
	ESX.PersistentFlags().StringVar(&options.Server, "server", "", "ESX server")
	ESX.PersistentFlags().StringVar(&options.Profile, "profile", "", "Profile")
	ESX.PersistentFlags().StringVar(&options.ConfigPath, "esx-config-file", "", "ESX config file (default \"~/"+auth.ESXConfigPath+"\")")
	cli.WrapPreRun(ESX, preflightCheck)
}

func preflightCheck(cmd *cobra.Command, args []string) error {
	plog.Debugf("Running ESX Preflight check.")
	api, err := esx.New(&options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create ESX client: %v\n", err)
		os.Exit(1)
	}
	if err = api.PreflightCheck(); err != nil {
		fmt.Fprintf(os.Stderr, "could not complete ESX preflight check: %v\n", err)
		os.Exit(1)
	}

	plog.Debugf("Preflight check success; we have liftoff")
	API = api
	return nil
}
