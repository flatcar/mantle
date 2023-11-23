// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package brightbox

import (
	"fmt"

	"github.com/coreos/pkg/capnslog"
	"github.com/spf13/cobra"

	"github.com/flatcar/mantle/cli"
	"github.com/flatcar/mantle/platform/api/brightbox"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "ore/brightbox")

	Brightbox = &cobra.Command{
		Use:   "brightbox [command]",
		Short: "Brightbox machine utilities",
	}

	API     *brightbox.API
	options brightbox.Options
)

func init() {
	Brightbox.PersistentFlags().StringVar(&options.ClientID, "brightbox-client-id", "", "Brightbox client ID")
	Brightbox.PersistentFlags().StringVar(&options.ClientSecret, "brightbox-client-secret", "", "Brightbox client secret")
	cli.WrapPreRun(Brightbox, preflightCheck)
}

func preflightCheck(cmd *cobra.Command, args []string) error {
	api, err := brightbox.New(&options)
	if err != nil {
		return fmt.Errorf("creating Brightbox client: %w", err)
	}

	API = api
	return nil
}
