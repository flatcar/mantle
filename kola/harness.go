// Copyright 2021 Kinvolk GmbH
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

package kola

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/agent"

	"github.com/coreos/go-semver/semver"
	"github.com/coreos/pkg/capnslog"

	"github.com/flatcar-linux/mantle/harness"
	"github.com/flatcar-linux/mantle/harness/reporters"
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/kola/torcx"
	"github.com/flatcar-linux/mantle/platform"
	awsapi "github.com/flatcar-linux/mantle/platform/api/aws"
	azureapi "github.com/flatcar-linux/mantle/platform/api/azure"
	doapi "github.com/flatcar-linux/mantle/platform/api/do"
	esxapi "github.com/flatcar-linux/mantle/platform/api/esx"
	gcloudapi "github.com/flatcar-linux/mantle/platform/api/gcloud"
	openstackapi "github.com/flatcar-linux/mantle/platform/api/openstack"
	packetapi "github.com/flatcar-linux/mantle/platform/api/packet"
	"github.com/flatcar-linux/mantle/platform/conf"
	"github.com/flatcar-linux/mantle/platform/machine/aws"
	"github.com/flatcar-linux/mantle/platform/machine/azure"
	"github.com/flatcar-linux/mantle/platform/machine/do"
	"github.com/flatcar-linux/mantle/platform/machine/esx"
	"github.com/flatcar-linux/mantle/platform/machine/external"
	"github.com/flatcar-linux/mantle/platform/machine/gcloud"
	"github.com/flatcar-linux/mantle/platform/machine/openstack"
	"github.com/flatcar-linux/mantle/platform/machine/packet"
	"github.com/flatcar-linux/mantle/platform/machine/qemu"
	"github.com/flatcar-linux/mantle/platform/machine/unprivqemu"
	"github.com/flatcar-linux/mantle/system"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "kola")

	Options          = platform.Options{}
	AWSOptions       = awsapi.Options{Options: &Options}       // glue to set platform options from main
	AzureOptions     = azureapi.Options{Options: &Options}     // glue to set platform options from main
	DOOptions        = doapi.Options{Options: &Options}        // glue to set platform options from main
	ESXOptions       = esxapi.Options{Options: &Options}       // glue to set platform options from main
	ExternalOptions  = external.Options{Options: &Options}     // glue to set platform options from main
	GCEOptions       = gcloudapi.Options{Options: &Options}    // glue to set platform options from main
	OpenStackOptions = openstackapi.Options{Options: &Options} // glue to set platform options from main
	PacketOptions    = packetapi.Options{Options: &Options}    // glue to set platform options from main
	QEMUOptions      = qemu.Options{Options: &Options}         // glue to set platform options from main

	TestParallelism   int    //glue var to set test parallelism from main
	TAPFile           string // if not "", write TAP results here
	TorcxManifestFile string // torcx manifest to expose to tests, if set
	// TorcxManifest is the unmarshalled torcx manifest file. It is available for
	// tests to access via `kola.TorcxManifest`. It will be nil if there was no
	// manifest given to kola.
	TorcxManifest *torcx.Manifest = nil

	UpdatePayloadFile string

	consoleChecks = []struct {
		desc        string
		match       *regexp.Regexp
		skipIfMatch *regexp.Regexp
		skipFlag    *register.Flag
	}{
		{
			desc:     "emergency shell",
			match:    regexp.MustCompile("Press Enter for emergency shell|Starting Emergency Shell|You are in emergency mode"),
			skipFlag: &[]register.Flag{register.NoEmergencyShellCheck}[0],
		},
		{
			desc:     "kernel panic",
			match:    regexp.MustCompile("Kernel panic - not syncing: (.*)"),
			skipFlag: &[]register.Flag{register.NoKernelPanicCheck}[0],
		},
		{
			desc:  "kernel oops",
			match: regexp.MustCompile("Oops:"),
		},
		{
			desc:  "kernel warning",
			match: regexp.MustCompile(`WARNING: CPU: \d+ PID: \d+ at (.+)`),
		},
		{
			desc:  "failure of disk under I/O",
			match: regexp.MustCompile("rejecting I/O to offline device"),
		},
		{
			// Failure to set up Packet networking in initramfs,
			// perhaps due to unresponsive metadata server
			desc:  "coreos-metadata failure to set up initramfs network",
			match: regexp.MustCompile("Failed to start CoreOS Static Network Agent"),
		},
		{
			// https://github.com/coreos/bugs/issues/2065
			desc:        "excessive bonding link status messages",
			match:       regexp.MustCompile("(?s:link status up for interface [^,]+, enabling it in [0-9]+ ms.*?){3}"),
			skipIfMatch: regexp.MustCompile("(bond.*? link status definitely up for interface)|(bond.*? first active interface up)|(bond.*? Gained carrier)|(bond.*? link becomes ready)"),
		},
		{
			// https://github.com/coreos/bugs/issues/2180
			desc:  "ext4 delayed allocation failure",
			match: regexp.MustCompile(`EXT4-fs \([^)]+\): Delayed block allocation failed for inode \d+ at logical offset \d+ with max blocks \d+ with (error \d+)`),
		},
		{
			// https://github.com/coreos/bugs/issues/2284
			desc:  "GRUB memory corruption",
			match: regexp.MustCompile("((alloc|free) magic) (is )?broken"),
		},
		{
			// https://github.com/coreos/bugs/issues/2435
			desc:  "Ignition fetch cancellation race",
			match: regexp.MustCompile("ignition\\[[0-9]+\\]: failed to fetch config: context canceled"),
		},
		{
			// https://github.com/coreos/bugs/issues/2526
			desc:  "initrd-cleanup.service terminated",
			match: regexp.MustCompile("initrd-cleanup\\.service: Main process exited, code=killed, status=15/TERM"),
		},
		{
			// kernel 4.14.11
			desc:  "bad page table",
			match: regexp.MustCompile("mm/pgtable-generic.c:\\d+: bad (p.d|pte)"),
		},
		{
			desc:  "Go panic",
			match: regexp.MustCompile("panic: (.*)"),
		},
		{
			desc:  "segfault",
			match: regexp.MustCompile("SIGSEGV|=11/SEGV"),
		},
		{
			desc:  "core dump",
			match: regexp.MustCompile("[Cc]ore dump"),
		},
	}
)

