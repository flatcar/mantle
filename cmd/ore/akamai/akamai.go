// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package akamai

import (
	"fmt"
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/cli"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/akamai"
	"github.com/spf13/cobra"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "ore/akamai")

	Akamai = &cobra.Command{
		Use:   "akamai [command]",
		Short: "akamai image utilities",
	}

	api    *akamai.API
	region string
	token  string
)

func init() {
	cli.WrapPreRun(Akamai, preflightCheck)
	Akamai.PersistentFlags().StringVar(&region, "akamai-region", "us-ord", "Akamai region")
	Akamai.PersistentFlags().StringVar(&token, "akamai-token", "", "Akamai access token")
}

func preflightCheck(cmd *cobra.Command, args []string) error {
	a, err := akamai.New(&akamai.Options{
		Region:  region,
		Token:   token,
		Options: &platform.Options{},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create Akamai API client: %v\n", err)
		os.Exit(1)
	}

	api = a
	return nil
}
