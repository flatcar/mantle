// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package packet

import (
	"fmt"
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar-linux/mantle/auth"
	"github.com/flatcar-linux/mantle/cli"
	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/gcloud"
	"github.com/flatcar-linux/mantle/platform/api/packet"
	"github.com/spf13/cobra"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "ore/packet")

	Packet = &cobra.Command{
		Use:   "packet [command]",
		Short: "Packet machine utilities",
	}

	API       *packet.API
	options   = packet.Options{Options: &platform.Options{}}
	gsOptions gcloud.Options
)

func init() {
	options.GSOptions = &gsOptions
	Packet.PersistentFlags().StringVar(&options.StorageURL, "storage-url", "gs://users.developer.core-os.net/"+os.Getenv("USER")+"/mantle", "Google Storage base URL for temporary uploads")
	Packet.PersistentFlags().StringVar(&gsOptions.JSONKeyFile, "gs-json-key", "", "use a Google service account's JSON key to authenticate to Google Storage")
	Packet.PersistentFlags().BoolVar(&gsOptions.ServiceAuth, "gs-service-auth", false, "use non-interactive Google auth when running within GCE")
	Packet.PersistentFlags().StringVar(&options.ConfigPath, "config-file", "", "config file (default \"~/"+auth.PacketConfigPath+"\")")
	Packet.PersistentFlags().StringVar(&options.Profile, "profile", "", "profile (default \"default\")")
	Packet.PersistentFlags().StringVar(&options.ApiKey, "api-key", "", "API key (overrides config file)")
	Packet.PersistentFlags().StringVar(&options.Project, "project", "", "project UUID (overrides config file)")
	cli.WrapPreRun(Packet, preflightCheck)

}

func preflightCheck(cmd *cobra.Command, args []string) error {
	plog.Debugf("Running Packet preflight check")
	api, err := packet.New(&options)
	if err != nil {
		return fmt.Errorf("could not create Packet client: %v", err)
	}
	if err := api.PreflightCheck(); err != nil {
		return fmt.Errorf("could not complete Packet preflight check: %v", err)
	}

	plog.Debugf("Preflight check success; we have liftoff")
	API = api
	return nil
}
