// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"fmt"

	"github.com/coreos/pkg/capnslog"
	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/auth"
	"github.com/flatcar-linux/mantle/cli"
	"github.com/flatcar-linux/mantle/platform/api/openstack"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "ore/openstack")

	OpenStack = &cobra.Command{
		Use:   "openstack [command]",
		Short: "OpenStack machine utilities",
	}

	API     *openstack.API
	options openstack.Options
)

func init() {
	OpenStack.PersistentFlags().StringVar(&options.ConfigPath, "config-file", "", "config file (default \"~/"+auth.OpenStackConfigPath+"\")")
	OpenStack.PersistentFlags().StringVar(&options.Profile, "profile", "", "profile (default \"default\")")
	cli.WrapPreRun(OpenStack, preflightCheck)
}

func preflightCheck(cmd *cobra.Command, args []string) error {
	plog.Debugf("Running OpenStack preflight check")
	api, err := openstack.New(&options)
	if err != nil {
		return fmt.Errorf("could not create OpenStack client: %v", err)
	}
	if err := api.PreflightCheck(); err != nil {
		return fmt.Errorf("could not complete OpenStack preflight check: %v", err)
	}

	plog.Debugf("Preflight check success; we have liftoff")
	API = api
	return nil
}
