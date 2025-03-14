// Copyright 2023 Flatcar Maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
)

const (
	versionsURL        = "https://docs.nvidia.com/datacenter/tesla/drivers/releases.json"
	bakeryReleasesURL  = "https://api.github.com/repos/flatcar/sysext-bakery/releases/latest"
	gpuOperatorTagsURL = "https://api.github.com/repos/NVIDIA/gpu-operator/tags"
	outputFile         = "nvidia_versions.go"
)

type DriverInfo struct {
	ReleaseVersion string   `json:"release_version"`
	ReleaseDate    string   `json:"release_date"`
	Architectures  []string `json:"architectures"`
}

type BranchInfo struct {
	Type       string       `json:"type"`
	DriverInfo []DriverInfo `json:"driver_info"`
}

// Extract major version from version string (e.g., "570.124.06" -> "570")
func getMajorVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return version
}

type asset struct {
	Name string `json:"name"`
}
type release struct {
	Assets []asset `json:"assets"`
}

type bakeryVersions struct {
	nvidiaRuntimeVersion string
	kubernetesVersion    string
}

func extractVersion(sysextName string) (string, semver.Version, error) {
	parts := strings.Split(sysextName, "-")
	if len(parts) < 2 {
		return "", semver.Version{}, fmt.Errorf("invalid sysext name: %s", sysextName)
	}
	versionStr := parts[1]
	versionStripped := strings.TrimPrefix(versionStr, "v")
	version, err := semver.NewVersion(versionStripped)
	if err != nil {
		return "", semver.Version{}, fmt.Errorf("failed to parse version %s: %w", versionStr, err)
	}
	return versionStr, *version, nil
}

func fetchLatestBakeryVersions() (bakeryVersions, error) {
	var versions bakeryVersions
	resp, err := http.Get(bakeryReleasesURL)
	if err != nil {
		return versions, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return versions, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return versions, err
	}

	nvidiaRuntimeSemver := semver.Version{}
	kubernetesSemver := semver.Version{}
	for _, asset := range release.Assets {
		if !strings.HasSuffix(asset.Name, ".raw") {
			continue
		}
		if strings.HasPrefix(asset.Name, "nvidia_runtime") {
			version, semver, err := extractVersion(asset.Name)
			if err != nil {
				return versions, err
			}
			fmt.Printf("Found nvidia_runtime version: %s\n", version)
			if nvidiaRuntimeSemver.LessThan(semver) {
				nvidiaRuntimeSemver = semver
				versions.nvidiaRuntimeVersion = version
			}
		} else if strings.HasPrefix(asset.Name, "kubernetes") {
			version, semver, err := extractVersion(asset.Name)
			if err != nil {
				return versions, err
			}
			fmt.Printf("Found kubernetes version: %s\n", version)
			if kubernetesSemver.LessThan(semver) {
				kubernetesSemver = semver
				versions.kubernetesVersion = version
			}
		}
	}

	return versions, nil
}

func fetchLatestGpuOperatorVersion() (string, error) {
	resp, err := http.Get(gpuOperatorTagsURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return "", err
	}

	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found")
	}

	return tags[0].Name, nil
}

