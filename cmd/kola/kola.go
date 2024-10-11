// Copyright 2015 CoreOS, Inc.
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

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"golang.org/x/crypto/ssh/agent"

	"github.com/coreos/go-semver/semver"
	"github.com/coreos/pkg/capnslog"
	"github.com/spf13/cobra"

	"github.com/flatcar/mantle/cli"
	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/register"

	// register OS test suite
	_ "github.com/flatcar/mantle/kola/registry"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "kola")

	root = &cobra.Command{
		Use:   "kola [command]",
		Short: "The CoreOS Superdeep Borehole",
		// http://en.wikipedia.org/wiki/Kola_Superdeep_Borehole
	}

	cmdRun = &cobra.Command{
		Use:   "run [glob pattern...]",
		Short: "Run kola tests by category",
		Long: `Run all kola tests (default) or related groups.

If the glob pattern is exactly equal to the name of a single test, any
restrictions on the versions of Container Linux supported by that test
will be ignored.
`,
		Run:    runRun,
		PreRun: preRun,
	}

	cmdList = &cobra.Command{
		Use:   "list [glob pattern..., only for --filter, defaults to '*']",
		Short: "List kola test names",
		Run:   runList,
	}

	listJSON   bool
	listFilter bool

	runRemove     bool
	runSetSSHKeys bool
	runSSHKeys    []string
)

func init() {
	root.AddCommand(cmdRun)
	root.AddCommand(cmdList)

	cmdList.Flags().BoolVar(&listJSON, "json", false, "format output in JSON")
	cmdList.Flags().BoolVar(&listFilter, "filter", false, "Filter by --platform and --distro, required for glob patterns, uses '*' as pattern if no pattern is specified")

	cmdRun.Flags().BoolVarP(&runRemove, "remove", "r", true, "remove instances after test exits (--remove=false will keep them)")
	cmdRun.Flags().BoolVarP(&runSetSSHKeys, "keys", "k", false, "add SSH keys from --key options")
	cmdRun.Flags().StringSliceVar(&runSSHKeys, "key", nil, "path to SSH public key (default: SSH agent + ~/.ssh/id_{rsa,dsa,ecdsa,ed25519}.pub)")

}

func main() {
	cli.Execute(root)
}

func preRun(cmd *cobra.Command, args []string) {
	err := syncOptions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(3)
	}

	// EquinixMetal uses storage, and storage talks too much.
	if !plog.LevelAt(capnslog.INFO) {
		mantleLogger := capnslog.MustRepoLogger("github.com/flatcar/mantle")
		mantleLogger.SetLogLevel(map[string]capnslog.LogLevel{
			"storage": capnslog.WARNING,
		})
	}
}

func runRun(cmd *cobra.Command, args []string) {
	var patterns []string
	if len(args) >= 1 {
		patterns = args
	} else {
		patterns = []string{"*"} // run all tests by default
	}

	var err error
	outputDir, err = kola.SetupOutputDir(outputDir, kolaPlatform)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	var sshKeys []agent.Key
	if runSetSSHKeys {
		sshKeys, err = GetSSHKeys(runSSHKeys)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	} else {
		sshKeys = nil
	}
	runErr := kola.RunTests(patterns, kolaChannel, kolaOffering, kolaPlatform, outputDir, &sshKeys, runRemove)

	// needs to be after RunTests() because harness empties the directory
	if err := writeProps(); err != nil {
		plog.Fatal(err)
	}

	if runErr != nil {
		plog.Fatal(runErr)
	}
}

