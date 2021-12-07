// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/kola"
)

var (
	cmdCheckConsole = &cobra.Command{
		Use:    "check-console [input-file...]",
		Run:    runCheckConsole,
		PreRun: preRun,
		Short:  "Check console output for badness.",
		Long: `
Check console output for expressions matching failure messages logged
by a Container Linux instance.

If no files are specified as arguments, stdin is checked.
`}

	checkConsoleVerbose bool
)

func init() {
	cmdCheckConsole.Flags().BoolVarP(&checkConsoleVerbose, "verbose", "v", false, "output user input prompts")
	root.AddCommand(cmdCheckConsole)
}

func runCheckConsole(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		// default to stdin
		args = append(args, "-")
	}

	errors := 0
	for _, arg := range args {
		var console []byte
		var err error
		sourceName := arg
		if arg == "-" {
			sourceName = "stdin"
			if checkConsoleVerbose {
				fmt.Printf("Reading input from %s...\n", sourceName)
			}
			console, err = ioutil.ReadAll(os.Stdin)
		} else {
			console, err = ioutil.ReadFile(arg)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			errors += 1
			continue
		}
		for _, badness := range kola.CheckConsole(console, nil) {
			fmt.Printf("%v: %v\n", sourceName, badness)
			errors += 1
		}
	}
	if errors > 0 {
		os.Exit(1)
	}
}
