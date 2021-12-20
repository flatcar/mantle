// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/cli"
	"github.com/flatcar-linux/mantle/kola/register"

	// Register any tests that we may wish to execute in kolet.
	_ "github.com/flatcar-linux/mantle/kola/registry"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "kolet")

	root = &cobra.Command{
		Use:   "kolet run [test] [func]",
		Short: "Native code runner for kola",
		Run:   run,
	}

	cmdRun = &cobra.Command{
		Use:   "run [test] [func]",
		Short: "Run a given test's native function",
		Run:   run,
	}
)

func run(cmd *cobra.Command, args []string) {
	cmd.Usage()
	os.Exit(2)
}

func main() {
	for testName, testObj := range register.Tests {
		if len(testObj.NativeFuncs) == 0 {
			continue
		}
		testCmd := &cobra.Command{
			Use: testName + " [func]",
			Run: run,
		}
		for nativeName := range testObj.NativeFuncs {
			nativeFunc := testObj.NativeFuncs[nativeName]
			nativeRun := func(cmd *cobra.Command, args []string) {
				if len(args) != 0 {
					cmd.Usage()
					os.Exit(2)
				}
				if err := nativeFunc(); err != nil {
					plog.Fatal(err)
				}
				// Explicitly exit successfully.
				os.Exit(0)
			}
			nativeCmd := &cobra.Command{
				Use: nativeName,
				Run: nativeRun,
			}
			testCmd.AddCommand(nativeCmd)
		}
		cmdRun.AddCommand(testCmd)
	}
	root.AddCommand(cmdRun)

	cli.Execute(root)
}
