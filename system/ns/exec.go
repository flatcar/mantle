// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package ns

import (
	"github.com/vishvananda/netns"

	"github.com/flatcar-linux/mantle/system/exec"
)

type Cmd struct {
	*exec.ExecCmd
	NsHandle netns.NsHandle
}

func Command(ns netns.NsHandle, name string, arg ...string) *Cmd {
	return &Cmd{
		ExecCmd:  exec.Command(name, arg...),
		NsHandle: ns,
	}
}

func (cmd *Cmd) CombinedOutput() ([]byte, error) {
	nsExit, err := Enter(cmd.NsHandle)
	if err != nil {
		return nil, err
	}
	defer nsExit()

	return cmd.ExecCmd.CombinedOutput()
}

func (cmd *Cmd) Output() ([]byte, error) {
	nsExit, err := Enter(cmd.NsHandle)
	if err != nil {
		return nil, err
	}
	defer nsExit()

	return cmd.ExecCmd.Output()
}

func (cmd *Cmd) Run() error {
	nsExit, err := Enter(cmd.NsHandle)
	if err != nil {
		return err
	}
	defer nsExit()

	return cmd.ExecCmd.Run()
}

func (cmd *Cmd) Start() error {
	nsExit, err := Enter(cmd.NsHandle)
	if err != nil {
		return err
	}
	defer nsExit()

	return cmd.ExecCmd.Start()
}