// NativeRunner is a closure passed to all kola test functions and used
// to run native go functions directly on kola machines. It is necessary
// glue until kola does introspection.
type NativeRunner func(funcName string, m platform.Machine) error

func NewFlight(pltfrm string) (flight platform.Flight, err error) {
	switch pltfrm {
	case "aws":
		flight, err = aws.NewFlight(&AWSOptions)
	case "azure":
		flight, err = azure.NewFlight(&AzureOptions)
	case "do":
		flight, err = do.NewFlight(&DOOptions)
	case "esx":
		flight, err = esx.NewFlight(&ESXOptions)
	case "external":
		flight, err = external.NewFlight(&ExternalOptions)
	case "gce":
		flight, err = gcloud.NewFlight(&GCEOptions)
	case "openstack":
		flight, err = openstack.NewFlight(&OpenStackOptions)
	case "packet":
		flight, err = packet.NewFlight(&PacketOptions)
	case "qemu":
		flight, err = qemu.NewFlight(&QEMUOptions)
	case "qemu-unpriv":
		flight, err = unprivqemu.NewFlight(&QEMUOptions)
	default:
		err = fmt.Errorf("invalid platform %q", pltfrm)
	}
	return
}

func FilterTests(tests map[string]*register.Test, patterns []string, channel, offering string, pltfrm string, version semver.Version) (map[string]*register.Test, error) {
	r := make(map[string]*register.Test)

	checkPlatforms := []string{pltfrm}

	// qemu-unpriv has the same restrictions as QEMU but might also want additional restrictions due to the lack of a Local cluster
	if pltfrm == "qemu-unpriv" {
		checkPlatforms = append(checkPlatforms, "qemu")
	}

	for name, t := range tests {
		noMatch := true
		for _, pattern := range patterns {
			match, err := filepath.Match(pattern, t.Name)
			if err != nil {
				return nil, err
			}
			if match {
				noMatch = false
				break
			}
		}
		if noMatch {
			continue
		}
		patternNotName := true
		for _, pattern := range patterns {
			if t.Name == pattern {
				patternNotName = false
				break
			}
		}

		// Check the test's min and end versions when running more than one test
		if patternNotName && versionOutsideRange(version, t.MinVersion, t.EndVersion) {
			continue
		}

		isAllowed := func(item string, include, exclude []string) (bool, bool) {
			allowed, excluded := true, false
			for _, i := range include {
				if i == item {
					allowed = true
					break
				} else {
					allowed = false
				}
			}
			for _, i := range exclude {
				if i == item {
					allowed = false
					excluded = true
				}
			}
			return allowed, excluded
		}

		isExcluded := false
		allowed := false
		for _, platform := range checkPlatforms {
			allowedPlatform, excluded := isAllowed(platform, t.Platforms, t.ExcludePlatforms)
			if excluded {
				isExcluded = true
				break
			}
			allowedArchitecture, _ := isAllowed(architecture(platform), t.Architectures, []string{})
			allowed = allowed || (allowedPlatform && allowedArchitecture)
		}
		if isExcluded || !allowed {
			continue
		}

		if allowed, excluded := isAllowed(Options.Distribution, t.Distros, t.ExcludeDistros); !allowed || excluded {
			continue
		}

		if allowed, excluded := isAllowed(channel, t.Channels, t.ExcludeChannels); !allowed || excluded {
			continue
		}

		if allowed, excluded := isAllowed(offering, t.Offerings, t.ExcludeOfferings); !allowed || excluded {
			continue
		}

		r[name] = t
	}

	return r, nil
}

