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
	"os"
	"path/filepath"

	"github.com/coreos/go-semver/semver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/coreos/mantle/sdk"
	"github.com/coreos/mantle/sdk/repo"
)

const (
	coreosManifestURL = "https://github.com/kinvolk/manifest.git"
	// Set repoUpstreamBranch to "maint", until the upstream repo >= v2.10
	// could be available in ordinary SDK environments. That is to avoid
	// incompatibility issue of an old repo tool not being able to work with
	// the default "stable" branch of the repo tool,
	// https://gerrit.googlesource.com/git-repo/+/refs/heads/stable,
	// which does not support python2 any more. OTOH its "maint" branch still
	// supports python2. In the long term, we should update "dev-vcs/repo"
	// in Flatcar SDK to v2.10 with python3, and set the default branch back
	// to "stable".
	repoUpstreamBranch = "maint"
)

var (
	// everything uses this flag
	chrootFlags *pflag.FlagSet
	chrootName  string

	// creation flags
	creationFlags  *pflag.FlagSet
	sdkUrlPath     string
	sdkVersion     string
	manifestURL    string
	manifestName   string
	manifestBranch string
	repoBranch     string
	repoVerify     bool
	sigVerify      bool

	// only for `create` command
	allowReplace bool

	// only for `enter` command
	bindGpgAgent bool

	// for create/update/enter
	useHostDNS bool

	// only for `update` command
	allowCreate      bool
	forceSync        bool
	downgradeInPlace bool
	downgradeReplace bool
	newVersion       string

	verifyKeyFile string

	scriptsPatch string
	portagePatch string
	overlayPatch string

	setupCmd = &cobra.Command{
		Use:   "setup",
		Short: "Setup the system for SDK use",
		Run:   runSetup,
	}
	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Download and unpack the SDK",
		Run:   runCreate,
	}
	enterCmd = &cobra.Command{
		Use:   "enter [-- command]",
		Short: "Enter the SDK chroot, optionally running a command",
		Run:   runEnter,
	}
	deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete the SDK chroot",
		Run:   runDelete,
	}
	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update the SDK chroot and source tree",
		Run:   runUpdate,
	}
	verifyCmd = &cobra.Command{
		Use:   "verify",
		Short: "Check repo tree and release manifest match",
		Run:   runVerify,
	}
)

func init() {
	// the names and error handling of these flag sets are meaningless,
	// the flag sets are only used to group common options together.
	chrootFlags = pflag.NewFlagSet("chroot", pflag.ExitOnError)
	chrootFlags.StringVar(&chrootName,
		"chroot", "chroot", "SDK chroot directory name")
	chrootFlags.BoolVar(&useHostDNS,
		"use-host-dns", false, "Use the host's /etc/resolv.conf instead of 8.8.8.8 and 8.8.4.4")

	creationFlags = pflag.NewFlagSet("creation", pflag.ExitOnError)
	creationFlags.StringVar(&sdkUrlPath,
		"sdk-url-path", "/flatcar-jenkins/sdk", "SDK URL path")
	creationFlags.StringVar(&sdkVersion,
		"sdk-version", "", "SDK version. Defaults to the SDK version in version.txt")
	creationFlags.StringVar(&manifestURL,
		"manifest-url", coreosManifestURL, "Manifest git repo location")
	creationFlags.StringVar(&manifestBranch,
		"manifest-branch", "flatcar-master", "Manifest git repo branch")
	creationFlags.StringVar(&manifestName,
		"manifest-name", "default.xml", "Manifest file name")
	creationFlags.StringVar(&repoBranch,
		"repo-branch", repoUpstreamBranch, "Branch name to be used from the upstream git repo of repo tool")
	creationFlags.BoolVar(&repoVerify,
		"verify", false, "Check repo tree and release manifest match")
	creationFlags.StringVar(&verifyKeyFile,
		"verify-key", "", "PGP public key to be used in verifing download signatures.  Defaults to CoreOS Buildbot (0412 7D0B FABE C887 1FFB  2CCE 50E0 8855 93D2 DCB4)")
	creationFlags.BoolVar(&sigVerify,
		"verify-signature", false, "Verify the manifest Git tag with GPG")
	creationFlags.StringVar(&scriptsPatch,
		"scripts-patch", "", "Path to a .patch file (can be a concatenation of multiple patches, e.g., from git format-patch -2) that is committed to the scripts repository after the manifest reference is checked out")
	creationFlags.StringVar(&portagePatch,
		"portage-patch", "", "Path to a .patch file (can be a concatenation of multiple patches, e.g., from git format-patch -2) that is committed to the portage repository after the manifest reference is checked out")
	creationFlags.StringVar(&overlayPatch,
		"overlay-patch", "", "Path to a .patch file (can be a concatenation of multiple patches, e.g., from git format-patch -2) that is committed to the overlay repository after the manifest reference is checked out")

	root.AddCommand(setupCmd)

	createCmd.Flags().AddFlagSet(chrootFlags)
	createCmd.Flags().AddFlagSet(creationFlags)
	createCmd.Flags().BoolVar(&allowReplace,
		"replace", false, "Replace an existing SDK chroot")
	root.AddCommand(createCmd)

	enterCmd.Flags().AddFlagSet(chrootFlags)
	enterCmd.Flags().BoolVar(&bindGpgAgent,
		"bind-gpg-agent", true, "bind mount the gpg agent socket directory")
	root.AddCommand(enterCmd)

	deleteCmd.Flags().AddFlagSet(chrootFlags)
	root.AddCommand(deleteCmd)

	updateCmd.Flags().AddFlagSet(chrootFlags)
	updateCmd.Flags().AddFlagSet(creationFlags)
	updateCmd.Flags().BoolVar(&allowCreate,
		"create", false, "Create the SDK chroot if missing")
	updateCmd.Flags().BoolVar(&forceSync,
		"force-sync", false, "Overrwrite stale .git directories if needed")
	updateCmd.Flags().BoolVar(&downgradeInPlace,
		"downgrade-in-place", false,
		"Allow in-place downgrades of SDK chroot")
	updateCmd.Flags().BoolVar(&downgradeReplace,
		"downgrade-replace", false,
		"Replace SDK chroot instead of downgrading")
	updateCmd.Flags().StringVar(&newVersion,
		"new-version", "", "Hint at the new version. Defaults to the version in version.txt")
	root.AddCommand(updateCmd)

	root.AddCommand(verifyCmd)
}