func writeProps() error {
	f, err := os.OpenFile(filepath.Join(outputDir, "properties.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")

	type AWS struct {
		Region       string `json:"region"`
		AMI          string `json:"ami"`
		InstanceType string `json:"type"`
	}
	type Azure struct {
		DiskURI   string `json:"diskUri"`
		BlobURL   string `json:"blobUrl"`
		ImageFile string `json:"imageFile"`
		Publisher string `json:"publisher"`
		Offer     string `json:"offer"`
		Sku       string `json:"sku"`
		Version   string `json:"version"`
		Location  string `json:"location"`
		Size      string `json:"size"`
	}
	type DO struct {
		Region string `json:"region"`
		Size   string `json:"size"`
		Image  string `json:"image"`
	}
	type ESX struct {
		Server     string `json:"server"`
		BaseVMName string `json:"base_vm_name"`
	}
	type GCE struct {
		Image       string `json:"image"`
		MachineType string `json:"type"`
	}
	type OpenStack struct {
		Region string `json:"region"`
		Image  string `json:"image"`
		Flavor string `json:"flavor"`
	}
	type EquinixMetal struct {
		Metro                 string `json:"metro"`
		Plan                  string `json:"plan"`
		InstallerImageBaseURL string `json:"installer"`
		ImageURL              string `json:"image"`
	}
	type QEMU struct {
		Image   string `json:"image"`
		Mangled bool   `json:"mangled"`
	}
	return enc.Encode(&struct {
		Cmdline         []string     `json:"cmdline"`
		Platform        string       `json:"platform"`
		Distro          string       `json:"distro"`
		IgnitionVersion string       `json:"ignitionversion"`
		Board           string       `json:"board"`
		OSContainer     string       `json:"oscontainer"`
		AWS             AWS          `json:"aws"`
		Azure           Azure        `json:"azure"`
		DO              DO           `json:"do"`
		ESX             ESX          `json:"esx"`
		GCE             GCE          `json:"gce"`
		OpenStack       OpenStack    `json:"openstack"`
		EquinixMetal    EquinixMetal `json:"equinixmetal"`
		QEMU            QEMU         `json:"qemu"`
	}{
		Cmdline:         os.Args,
		Platform:        kolaPlatform,
		Distro:          kola.Options.Distribution,
		IgnitionVersion: kola.Options.IgnitionVersion,
		Board:           kola.QEMUOptions.Board,
		OSContainer:     kola.Options.OSContainer,
		AWS: AWS{
			Region:       kola.AWSOptions.Region,
			AMI:          kola.AWSOptions.AMI,
			InstanceType: kola.AWSOptions.InstanceType,
		},
		Azure: Azure{
			DiskURI:   kola.AzureOptions.DiskURI,
			BlobURL:   kola.AzureOptions.BlobURL,
			ImageFile: kola.AzureOptions.ImageFile,
			Publisher: kola.AzureOptions.Publisher,
			Offer:     kola.AzureOptions.Offer,
			Sku:       kola.AzureOptions.Sku,
			Version:   kola.AzureOptions.Version,
			Location:  kola.AzureOptions.Location,
			Size:      kola.AzureOptions.Size,
		},
		DO: DO{
			Region: kola.DOOptions.Region,
			Size:   kola.DOOptions.Size,
			Image:  kola.DOOptions.Image,
		},
		ESX: ESX{
			Server:     kola.ESXOptions.Server,
			BaseVMName: kola.ESXOptions.BaseVMName,
		},
		GCE: GCE{
			Image:       kola.GCEOptions.Image,
			MachineType: kola.GCEOptions.MachineType,
		},
		OpenStack: OpenStack{
			Region: kola.OpenStackOptions.Region,
			Image:  kola.OpenStackOptions.Image,
			Flavor: kola.OpenStackOptions.Flavor,
		},
		EquinixMetal: EquinixMetal{
			Metro:                 kola.EquinixMetalOptions.Metro,
			Plan:                  kola.EquinixMetalOptions.Plan,
			InstallerImageBaseURL: kola.EquinixMetalOptions.InstallerImageBaseURL,
			ImageURL:              kola.EquinixMetalOptions.ImageURL,
		},
		QEMU: QEMU{
			Image:   kola.QEMUOptions.DiskImage,
			Mangled: !kola.QEMUOptions.UseVanillaImage,
		},
	})
}

func runList(cmd *cobra.Command, args []string) {
	tests := register.Tests

	if listFilter {
		var patterns []string
		if len(args) >= 1 {
			patterns = args
		} else {
			patterns = []string{"*"} // run all tests by default
		}
		var err error
		tests, err = kola.FilterTests(register.Tests, patterns, kolaChannel, kolaOffering, kolaPlatform, semver.Version{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "filtering error: %v\n", err)
			os.Exit(1)
		}
	}

	var testlist []*item

	for name, test := range tests {
		item := &item{
			name,
			test.Platforms,
			test.ExcludePlatforms,
			test.Architectures,
			test.Distros,
			test.ExcludeDistros,
			test.Channels,
			test.ExcludeChannels,
			test.Offerings,
			test.ExcludeOfferings,
		}
		item.updateValues()
		testlist = append(testlist, item)
	}

	sort.Slice(testlist, func(i, j int) bool {
		return testlist[i].Name < testlist[j].Name
	})

	if !listJSON {
		var w = tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)

		fmt.Fprintln(w, "Test Name\tPlatforms\tArchitectures\tDistributions\tChannels\tOfferings")
		fmt.Fprintln(w, "\t\t\t\t\t")
		for _, item := range testlist {
			fmt.Fprintf(w, "%v\n", item)
		}
		w.Flush()
	} else {
		out, err := json.MarshalIndent(testlist, "", "\t")
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshalling test list: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(out))
	}
}

type item struct {
	Name             string
	Platforms        []string
	ExcludePlatforms []string `json:"-"`
	Architectures    []string
	Distros          []string
	ExcludeDistros   []string `json:"-"`
	Channels         []string
	ExcludeChannels  []string `json:"-"`
	Offerings        []string
	ExcludeOfferings []string `json:"-"`
}

func (i *item) updateValues() {
	buildItems := func(include, exclude, all []string) []string {
		if len(include) == 0 && len(exclude) == 0 {
			if listJSON {
				return all
			} else {
				return []string{"all"}
			}
		}
		var retItems []string
		if len(exclude) > 0 {
			excludeMap := map[string]struct{}{}
			for _, item := range exclude {
				excludeMap[item] = struct{}{}
			}
			if len(include) == 0 {
				retItems = all
			} else {
				retItems = include
			}
			items := []string{}
			for _, item := range retItems {
				if _, ok := excludeMap[item]; !ok {
					items = append(items, item)
				}
			}
			retItems = items
		} else {
			retItems = include
		}
		return retItems
	}
	i.Platforms = buildItems(i.Platforms, i.ExcludePlatforms, kolaPlatforms)
	i.Architectures = buildItems(i.Architectures, nil, kolaArchitectures)
	i.Distros = buildItems(i.Distros, i.ExcludeDistros, kolaDistros)
	i.Channels = buildItems(i.Channels, i.ExcludeChannels, kolaChannels)
	i.Offerings = buildItems(i.Offerings, i.ExcludeOfferings, kolaOfferings)
}

func (i item) String() string {
	return fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v", i.Name, i.Platforms, i.Architectures, i.Distros, i.Channels, i.Offerings)
}
