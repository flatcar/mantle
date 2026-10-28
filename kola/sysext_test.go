// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package kola_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/register"
	_ "github.com/flatcar/mantle/kola/tests/podman"
	_ "github.com/flatcar/mantle/kola/tests/sysext"
)

type cacheTransport func(*http.Request) (*http.Response, error)

func (f cacheTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSysextFiltering(t *testing.T) {
	oldOptions := kola.Options
	oldBoard := kola.QEMUOptions.Board
	oldTransport := http.DefaultTransport
	t.Cleanup(func() {
		kola.Options = oldOptions
		kola.QEMUOptions.Board = oldBoard
		http.DefaultTransport = oldTransport
	})
	kola.QEMUOptions.Board = "amd64-usr"

	tests := make(map[string]*register.Test)
	for _, name := range []string{
		"podman.base", "podman.workflow", "podman.network-single",
		"sysext.zfs.reboot", "sysext.zfs.docker", "sysext.zfs.nfs",
	} {
		test, ok := register.Tests[name]
		if !ok {
			t.Fatalf("test %q is not registered", name)
		}
		tests[name] = test
	}

	for _, tc := range []struct {
		name         string
		distribution string
		version      semver.Version
		imageVersion string
		secureboot   bool
		status       int
		wantTests    int
		wantProbes   int
		pattern      string
		err          error
	}{
		{"explicit development version", "cl", semver.Version{Major: 4593}, "4593.0.0+developer-build", false, http.StatusOK, 6, 1, "*", nil},
		{"nightly comparison version", "cl", semver.Version{Major: 999999, Minor: 99, Patch: 99}, "4593.0.0+dev-main-nightly-20261028", false, http.StatusOK, 6, 1, "*", nil},
		{"release", "cl", semver.Version{Major: 4593}, "4593.0.0", false, http.StatusOK, 0, 0, "*", nil},
		{"missing artifacts", "cl", semver.Version{Major: 4593}, "4593.0.0+developer-build", false, http.StatusNotFound, 0, 1, "*", nil},
		{"request failure", "cl", semver.Version{Major: 4593}, "4593.0.0+developer-build", false, 0, 0, 1, "*", errors.New("network failure")},
		{"secure boot excludes ZFS", "cl", semver.Version{Major: 4593}, "4593.0.0+developer-build", true, http.StatusOK, 3, 1, "*", nil},
		{"Fedora release excludes Flatcar tests", "fcos", semver.Version{Major: 4593}, "4593.0.0", false, http.StatusOK, 0, 0, "*", nil},
		{"Fedora development excludes Flatcar tests", "fcos", semver.Version{Major: 4593}, "4593.0.0+developer-build", false, http.StatusOK, 0, 0, "*", nil},
		{"unrelated name", "cl", semver.Version{Major: 4593}, "4593.0.0+developer-build", false, http.StatusOK, 0, 0, "unrelated.*", nil},
		{"secure boot ZFS only", "cl", semver.Version{Major: 4593}, "4593.0.0+developer-build", true, http.StatusOK, 0, 0, "sysext.*", nil},
		{"unknown version", "cl", semver.Version{}, "", false, http.StatusOK, 6, 0, "*", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			kola.Options.Distribution = tc.distribution
			kola.Options.EnableSecureboot = tc.secureboot
			probes := 0
			http.DefaultTransport = cacheTransport(func(req *http.Request) (*http.Response, error) {
				probes++
				wantURL := "https://bincache.flatcar-linux.net/images/amd64/" + tc.imageVersion + "/version.txt"
				if req.Method != http.MethodHead || req.URL.String() != wantURL {
					t.Errorf("request = %s %s, want HEAD %s", req.Method, req.URL, wantURL)
				}
				if tc.err != nil {
					return nil, tc.err
				}
				return &http.Response{
					StatusCode: tc.status,
					Status:     http.StatusText(tc.status),
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			})
			for pass := 1; pass <= 2; pass++ {
				filtered, err := kola.FilterTests(tests, []string{tc.pattern}, "alpha", "", "qemu", tc.version, tc.imageVersion)
				if err != nil {
					t.Fatal(err)
				}
				if len(filtered) != tc.wantTests || probes != pass*tc.wantProbes {
					t.Errorf("pass %d: selected %d tests with %d total probes, want %d tests with %d total probes", pass, len(filtered), probes, tc.wantTests, pass*tc.wantProbes)
				}
			}
		})
	}
}