// versionOutsideRange checks to see if version is outside [min, end). If end
// is a zero value, it is ignored and there is no upper bound. If version is a
// zero value, the bounds are ignored.
func versionOutsideRange(version, minVersion, endVersion semver.Version) bool {
	if version == (semver.Version{}) {
		return false
	}

	if version.LessThan(minVersion) {
		return true
	}

	if (endVersion != semver.Version{}) && !version.LessThan(endVersion) {
		return true
	}

	return false
}

// RunTests is a harness for running multiple tests in parallel. Filters
// tests based on glob patterns and by platform. Has access to all
// tests either registered in this package or by imported packages that
// register tests in their init() function.
// outputDir is where various test logs and data will be written for
// analysis after the test run. If it already exists it will be erased!
func RunTests(patterns []string, channel, offering, pltfrm, outputDir string, sshKeys *[]agent.Key, remove bool) error {
	var versionStr string

	// Avoid incurring cost of starting machine in getClusterSemver when
	// either:
	// 1) none of the selected tests care about the version
	// 2) glob is an exact match which means minVersion will be ignored
	//    either way
	// 3) the provided torcx flag is wrong
	tests, err := FilterTests(register.Tests, patterns, channel, offering, pltfrm, semver.Version{})
	if err != nil {
		plog.Fatal(err)
	}

	skipGetVersion := true
	for name, t := range tests {
		patternNotName := true
		for _, pattern := range patterns {
			if name == pattern {
				patternNotName = false
				break
			}
		}
		if patternNotName && (t.MinVersion != semver.Version{} || t.EndVersion != semver.Version{}) {
			skipGetVersion = false
			break
		}
	}

	if TorcxManifestFile != "" {
		TorcxManifest = &torcx.Manifest{}
		torcxManifestFile, err := os.Open(TorcxManifestFile)
		if err != nil {
			return errors.New("Torcx manifest path provided could not be read")
		}
		if err := json.NewDecoder(torcxManifestFile).Decode(TorcxManifest); err != nil {
			return fmt.Errorf("could not parse torcx manifest as valid json: %v", err)
		}
		torcxManifestFile.Close()
	}

	flight, err := NewFlight(pltfrm)
	if err != nil {
		plog.Fatalf("creating flight for RunTests failed: %v", err)
	}
	(*flight.GetBaseFlight()).AdditionalSshKeys = sshKeys
	if remove {
		defer flight.Destroy()
	}

	if !skipGetVersion {
		plog.Info("Creating cluster to check semver...")
		version, err := getClusterSemver(flight, outputDir)
		if err != nil {
			plog.Fatal(err)
		}

		versionStr = version.String()

		// one more filter pass now that we know real version
		tests, err = FilterTests(tests, patterns, channel, offering, pltfrm, *version)
		if err != nil {
			plog.Fatal(err)
		}
	}

	opts := harness.Options{
		OutputDir: outputDir,
		Parallel:  TestParallelism,
		Verbose:   true,
		Reporters: reporters.Reporters{
			reporters.NewJSONReporter("report.json", pltfrm, versionStr),
		},
	}
	var htests harness.Tests
	for _, test := range tests {
		test := test // for the closure
		run := func(h *harness.H) {
			runTest(h, test, pltfrm, flight, remove)
		}
		htests.Add(test.Name, run)
	}

	suite := harness.NewSuite(opts, htests)
	err = suite.Run()

	if TAPFile != "" {
		src := filepath.Join(outputDir, "test.tap")
		if err2 := system.CopyRegularFile(src, TAPFile); err == nil && err2 != nil {
			err = err2
		}
	}

	if err != nil {
		fmt.Printf("FAIL, output in %v\n", outputDir)
	} else {
		fmt.Printf("PASS, output in %v\n", outputDir)
	}

	return err
}

