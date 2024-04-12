// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package scaleway

import (
	"fmt"
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/cli"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/scaleway"
	"github.com/spf13/cobra"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "ore/scaleway")

	Scaleway = &cobra.Command{
		Use:   "scaleway [command]",
		Short: "scaleway image utilities",
	}

	API            *scaleway.API
	region         string
	zone           string
	accessKey      string
	secretKey      string
	organizationID string
	projectID      string
)

func init() {
	cli.WrapPreRun(Scaleway, preflightCheck)
	Scaleway.PersistentFlags().StringVar(&region, "scaleway-region", "fr-par", "Scaleway region")
	Scaleway.PersistentFlags().StringVar(&zone, "scaleway-zone", "fr-par-1", "Scaleway region")
	Scaleway.PersistentFlags().StringVar(&accessKey, "scaleway-access-key", "", "Scaleway access key")
	Scaleway.PersistentFlags().StringVar(&secretKey, "scaleway-secret-key", "", "Scaleway secret key")
	Scaleway.PersistentFlags().StringVar(&organizationID, "scaleway-organization-id", "", "Scaleway organization ID")
	Scaleway.PersistentFlags().StringVar(&projectID, "scaleway-project-id", "", "Scaleway project ID")
}

func preflightCheck(cmd *cobra.Command, args []string) error {
	api, err := scaleway.New(&scaleway.Options{
		Region:         region,
		Zone:           zone,
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		OrganizationID: organizationID,
		ProjectID:      projectID,
		Options:        &platform.Options{},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not create Scaleway API client: %v\n", err)
		os.Exit(1)
	}

	API = api
	return nil
}
