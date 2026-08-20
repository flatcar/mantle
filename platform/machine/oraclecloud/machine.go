// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

import (
	"context"

	"golang.org/x/crypto/ssh"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/oraclecloud"
)

type machine struct {
	cluster *cluster
	mach    *oraclecloud.Instance
	dir     string
	journal *platform.Journal
	console string
}

func (om *machine) ID() string {
	if om.mach.Id == nil {
		return ""
	}
	return *om.mach.Id
}

func (om *machine) IP() string {
	return om.mach.PublicIP
}

func (om *machine) PrivateIP() string {
	return om.mach.PrivateIP
}

func (om *machine) RuntimeConf() *platform.RuntimeConfig {
	return om.cluster.RuntimeConf()
}

func (om *machine) SSHClient() (*ssh.Client, error) {
	return om.cluster.SSHClient(om.IP())
}

func (om *machine) PasswordSSHClient(user string, password string) (*ssh.Client, error) {
	return om.cluster.PasswordSSHClient(om.IP(), user, password)
}

func (om *machine) SSH(cmd string) ([]byte, []byte, error) {
	return om.cluster.SSH(om, cmd)
}

func (om *machine) Reboot() error {
	return platform.RebootMachine(om, om.journal)
}

func (om *machine) Destroy() {
	if id := om.ID(); id != "" {
		if err := om.cluster.flight.api.TerminateInstance(context.TODO(), id); err != nil {
			plog.Errorf("terminating instance %v: %v", id, err)
		}
	}

	if om.journal != nil {
		om.journal.Destroy()
	}

	om.cluster.DelMach(om)
}

func (om *machine) ConsoleOutput() string {
	return om.console
}

func (om *machine) JournalOutput() string {
	if om.journal == nil {
		return ""
	}

	data, err := om.journal.Read()
	if err != nil {
		plog.Errorf("Reading journal for instance %v: %v", om.ID(), err)
	}
	return string(data)
}

func (om *machine) Board() string {
	return om.cluster.flight.Options().Board
}
