// Copyright 2020 Kinvolk GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package esx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cmdRemoveVMs = &cobra.Command{
		Use:   "remove-vms",
		Short: "Remove VMs on ESX",
		Long: `Remove all VMs on ESX that match a pattern

After a successful run, all names of deleted VMs are written in one line each.
`,
		RunE: runRemoveVMs,
	}

	patternToRemove string
)

func init() {
	ESX.AddCommand(cmdRemoveVMs)
	cmdRemoveVMs.Flags().StringVar(&patternToRemove, "pattern", "*", "Pattern that VMs to be removed should match")
}

func runRemoveVMs(cmd *cobra.Command, args []string) error {
	names, err := API.GetDevices(patternToRemove)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't list VMs: %v\n", err)
		os.Exit(1)
	}

	var failed bool
	for _, name := range names {
		err := API.TerminateDevice(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't delete VM %q: %v\n", name, err)
			failed = true
		}
		fmt.Println(name)
	}

	if failed {
		os.Exit(1)
	}
	return nil
}
