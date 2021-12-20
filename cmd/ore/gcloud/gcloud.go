// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package gcloud

import (
	"github.com/coreos/pkg/capnslog"
	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/cli"
	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/gcloud"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "ore/gce")

	GCloud = &cobra.Command{
		Use:   "gcloud [command]",
		Short: "GCloud image creation and upload tools",
	}

	opts = gcloud.Options{Options: &platform.Options{}}

	api *gcloud.API
)

func init() {
	sv := GCloud.PersistentFlags().StringVar

	sv(&opts.Image, "image", "", "image name")
	sv(&opts.Project, "project", "flatcar-212911", "project")
	sv(&opts.Zone, "zone", "us-central1-a", "zone")
	sv(&opts.MachineType, "machinetype", "n1-standard-1", "machine type")
	sv(&opts.DiskType, "disktype", "pd-ssd", "disk type")
	sv(&opts.BaseName, "basename", "kola", "instance name prefix")
	sv(&opts.Network, "network", "default", "network name")
	sv(&opts.JSONKeyFile, "json-key", "", "use a service account's JSON key for authentication")
	GCloud.PersistentFlags().BoolVar(&opts.ServiceAuth, "service-auth", false, "use non-interactive auth when running within GCE")

	cli.WrapPreRun(GCloud, preauth)
}

func preauth(cmd *cobra.Command, args []string) error {
	a, err := gcloud.New(&opts)
	if err != nil {
		return err
	}

	api = a

	return nil
}
