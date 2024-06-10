// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package hetzner

import (
	"context"
	"strconv"

	"golang.org/x/crypto/ssh"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/hetzner"
)

type machine struct {
	cluster *cluster
	mach    *hetzner.Server
	dir     string
	journal *platform.Journal
	console string
}

// ID returns the ID of the machine.
func (bm *machine) ID() string {
	return strconv.FormatInt(bm.mach.Server.ID, 10)
}

// IP returns the IP of the machine.
func (bm *machine) IP() string {
	return bm.mach.Server.PublicNet.IPv4.IP.String()
}

// IP returns the private IP of the machine.
func (bm *machine) PrivateIP() string {
	if len(bm.mach.Server.PrivateNet) > 0 {
		return bm.mach.Server.PrivateNet[0].IP.String()
	}

	return ""
}

// RuntimeConf returns the runtime configuration of the cluster.
func (bm *machine) RuntimeConf() *platform.RuntimeConfig {
	return bm.cluster.RuntimeConf()
}

func (bm *machine) SSHClient() (*ssh.Client, error) {
	return bm.cluster.SSHClient(bm.IP())
}

func (bm *machine) PasswordSSHClient(user string, password string) (*ssh.Client, error) {
	return bm.cluster.PasswordSSHClient(bm.IP(), user, password)
}

func (bm *machine) SSH(cmd string) ([]byte, []byte, error) {
	return bm.cluster.SSH(bm, cmd)
}

func (bm *machine) Reboot() error {
	return platform.RebootMachine(bm, bm.journal)
}

func (bm *machine) Destroy() {
	// TODO: Add "saveConsole" logic here when Hetzner API will support fetching the console output.

	if err := bm.cluster.flight.api.DeleteServer(context.TODO(), bm.ID()); err != nil {
		plog.Errorf("deleting server %v: %v", bm.ID(), err)
	}

	if bm.journal != nil {
		bm.journal.Destroy()
	}

	bm.cluster.DelMach(bm)
}

func (bm *machine) ConsoleOutput() string {
	return bm.console
}

func (bm *machine) JournalOutput() string {
	if bm.journal == nil {
		return ""
	}

	data, err := bm.journal.Read()
	if err != nil {
		plog.Errorf("Reading journal for instance %v: %v", bm.ID(), err)
	}
	return string(data)
}

func (bm *machine) Board() string {
	return bm.cluster.flight.Options().Board
}
