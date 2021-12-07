// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"

	"github.com/coreos/pkg/capnslog"
	"github.com/spf13/cobra"

	"github.com/flatcar-linux/mantle/system/exec"
	"github.com/flatcar-linux/mantle/version"
)

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number and exit.",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("mantle/%s version %s\n",
				cmd.Root().Name(), version.Version)
		},
	}

	logDebug   bool
	logVerbose bool
	logLevel   = capnslog.NOTICE

	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "cli")
)

// Execute sets up common features that all mantle commands should share
// and then executes the command. It does not return.
func Execute(main *cobra.Command) {
	// If we were invoked via a multicall entrypoint run it instead.
	// TODO(marineam): should we figure out a way to initialize logging?
	exec.MaybeExec()

	main.AddCommand(versionCmd)

	// TODO(marineam): pflags defines the Value interface differently,
	// update capnslog accordingly...
	main.PersistentFlags().Var(&logLevel, "log-level",
		"Set global log level.")
	main.PersistentFlags().BoolVarP(&logVerbose, "verbose", "v", false,
		"Alias for --log-level=INFO")
	main.PersistentFlags().BoolVarP(&logDebug, "debug", "d", false,
		"Alias for --log-level=DEBUG")

	WrapPreRun(main, func(cmd *cobra.Command, args []string) error {
		startLogging(cmd)
		return nil
	})

	if err := main.Execute(); err != nil {
		plog.Fatal(err)
	}
	os.Exit(0)
}

func setRepoLogLevel(repo string, l capnslog.LogLevel) {
	r, err := capnslog.GetRepoLogger(repo)
	if err != nil {
		return // don't care if it isn't linked in
	}
	r.SetRepoLogLevel(l)
}

func startLogging(cmd *cobra.Command) {
	switch {
	case logDebug:
		logLevel = capnslog.DEBUG
	case logVerbose:
		logLevel = capnslog.INFO
	}

	capnslog.SetFormatter(capnslog.NewStringFormatter(cmd.Out()))
	capnslog.SetGlobalLogLevel(logLevel)

	// In the context of the internally linked etcd, the NOTICE messages
	// aren't really interesting, so translate NOTICE to WARNING instead.
	if logLevel == capnslog.NOTICE {
		// etcd sure has a lot of repos in its repo
		setRepoLogLevel("github.com/coreos/etcd", capnslog.WARNING)
		setRepoLogLevel("github.com/coreos/etcd/etcdserver", capnslog.WARNING)
		setRepoLogLevel("github.com/coreos/etcd/etcdserver/etcdhttp", capnslog.WARNING)
		setRepoLogLevel("github.com/coreos/etcd/pkg", capnslog.WARNING)
	}

	plog.Infof("Started logging at level %s", logLevel)
}

type PreRunEFunc func(cmd *cobra.Command, args []string) error

func WrapPreRun(root *cobra.Command, f PreRunEFunc) {
	preRun, preRunE := root.PersistentPreRun, root.PersistentPreRunE
	root.PersistentPreRun, root.PersistentPreRunE = nil, nil

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := f(cmd, args); err != nil {
			return err
		}
		if preRun != nil {
			preRun(cmd, args)
		} else if preRunE != nil {
			return preRunE(cmd, args)
		}
		return nil
	}
}
