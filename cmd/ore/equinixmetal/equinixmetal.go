// Copyright The Mantle Authors
// Copyright 2017 CoreOS, Inc.
// SPDX-License-Identifier: Apache-2.0

package equinixmetal

import (
	"fmt"
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar-linux/mantle/auth"
	"github.com/flatcar-linux/mantle/cli"
	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/equinixmetal"
	"github.com/flatcar-linux/mantle/platform/api/gcloud"
	"github.com/spf13/cobra"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "ore/equinixmetal")

	EquinixMetal = &cobra.Command{
		Use:   "equinixmetal [command]",
		Short: "EquinixMetal machine utilities",
	}

	API       *equinixmetal.API
	options   = equinixmetal.Options{Options: &platform.Options{}}
	gsOptions gcloud.Options
)

func init() {
	options.GSOptions = &gsOptions
	EquinixMetal.PersistentFlags().StringVar(&options.StorageURL, "storage-url", "gs://users.developer.core-os.net/"+os.Getenv("USER")+"/mantle", "Google Storage base URL for temporary uploads")
	EquinixMetal.PersistentFlags().StringVar(&gsOptions.JSONKeyFile, "gs-json-key", "", "use a Google service account's JSON key to authenticate to Google Storage")
	EquinixMetal.PersistentFlags().BoolVar(&gsOptions.ServiceAuth, "gs-service-auth", false, "use non-interactive Google auth when running within GCE")
	EquinixMetal.PersistentFlags().StringVar(&options.ConfigPath, "config-file", "", "config file (default \"~/"+auth.EquinixMetalConfigPath+"\")")
	EquinixMetal.PersistentFlags().StringVar(&options.Profile, "profile", "", "profile (default \"default\")")
	EquinixMetal.PersistentFlags().StringVar(&options.ApiKey, "api-key", "", "API key (overrides config file)")
	EquinixMetal.PersistentFlags().StringVar(&options.Project, "project", "", "project UUID (overrides config file)")
	cli.WrapPreRun(EquinixMetal, preflightCheck)

}

func preflightCheck(cmd *cobra.Command, args []string) error {
	plog.Debugf("Running EquinixMetal preflight check")
	api, err := equinixmetal.New(&options)
	if err != nil {
		return fmt.Errorf("could not create EquinixMetal client: %v", err)
	}
	if err := api.PreflightCheck(); err != nil {
		return fmt.Errorf("could not complete EquinixMetal preflight check: %v", err)
	}

	plog.Debugf("Preflight check success; we have liftoff")
	API = api
	return nil
}
