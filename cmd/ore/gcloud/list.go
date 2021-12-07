// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package gcloud

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/platform/api/gcloud"
)

var (
	cmdList = &cobra.Command{
		Use:   "list-instances --prefix=<prefix>",
		Short: "List instances on GCE",
		Run:   runList,
	}
)

func init() {
	GCloud.AddCommand(cmdList)
}

func runList(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "Unrecognized args in plume list cmd: %v\n", args)
		os.Exit(2)
	}

	vms, err := api.ListInstances(opts.BaseName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed listing vms: %v\n", err)
		os.Exit(1)
	}

	for _, vm := range vms {
		_, extIP := gcloud.InstanceIPs(vm)
		fmt.Printf("%v:\n", vm.Name)
		fmt.Printf(" extIP: %v\n", extIP)
	}
}
