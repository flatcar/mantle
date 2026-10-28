// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package kola

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
)

func TestParseImageVersion(t *testing.T) {
	for _, imageVersion := range []string{
		"4593.0.0",
		"4593.0.0+dev-main-nightly-20261028",
		"4593.0.0+build_with_underscores",
	} {
		t.Run(imageVersion, func(t *testing.T) {
			version, err := parseImageVersion(imageVersion)
			if err != nil {
				t.Fatal(err)
			}
			if version != (semver.Version{Major: 4593}) {
				t.Fatalf("comparison version = %v, want 4593.0.0", version)
			}
		})
	}
	if _, err := parseImageVersion("invalid+build"); err == nil {
		t.Fatal("expected invalid image version to fail")
	}
}

type versionFlight struct {
	platform.Flight
	cluster platform.Cluster
}

func (f versionFlight) NewCluster(*platform.RuntimeConfig) (platform.Cluster, error) {
	return f.cluster, nil
}

type versionCluster struct {
	platform.Cluster
	machine platform.Machine
}

func (c versionCluster) NewMachine(*conf.UserData) (platform.Machine, error) {
	return c.machine, nil
}

func (c versionCluster) Destroy() {}

type versionMachine struct {
	platform.Machine
	version string
	buildID string
}

func (m versionMachine) SSH(cmd string) ([]byte, []byte, error) {
	switch cmd {
	case `. /etc/os-release && echo "${VERSION}"`:
		return []byte(m.version), nil, nil
	case `. /etc/os-release && echo "${BUILD_ID}"`:
		return []byte(m.buildID), nil, nil
	default:
		return nil, nil, errors.New("unexpected SSH command: " + cmd)
	}
}

func TestGetClusterSemver(t *testing.T) {
	oldOptions := Options
	t.Cleanup(func() { Options = oldOptions })
	Options.Distribution = "cl"

	for _, tc := range []struct {
		buildID string
		want    string
	}{
		{"", "4593.0.0"},
		{"developer-build", "4593.0.0+developer-build"},
		{"dev-main-nightly-20261028", "999999.99.99"},
		{"dev-flatcar-master-20261028", "999999.99.99"},
		{"dev-flatcar-4593-20261028", "4593.99.99"},
		{"dev-flatcar-lts-4081-20261028", "4081.99.99"},
	} {
		t.Run(tc.buildID, func(t *testing.T) {
			imageVersion := "4593.0.0"
			if tc.buildID != "" {
				imageVersion += "+" + tc.buildID
			}
			flight := versionFlight{cluster: versionCluster{
				machine: versionMachine{version: imageVersion, buildID: tc.buildID},
			}}
			version, rawVersion, err := getClusterSemver(flight, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if version.String() != tc.want {
				t.Errorf("comparison version = %s, want %s", version, tc.want)
			}
			if rawVersion != imageVersion {
				t.Errorf("image version = %q, want %q", rawVersion, imageVersion)
			}
		})
	}
}

func TestFilterTestsImageVersion(t *testing.T) {
	oldOptions := Options
	t.Cleanup(func() { Options = oldOptions })
	Options.Distribution = "cl"
	QEMUBoard := QEMUOptions.Board
	QEMUOptions.Board = "amd64-usr"
	t.Cleanup(func() { QEMUOptions.Board = QEMUBoard })

	imageVersion := "4593.0.0+dev-main-nightly-20261028"
	for _, version := range []semver.Version{
		{Major: 4593},
		{Major: 999999, Minor: 99, Patch: 99},
	} {
		t.Run(version.String(), func(t *testing.T) {
			called := 0
			skip := func(got semver.Version, rawVersion, channel, arch, platform string) bool {
				called++
				if got != version || rawVersion != imageVersion {
					t.Errorf("skip versions = (%v, %q), want (%v, %q)", got, rawVersion, version, imageVersion)
				}
				if channel != "alpha" || arch != "amd64" || platform != "qemu" {
					t.Errorf("unexpected skip context: %q, %q, %q", channel, arch, platform)
				}
				return false
			}
			tests := map[string]*register.Test{
				"test.allowed": {Name: "test.allowed", MinVersion: version, SkipFunc: skip},
				"test.too-old": {Name: "test.too-old", EndVersion: version, SkipFunc: skip},
				"test.too-new": {Name: "test.too-new", MinVersion: semver.Version{Major: version.Major + 1}, SkipFunc: skip},
				"test.skipped": {
					Name: "test.skipped",
					SkipFunc: func(semver.Version, string, string, string, string) bool {
						return true
					},
				},
			}
			filtered, err := FilterTests(tests, []string{"test.*"}, "alpha", "", "qemu", version, imageVersion)
			if err != nil {
				t.Fatal(err)
			}
			if called != 1 || len(filtered) != 1 || filtered["test.allowed"] == nil {
				t.Fatalf("skip calls = %d, filtered tests = %v", called, filtered)
			}
		})
	}
}

func TestFilterTestsExactNames(t *testing.T) {
	for _, tc := range []struct {
		name      string
		version   semver.Version
		skip      bool
		wantCalls int
		wantTests int
	}{
		{"unknown version bypasses skip", semver.Version{}, true, 0, 1},
		{"exact name bypasses version range", semver.Version{Major: 4593}, false, 1, 1},
		{"known version still checks skip", semver.Version{Major: 4593}, true, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := 0
			test := &register.Test{
				Name:       "test.exact",
				MinVersion: semver.Version{Major: 999999},
				SkipFunc: func(semver.Version, string, string, string, string) bool {
					called++
					return tc.skip
				},
			}
			filtered, err := FilterTests(map[string]*register.Test{test.Name: test}, []string{test.Name}, "", "", "qemu", tc.version, "")
			if err != nil {
				t.Fatal(err)
			}
			if called != tc.wantCalls || len(filtered) != tc.wantTests {
				t.Errorf("skip calls = %d, selected tests = %d, want %d calls and %d tests", called, len(filtered), tc.wantCalls, tc.wantTests)
			}
		})
	}
}

