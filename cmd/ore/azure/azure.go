// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"github.com/coreos/pkg/capnslog"
	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/auth"
	"github.com/flatcar-linux/mantle/cli"
	"github.com/flatcar-linux/mantle/platform/api/azure"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "ore/azure")

	Azure = &cobra.Command{
		Use:   "azure [command]",
		Short: "azure image and vm utilities",
	}

	azureProfile      string
	azureAuth         string
	azureSubscription string
	azureLocation     string

	api *azure.API
)

func init() {
	cli.WrapPreRun(Azure, preauth)

	sv := Azure.PersistentFlags().StringVar
	sv(&azureProfile, "azure-profile", "", "Azure Profile json file")
	sv(&azureAuth, "azure-auth", "", "Azure auth location (default \"~/"+auth.AzureAuthPath+"\")")
	sv(&azureSubscription, "azure-subscription", "", "Azure subscription name. If unset, the first is used.")
	sv(&azureLocation, "azure-location", "westus", "Azure location (default \"westus\")")
}

func preauth(cmd *cobra.Command, args []string) error {
	plog.Printf("Creating Azure API...")

	a, err := azure.New(&azure.Options{
		AzureProfile:      azureProfile,
		AzureAuthLocation: azureAuth,
		AzureSubscription: azureSubscription,
		Location:          azureLocation,
	})
	if err != nil {
		plog.Fatalf("Failed to create Azure API: %v", err)
	}

	api = a
	return nil
}
