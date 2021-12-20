// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package do

import (
	"context"
	"strconv"

	"github.com/digitalocean/godo"
	"golang.org/x/crypto/ssh"

	"github.com/flatcar-linux/mantle/platform"
)

type machine struct {
	cluster   *cluster
	droplet   *godo.Droplet
	journal   *platform.Journal
	publicIP  string
	privateIP string
}

func (dm *machine) ID() string {
	return strconv.Itoa(dm.droplet.ID)
}

func (dm *machine) IP() string {
	return dm.publicIP
}

func (dm *machine) PrivateIP() string {
	return dm.privateIP
}

func (dm *machine) RuntimeConf() platform.RuntimeConfig {
	return dm.cluster.RuntimeConf()
}

func (dm *machine) SSHClient() (*ssh.Client, error) {
	return dm.cluster.SSHClient(dm.IP())
}

func (dm *machine) PasswordSSHClient(user string, password string) (*ssh.Client, error) {
	return dm.cluster.PasswordSSHClient(dm.IP(), user, password)
}

func (dm *machine) SSH(cmd string) ([]byte, []byte, error) {
	return dm.cluster.SSH(dm, cmd)
}

func (dm *machine) Reboot() error {
	return platform.RebootMachine(dm, dm.journal)
}

func (dm *machine) Destroy() {
	if err := dm.cluster.flight.api.DeleteDroplet(context.TODO(), dm.droplet.ID); err != nil {
		plog.Errorf("Error deleting droplet %v: %v", dm.droplet.ID, err)
	}

	if dm.journal != nil {
		dm.journal.Destroy()
	}

	dm.cluster.DelMach(dm)
}

func (dm *machine) ConsoleOutput() string {
	// DigitalOcean provides no API for retrieving ConsoleOutput
	return ""
}

func (dm *machine) JournalOutput() string {
	if dm.journal == nil {
		return ""
	}

	data, err := dm.journal.Read()
	if err != nil {
		plog.Errorf("Reading journal for droplet %v: %v", dm.droplet.ID, err)
	}
	return string(data)
}

func (dm *machine) Board() string {
	return dm.cluster.flight.Options().Board
}