func runSetup(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		plog.Fatal("No args accepted")
	}

	if err := installQemuBinfmt(); err != nil {
		plog.Fatal("Install QEMU binfmt failed: ", err)
	}
}

func installQemuBinfmt() error {
	const dest = "/etc/binfmt.d/qemu-aarch64.conf"
	const data = ":qemu-aarch64:M::\\x7fELF\\x02\\x01\\x01\\x00\\x00\\x00\\x00\\x00\\x00\\x00\\x00\\x00\\x02\\x00\\xb7:\\xff\\xff\\xff\\xff\\xff\\xff\\xff\\x00\\xff\\xff\\xff\\xff\\xff\\xff\\xff\\xff\\xfe\\xff\\xff:/usr/bin/qemu-aarch64-static:"

	// Only install if file does not exist
	if _, err := os.Stat(dest); err == nil {
		return nil
	}

	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(dest)
	if err != nil {
		return err
	}

	if _, err := file.WriteString(data); err != nil {
		return err
	}

	if err := file.Close(); err != nil {
		os.Remove(dest)
		return err
	}

	return nil
}

func runCreate(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		plog.Fatal("No args accepted")
	}

	if sdkVersion == "" {
		plog.Noticef("Detecting SDK version")

		getRemoteVersions := sdk.VersionsFromRemoteRepo
		if sigVerify {
			getRemoteVersions = sdk.VersionsFromSignedRemoteRepo
		}

		if ver, err := sdk.VersionsFromManifest(); err == nil {
			sdkVersion = ver.SDKVersion
			plog.Noticef("Found SDK version %s from local repo", sdkVersion)
		} else if ver, err := getRemoteVersions(manifestURL, manifestBranch); err == nil {
			sdkVersion = ver.SDKVersion
			plog.Noticef("Found SDK version %s from remote repo", sdkVersion)
		} else {
			plog.Fatalf("Reading from remote repo failed: %v", err)
		}
	}

	unpackChroot(allowReplace)
	updateRepo()
	sdk.SetManifestSDKVersion(sdkVersion)
}

func unpackChroot(replace bool) {
	plog.Noticef("Downloading SDK version %s", sdkVersion)
	if err := sdk.DownloadSDK(sdkUrlPath, sdkVersion, verifyKeyFile); err != nil {
		plog.Fatalf("Download failed: %v", err)
	}

	if replace {
		if err := sdk.Delete(chrootName); err != nil {
			plog.Fatalf("Replace failed: %v", err)
		}
	}

	if err := sdk.Unpack(sdkVersion, chrootName); err != nil {
		plog.Fatalf("Create failed: %v", err)
	}

	if err := sdk.Setup(chrootName); err != nil {
		plog.Fatalf("Create failed: %v", err)
	}
}