func TestFilterTestsCheapFilters(t *testing.T) {
	oldOptions := Options
	oldBoard := QEMUOptions.Board
	t.Cleanup(func() {
		Options = oldOptions
		QEMUOptions.Board = oldBoard
	})
	Options.Distribution = "cl"
	QEMUOptions.Board = "amd64-usr"

	for _, tc := range []struct {
		name    string
		pattern string
		test    register.Test
		wantErr bool
	}{
		{name: "name", pattern: "other.*"},
		{name: "invalid pattern", pattern: "[", wantErr: true},
		{name: "minimum version", test: register.Test{MinVersion: semver.Version{Major: 5000}}},
		{name: "end version", test: register.Test{EndVersion: semver.Version{Major: 4593}}},
		{name: "platform", test: register.Test{Platforms: []string{"aws"}}},
		{name: "excluded platform", test: register.Test{ExcludePlatforms: []string{"qemu"}}},
		{name: "architecture", test: register.Test{Architectures: []string{"arm64"}}},
		{name: "distribution", test: register.Test{Distros: []string{"fcos"}}},
		{name: "excluded distribution", test: register.Test{ExcludeDistros: []string{"cl"}}},
		{name: "channel", test: register.Test{Channels: []string{"stable"}}},
		{name: "excluded channel", test: register.Test{ExcludeChannels: []string{"alpha"}}},
		{name: "offering", test: register.Test{Offerings: []string{"other"}}},
		{name: "excluded offering", test: register.Test{ExcludeOfferings: []string{"basic"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			test := tc.test
			test.Name = "test.filtered"
			test.SkipFunc = func(semver.Version, string, string, string, string) bool {
				t.Error("skip function called for a filtered test")
				return true
			}
			pattern := tc.pattern
			if pattern == "" {
				pattern = "test.*"
			}
			filtered, err := FilterTests(map[string]*register.Test{test.Name: &test}, []string{pattern}, "alpha", "basic", "qemu", semver.Version{Major: 4593}, "4593.0.0+developer-build")
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, want error = %t", err, tc.wantErr)
			}
			if len(filtered) != 0 {
				t.Errorf("selected %d tests, want none", len(filtered))
			}
		})
	}
}

type sysextTransport func(*http.Request) (*http.Response, error)

func (f sysextTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type sysextBody struct {
	io.Reader
	closed bool
}

func (b *sysextBody) Close() error {
	b.closed = true
	return nil
}

func TestSkipUnavailableSysext(t *testing.T) {
	oldOptions := Options
	oldTransport := http.DefaultTransport
	t.Cleanup(func() {
		Options = oldOptions
		http.DefaultTransport = oldTransport
	})

	for _, tc := range []struct {
		name         string
		distribution string
		version      string
		status       int
		err          error
		want         bool
		probe        bool
	}{
		{name: "unknown", distribution: "cl", want: true},
		{name: "release", distribution: "cl", version: "4593.0.0", want: true},
		{name: "empty metadata", distribution: "cl", version: "4593.0.0+", want: true},
		{name: "available", distribution: "cl", version: "4593.0.0+dev-main-nightly-20261028", status: http.StatusOK, probe: true},
		{name: "unavailable", distribution: "cl", version: "4593.0.0+dev-main-nightly-20261028", status: http.StatusNotFound, want: true, probe: true},
		{name: "request failure", distribution: "cl", version: "4593.0.0+dev-main-nightly-20261028", err: errors.New("network failure"), want: true, probe: true},
		{name: "non-Flatcar release", distribution: "fcos", version: "4593.0.0"},
		{name: "non-Flatcar development", distribution: "fcos", version: "4593.0.0+developer-build"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			Options.Distribution = tc.distribution
			called := false
			body := &sysextBody{Reader: strings.NewReader("")}
			http.DefaultTransport = sysextTransport(func(req *http.Request) (*http.Response, error) {
				called = true
				wantURL := "https://bincache.flatcar-linux.net/images/arm64/" + tc.version + "/version.txt"
				if req.Method != http.MethodHead || req.URL.String() != wantURL {
					t.Errorf("request = %s %s, want HEAD %s", req.Method, req.URL, wantURL)
				}
				if tc.err != nil {
					return nil, tc.err
				}
				return &http.Response{StatusCode: tc.status, Status: http.StatusText(tc.status), Body: body}, nil
			})
			if got := SkipUnavailableSysext(tc.version, "arm64"); got != tc.want {
				t.Errorf("skip = %t, want %t", got, tc.want)
			}
			if called != tc.probe {
				t.Errorf("cache probed = %t, want %t", called, tc.probe)
			}
			if called && tc.err == nil && !body.closed {
				t.Error("response body was not closed")
			}
		})
	}
}