// getClusterSemVer returns the CoreOS semantic version via starting a
// machine and checking
func getClusterSemver(flight platform.Flight, outputDir string) (*semver.Version, error) {
	var err error

	testDir := filepath.Join(outputDir, "get_cluster_semver")
	if err := os.MkdirAll(testDir, 0777); err != nil {
		return nil, err
	}

	cluster, err := flight.NewCluster(&platform.RuntimeConfig{
		OutputDir: testDir,
	})
	if err != nil {
		return nil, fmt.Errorf("creating cluster for semver check: %v", err)
	}
	defer cluster.Destroy()

	m, err := cluster.NewMachine(nil)
	if err != nil {
		return nil, fmt.Errorf("creating new machine for semver check: %v", err)
	}

	out, stderr, err := m.SSH("grep ^VERSION_ID= /etc/os-release")
	if err != nil {
		return nil, fmt.Errorf("parsing /etc/os-release for VERSION_ID: %v: %s", err, stderr)
	}
	ver := strings.Split(string(out), "=")[1]

	out, stderr, err = m.SSH("grep ^BUILD_ID= /etc/os-release")
	if err != nil {
		return nil, fmt.Errorf("parsing /etc/os-release for BUILD_ID: %v: %s", err, stderr)
	}
	build_id := strings.Split(string(out), "=")[1]
	if strings.HasPrefix(build_id, "dev-main-nightly-") || strings.HasPrefix(build_id, "dev-flatcar-master-") {
		// "main" is a nightly build of the main branch,
		// "flatcar-master" refers to the manifest branch where dev builds are started
		ver = "999999.99.99"
	} else if strings.HasPrefix(build_id, "dev-flatcar-") {
		// flatcar-MAJOR is a nightly build of the release branch
		parts := strings.Split(build_id, "-")
		major := parts[2]
		if major == "lts" {
			major = parts[3]
		}
		ver = major + ".99.99"
	}
	plog.Noticef("Using %q as version to filter tests...", ver)

	// TODO: add distro specific version handling
	switch Options.Distribution {
	case "cl":
		return parseCLVersion(ver)
	case "rhcos":
		return &semver.Version{}, nil
	}

	return nil, fmt.Errorf("no case to handle version parsing for distribution %q", Options.Distribution)
}

func parseCLVersion(input string) (*semver.Version, error) {
	version, err := semver.NewVersion(input)
	if err != nil {
		return nil, fmt.Errorf("parsing os-release semver: %v", err)
	}

	return version, nil
}

// runTest is a harness for running a single test.
// outputDir is where various test logs and data will be written for
// analysis after the test run. It should already exist.
func runTest(h *harness.H, t *register.Test, pltfrm string, flight platform.Flight, remove bool) {
	h.Parallel()

	rconf := &platform.RuntimeConfig{
		OutputDir:          h.OutputDir(),
		NoSSHKeyInUserData: t.HasFlag(register.NoSSHKeyInUserData),
		NoSSHKeyInMetadata: t.HasFlag(register.NoSSHKeyInMetadata),
		NoEnableSelinux:    t.HasFlag(register.NoEnableSelinux),
	}
	c, err := flight.NewCluster(rconf)
	if err != nil {
		h.Fatalf("Cluster failed: %v", err)
	}
	defer func() {
		if remove {
			c.Destroy()
		}
		for id, output := range c.ConsoleOutput() {
			for _, badness := range CheckConsole([]byte(output), t) {
				h.Errorf("Found %s on machine %s console", badness, id)
			}
		}
		for id, output := range c.JournalOutput() {
			for _, badness := range CheckConsole([]byte(output), t) {
				h.Errorf("Found %s on machine %s journal", badness, id)
			}
		}
	}()

	if t.ClusterSize > 0 {
		var userdata *conf.UserData
		if Options.IgnitionVersion == "v2" {
			userdata = t.UserData
		} else if Options.IgnitionVersion == "v3" {
			userdata = t.UserDataV3
		}
		if userdata != nil && userdata.Contains("$discovery") {
			url, err := c.GetDiscoveryURL(t.ClusterSize)
			if err != nil {
				// Skip instead of failing since the harness not being able to
				// get a discovery url is likely an outage (e.g
				// 503 Service Unavailable: Back-end server is at capacity)
				// not a problem with the OS
				h.Skipf("Failed to create discovery endpoint: %v", err)
			}
			userdata = userdata.Subst("$discovery", url)
		}

		if _, err := platform.NewMachines(c, userdata, t.ClusterSize); err != nil {
			h.Fatalf("Cluster failed starting machines: %v", err)
		}
	}

	// pass along all registered native functions
	var names []string
	for k := range t.NativeFuncs {
		names = append(names, k)
	}

	// Cluster -> TestCluster
	tcluster := cluster.TestCluster{
		H:           h,
		Cluster:     c,
		NativeFuncs: names,
		FailFast:    t.FailFast,
	}

	// drop kolet binary on machines
	if t.NativeFuncs != nil {
		scpKolet(tcluster, architecture(pltfrm))
	}

	defer func() {
		// give some time for the remote journal to be flushed so it can be read
		// before we run the deferred machine destruction
		time.Sleep(2 * time.Second)
	}()

	// run test
	t.Run(tcluster)
}

