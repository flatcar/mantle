// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package stackit

import (
	"fmt"
	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/cli"
	"github.com/flatcar/mantle/platform/api/stackit"
	"github.com/spf13/cobra"
	"os"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "ore/stackit")

	STACKIT = &cobra.Command{
		Use:   "stackit [command]",
		Short: "stackit image utilities",
	}

	API     *stackit.API
	options stackit.Options
)

func init() {
	cli.WrapPreRun(STACKIT, preflightCheck)
	STACKIT.PersistentFlags().StringVar(&options.Region, "stackit-region", "eu01", "STACKIT region")
	STACKIT.PersistentFlags().StringVar(&options.ProjectId, "stackit-project-id", "", "STACKIT project ID")
	STACKIT.PersistentFlags().StringVar(&options.ServiceAccountKeyPath, "stackit-service-account-key-path", "", "STACKIT service account key path")
}

func preflightCheck(cmd *cobra.Command, args []string) error {
	a, err := stackit.New(&options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create STACKIT API client: %v\n", err)
		os.Exit(1)
	}

	API = a
	return nil
}
