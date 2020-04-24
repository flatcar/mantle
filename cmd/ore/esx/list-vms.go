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
	cmdListVMs = &cobra.Command{
		Use:   "list-vms",
		Short: "List VMs on ESX",
		Long: `List all names of VMs on ESX

After a successful run, all names are written in one line each.
`,
		RunE: runListVMs,
	}

	patternToList string
)

func init() {
	ESX.AddCommand(cmdListVMs)
	cmdListVMs.Flags().StringVar(&patternToList, "pattern", "*", "Pattern to match for")
}

func runListVMs(cmd *cobra.Command, args []string) error {
	names, err := API.GetDevices(patternToList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't list VMs: %v\n", err)
		os.Exit(1)
	}
	for _, name := range names {
		fmt.Println(name)
	}
	return nil
}