// architecture returns the machine architecture of the given platform.
func architecture(pltfrm string) string {
	nativeArch := "amd64"
	if pltfrm == "qemu" && QEMUOptions.Board != "" {
		nativeArch = boardToArch(QEMUOptions.Board)
	}
	if pltfrm == "packet" && PacketOptions.Board != "" {
		nativeArch = boardToArch(PacketOptions.Board)
	}
	return nativeArch
}

// returns the arch part of an sdk board name
func boardToArch(board string) string {
	return strings.SplitN(board, "-", 2)[0]
}

// scpKolet searches for a kolet binary and copies it to the machine.
func scpKolet(c cluster.TestCluster, mArch string) {
	for _, d := range []string{
		".",
		filepath.Dir(os.Args[0]),
		filepath.Join(filepath.Dir(os.Args[0]), mArch),
		filepath.Join("/usr/lib/kola", mArch),
	} {
		kolet := filepath.Join(d, "kolet")
		if _, err := os.Stat(kolet); err == nil {
			if err := c.DropFile(kolet); err != nil {
				c.Fatalf("dropping kolet binary: %v", err)
			}
			// The default SELinux rules do not allow init_t to execute user_home_t
			if Options.Distribution == "rhcos" || Options.Distribution == "fcos" {
				for _, machine := range c.Machines() {
					out, stderr, err := machine.SSH("sudo chcon -t bin_t kolet")
					if err != nil {
						c.Fatalf("running chcon on kolet: %s: %s: %v", out, stderr, err)
					}
				}
			}
			return
		}
	}
	c.Fatalf("Unable to locate kolet binary for %s", mArch)
}

// CheckConsole checks some console output for badness and returns short
// descriptions of any badness it finds. If t is specified, its flags are
// respected.
func CheckConsole(output []byte, t *register.Test) []string {
	var ret []string
	for _, check := range consoleChecks {
		if check.skipFlag != nil && t != nil && t.HasFlag(*check.skipFlag) {
			continue
		}
		match := check.match.FindSubmatch(output)
		if match != nil {
			if check.skipIfMatch != nil {
				skipMatch := check.skipIfMatch.FindSubmatch(output)
				if skipMatch != nil {
					continue
				}
			}
			badness := check.desc
			if len(match) > 1 {
				// include first subexpression
				badness += fmt.Sprintf(" (%s)", match[1])
			}
			ret = append(ret, badness)
		}
	}
	return ret
}

func SetupOutputDir(outputDir, platform string) (string, error) {
	defaulted := outputDir == ""
	defaultBaseDirName := "_kola_temp"
	defaultDirName := fmt.Sprintf("%s-%s-%d", platform, time.Now().Format("2006-01-02-1504"), os.Getpid())

	if defaulted {
		if _, err := os.Stat(defaultBaseDirName); os.IsNotExist(err) {
			if err := os.Mkdir(defaultBaseDirName, 0777); err != nil {
				return "", err
			}
		}
		outputDir = filepath.Join(defaultBaseDirName, defaultDirName)
	}

	outputDir, err := harness.CleanOutputDir(outputDir)
	if err != nil {
		return "", err
	}

	if defaulted {
		tempLinkPath := filepath.Join(outputDir, "latest")
		linkPath := filepath.Join(defaultBaseDirName, platform+"-latest")
		// don't clobber existing files that are not symlinks
		st, err := os.Lstat(linkPath)
		if err == nil && (st.Mode()&os.ModeType) != os.ModeSymlink {
			return "", fmt.Errorf("%v exists and is not a symlink", linkPath)
		} else if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err := os.Symlink(defaultDirName, tempLinkPath); err != nil {
			return "", err
		}
		// atomic rename
		if err := os.Rename(tempLinkPath, linkPath); err != nil {
			os.Remove(tempLinkPath)
			return "", err
		}
	}

	return outputDir, nil
}