func main() {
	// Fetch the releases JSON
	resp, err := http.Get(versionsURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching NVIDIA driver versions: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Unexpected status code: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		os.Exit(1)
	}

	// Parse the JSON
	var releasesMap map[string]BranchInfo
	if err := json.Unmarshal(body, &releasesMap); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	// Set cutoff date to January 1, 2024
	cutoffDate, _ := time.Parse("2006-01-02", "2024-01-01")

	// Map to store versions by major version number
	majorVersionMap := make(map[string][]DriverInfo)

	// Group versions by major version number
	for _, branchInfo := range releasesMap {
		for _, driverInfo := range branchInfo.DriverInfo {
			// Parse the release date
			releaseDate, err := time.Parse("2006-01-02", driverInfo.ReleaseDate)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not parse date '%s' for version '%s': %v\n",
					driverInfo.ReleaseDate, driverInfo.ReleaseVersion, err)
				continue
			}

			// Only process versions released in 2024 or later
			if releaseDate.Before(cutoffDate) {
				fmt.Printf("Skipping version %s (released on %s before 2024)\n",
					driverInfo.ReleaseVersion, driverInfo.ReleaseDate)
				continue
			}

			// Group by major version
			majorVersion := getMajorVersion(driverInfo.ReleaseVersion)
			majorVersionMap[majorVersion] = append(majorVersionMap[majorVersion], driverInfo)
		}
	}

	// Sort each major version group by release date (newest first)
	for majorVersion, versions := range majorVersionMap {
		sort.Slice(versions, func(i, j int) bool {
			dateI, _ := time.Parse("2006-01-02", versions[i].ReleaseDate)
			dateJ, _ := time.Parse("2006-01-02", versions[j].ReleaseDate)
			return dateI.After(dateJ)
		})
		majorVersionMap[majorVersion] = versions
	}

	// Get a list of all major versions and sort them (newest first)
	majorVersions := make([]string, 0, len(majorVersionMap))
	for majorVersion := range majorVersionMap {
		majorVersions = append(majorVersions, majorVersion)
	}
	sort.Slice(majorVersions, func(i, j int) bool {
		// Convert to integer for proper numeric comparison
		numI := 0
		numJ := 0
		fmt.Sscanf(majorVersions[i], "%d", &numI)
		fmt.Sscanf(majorVersions[j], "%d", &numJ)
		return numI > numJ
	})

	// Take up to 2 versions from each major version
	selectedVersions := []string{}
	for _, majorVersion := range majorVersions {
		versionsForMajor := majorVersionMap[majorVersion]
		// Take up to 2 versions per major version
		count := 0
		for _, v := range versionsForMajor {
			if count < 2 {
				selectedVersions = append(selectedVersions, v.ReleaseVersion)
				fmt.Printf("Including version %s (released on %s) - Major version: %s\n",
					v.ReleaseVersion, v.ReleaseDate, majorVersion)
				count++
			} else {
				fmt.Printf("Skipping extra version %s (already have 2 for major version %s)\n",
					v.ReleaseVersion, majorVersion)
			}
		}
	}

	// Sort versions in descending order
	sort.Slice(selectedVersions, func(i, j int) bool {
		return selectedVersions[i] > selectedVersions[j]
	})

	versions, err := fetchLatestBakeryVersions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching Kubernetes version: %v\n", err)
		os.Exit(1)
	}

	nvidiaRuntimeVersion := versions.nvidiaRuntimeVersion
	kubernetesVersion := versions.kubernetesVersion

	gpuOperatorVersion, err := fetchLatestGpuOperatorVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching GPU operator version: %v\n", err)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching CUDA sample image tag: %v\n", err)
		os.Exit(1)
	}

	// Generate the Go code
	output := fmt.Sprintf(`// Code generated by nvidia_versions_gen.go; DO NOT EDIT.

package misc

// GetNvidiaVersions returns the latest NVIDIA driver version per release branch
// Generated from %s
// Only includes versions released in 2024 or later
// Keeps up to 2 minor versions for each major version
func GetNvidiaVersions() []string {
	return []string{
%s
	}
}

const KubernetesVersion = "%s"
const NvidiaRuntimeVersion = "%s"
const GpuOperatorVersion = "%s"
`, versionsURL, formatVersionsAsGoList(selectedVersions), kubernetesVersion, nvidiaRuntimeVersion, gpuOperatorVersion)

	// Write the output file
	if err := ioutil.WriteFile(outputFile, []byte(output), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s with %d NVIDIA driver versions\n", outputFile, len(selectedVersions))
}

func formatVersionsAsGoList(versions []string) string {
	lines := make([]string, len(versions))
	for i, version := range versions {
		lines[i] = fmt.Sprintf("\t\t%q,", version)
	}
	return strings.Join(lines, "\n")
}