func updateRepo() {
	if err := sdk.RepoInit(chrootName, manifestURL, manifestBranch, manifestName, repoBranch, useHostDNS); err != nil {
		plog.Fatalf("repo init failed: %v", err)
	}

	if sigVerify {
		if err := sdk.RepoVerifyTag(manifestBranch); err != nil {
			plog.Fatalf("repo tag verification failed: %v", err)
		}
	}

	if err := sdk.RepoSync(chrootName, forceSync, useHostDNS); err != nil {
		plog.Fatalf("repo sync failed: %v", err)
	}

	if repoVerify {
		if err := repo.VerifySync(manifestName); err != nil {
			plog.Fatalf("Verify failed: %v", err)
		}
	}

	if scriptsPatch != "" {
		if err := sdk.ApplyPatch(chrootName, useHostDNS, "src/scripts", scriptsPatch); err != nil {
			plog.Fatalf("Applying scripts patch failed: %v", err)
		}
	}
	if portagePatch != "" {
		if err := sdk.ApplyPatch(chrootName, useHostDNS, "src/third_party/portage-stable", portagePatch); err != nil {
			plog.Fatalf("Applying portage patch failed: %v", err)
		}
	}
	if overlayPatch != "" {
		if err := sdk.ApplyPatch(chrootName, useHostDNS, "src/third_party/coreos-overlay", overlayPatch); err != nil {
			plog.Fatalf("Applying overlay patch failed: %v", err)
		}
	}
}

func runEnter(cmd *cobra.Command, args []string) {
	err := sdk.Enter(chrootName, bindGpgAgent, useHostDNS, args...)
	if err != nil && len(args) != 0 {
		plog.Fatalf("Running %v failed: %v", args, err)
	}
}

func runDelete(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		plog.Fatal("No args accepted")
	}

	if err := sdk.Delete(chrootName); err != nil {
		plog.Fatalf("Delete failed: %v", err)
	}
}

func verLessThan(a, b string) bool {
	aver, err := semver.NewVersion(a)
	if err != nil {
		plog.Fatal(err)
	}
	bver, err := semver.NewVersion(b)
	if err != nil {
		plog.Fatal(err)
	}
	return aver.LessThan(*bver)
}

func runUpdate(cmd *cobra.Command, args []string) {
	const updateChroot = "/mnt/host/source/src/scripts/update_chroot"
	updateCommand := append([]string{updateChroot}, args...)

	// avoid downgrade strategy ambiguity
	if downgradeInPlace && downgradeReplace {
		plog.Fatal("Conflicting downgrade options")
	}

	if sdkVersion == "" || newVersion == "" {
		plog.Notice("Detecting versions in remote repo")
		getRemoteVersions := sdk.VersionsFromRemoteRepo
		if sigVerify {
			getRemoteVersions = sdk.VersionsFromSignedRemoteRepo
		}
		ver, err := getRemoteVersions(manifestURL, manifestBranch)
		if err != nil {
			plog.Fatalf("Reading from remote repo failed: %v", err)
		}

		if newVersion == "" {
			newVersion = ver.Version
		}

		if sdkVersion == "" {
			sdkVersion = ver.SDKVersion
		}
	}

	plog.Infof("New version %s", newVersion)
	plog.Infof("SDK version %s", sdkVersion)

	plog.Info("Checking version of local chroot")
	chroot := filepath.Join(sdk.RepoRoot(), chrootName)
	old, err := sdk.OSRelease(chroot)
	if err != nil {
		if allowCreate && os.IsNotExist(err) {
			unpackChroot(false)
		} else {
			plog.Fatal(err)
		}
	} else if verLessThan(newVersion, old.Version) {
		plog.Noticef("Downgrade from %s to %s required!",
			old.Version, newVersion)
		if downgradeReplace {
			unpackChroot(true)
		} else if downgradeInPlace {
			plog.Infof("Attempting to downgrade existing chroot.")
		} else {
			plog.Fatalf("Refusing to downgrade.")
		}
	}

	updateRepo()
	sdk.SetManifestSDKVersion(sdkVersion)

	if err := sdk.Enter(chrootName, false, false, updateCommand...); err != nil {
		plog.Fatalf("update_chroot failed: %v", err)
	}
}

func runVerify(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		plog.Fatal("No args accepted")
	}

	if err := repo.VerifySync(""); err != nil {
		plog.Fatalf("Verify failed: %v", err)
	}
}
