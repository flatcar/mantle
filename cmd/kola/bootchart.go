// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/kola"
	"github.com/flatcar-linux/mantle/platform"
)

var cmdBootchart = &cobra.Command{
	Run:    runBootchart,
	PreRun: preRun,
	Use:    "bootchart > bootchart.svg",
	Short:  "Boot performance graphing tool",
	Long: `
Boot a single instance and plot how the time was spent.

Note that this actually uses systemd-analyze plot rather than
systemd-bootchart since the latter requires setting a different
init process.

This must run as root!
`}

func init() {
	root.AddCommand(cmdBootchart)
}

func runBootchart(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "No args accepted\n")
		os.Exit(2)
	}

	var err error
	outputDir, err = kola.SetupOutputDir(outputDir, kolaPlatform)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
		os.Exit(1)
	}

	flight, err := kola.NewFlight(kolaPlatform)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Flight failed: %v\n", err)
		os.Exit(1)
	}
	defer flight.Destroy()

	cluster, err := flight.NewCluster(&platform.RuntimeConfig{
		OutputDir: outputDir,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cluster failed: %v\n", err)
		os.Exit(1)
	}
	defer cluster.Destroy()

	m, err := cluster.NewMachine(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Machine failed: %v\n", err)
		os.Exit(1)
	}
	defer m.Destroy()

	out, stderr, err := m.SSH("systemd-analyze plot")
	if err != nil {
		fmt.Fprintf(os.Stderr, "SSH failed: %v: %s\n", err, stderr)
		os.Exit(1)
	}

	fmt.Printf("%s", out)
}
