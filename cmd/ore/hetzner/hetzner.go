// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package hetzner

import (
	"fmt"
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/cli"
	"github.com/flatcar/mantle/platform/api/hetzner"
	"github.com/spf13/cobra"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "ore/hetzner")

	Hetzner = &cobra.Command{
		Use:   "hetzner [command]",
		Short: "hetzner image utilities",
	}

	API     *hetzner.API
	options hetzner.Options
)

func init() {
	cli.WrapPreRun(Hetzner, preflightCheck)
	Hetzner.PersistentFlags().StringVar(&options.Token, "hetzner-token", "", "Hetzner token for client authentication")
	Hetzner.PersistentFlags().StringVar(&options.Location, "hetzner-location", "", "Hetzner location name")
}

func preflightCheck(cmd *cobra.Command, args []string) error {
	api, err := hetzner.New(&options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create Hetzner API client: %v\n", err)
		os.Exit(1)
	}

	API = api
	return nil
}
