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

package register

import (
	"fmt"

	"github.com/coreos/go-semver/semver"

	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/platform/conf"
)

type Flag int

const (
	NoSSHKeyInUserData      Flag = iota // don't inject SSH key into Ignition/cloud-config
	NoSSHKeyInMetadata                  // don't add SSH key to platform metadata
	NoEmergencyShellCheck               // don't check console output for emergency shell invocation
	NoEnableSelinux                     // don't enable selinux
	NoKernelPanicCheck                  // don't check console output for kernel panic
	NoVerityCorruptionCheck             // don't check console output for verity corruption
	NoDisableUpdates                    // don't disable usage of the public update server
)

// Test provides the main test abstraction for kola. The run function is
// the actual testing function while the other fields provide ways to
// statically declare state of the platform.TestCluster before the test
// function is run.
type Test struct {
	Name             string // should be unique
	Run              func(cluster.TestCluster)
	NativeFuncs      map[string]func() error
	UserData         *conf.UserData
	UserDataV3       *conf.UserData
	ClusterSize      int
	Platforms        []string // whitelist of platforms to run test against -- defaults to all
	ExcludePlatforms []string // blacklist of platforms to ignore -- defaults to none
	Distros          []string // whitelist of distributions to run test against -- defaults to all
	ExcludeDistros   []string // blacklist of distributions to ignore -- defaults to none
	Channels         []string // whitelist of channels to run test against -- defaults to all
	ExcludeChannels  []string // blacklist of channels to ignore -- defaults to none
	Offerings        []string // whitelist of offerings to run the test against -- defaults to all
	ExcludeOfferings []string // blacklist of offerings to ignore -- defaults to none
	Architectures    []string // whitelist of machine architectures supported -- defaults to all
	Flags            []Flag   // special-case options for this test

	// FailFast skips any sub-test that occurs after a sub-test has
	// failed.
	FailFast bool

	// MinVersion prevents the test from executing on CoreOS machines
	// less than MinVersion. This will be ignored if the name fully
	// matches without globbing.
	MinVersion semver.Version

	// EndVersion prevents the test from executing on CoreOS machines
	// greater than or equal to EndVersion. This will be ignored if
	// the name fully matches without globbing.
	EndVersion semver.Version

	// SkipFunc can be used to define if a test should be skip or not based on some
	// condition on the version, channel, arch and platform.
	SkipFunc func(version semver.Version, channel, arch, platform string) bool

	// DefaultUser is the user used for SSH connection, it will be created via Ignition when possible.
	DefaultUser string
}

// Registered tests live here. Mapping of names to tests.
var Tests = map[string]*Test{}

// Register is usually called in init() functions and is how kola test
// harnesses knows which tests it can choose from. Panics if existing
// name is registered
func Register(t *Test) {
	_, ok := Tests[t.Name]
	if ok {
		panic(fmt.Sprintf("test %v already registered", t.Name))
	}

	if (t.EndVersion != semver.Version{}) && !t.MinVersion.LessThan(t.EndVersion) {
		panic(fmt.Sprintf("test %v has an invalid version range", t.Name))
	}

	Tests[t.Name] = t
}

func (t *Test) HasFlag(flag Flag) bool {
	for _, f := range t.Flags {
		if f == flag {
			return true
		}
	}
	return false
}
