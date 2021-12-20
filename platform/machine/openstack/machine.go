// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package openstack

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"

	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/openstack"
)

type machine struct {
	cluster *cluster
	mach    *openstack.Server
	dir     string
	journal *platform.Journal
	console string
}

func (om *machine) ID() string {
	return om.mach.Server.ID
}

func (om *machine) IP() string {
	if om.mach.FloatingIP != nil {
		return om.mach.FloatingIP.IP
	} else {
		return om.mach.Server.AccessIPv4
	}
}

func (om *machine) PrivateIP() string {
	for _, addrs := range om.mach.Server.Addresses {
		addrs, ok := addrs.([]interface{})
		if !ok {
			continue
		}
		for _, addr := range addrs {
			a, ok := addr.(map[string]interface{})
			if !ok {
				continue
			}
			iptype, ok := a["OS-EXT-IPS:type"].(string)
			ip, ok2 := a["addr"].(string)
			if ok && ok2 && iptype == "fixed" {
				return ip
			}
		}
	}
	return om.IP()
}

func (om *machine) RuntimeConf() platform.RuntimeConfig {
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
	if err := om.saveConsole(); err != nil {
		plog.Errorf("Error saving console for instance %v: %v", om.ID(), err)
	}

	if err := om.cluster.flight.api.DeleteServer(om.ID()); err != nil {
		plog.Errorf("deleting server %v: %v", om.ID(), err)
	}

	if om.journal != nil {
		om.journal.Destroy()
	}

	om.cluster.DelMach(om)
}

func (om *machine) ConsoleOutput() string {
	return om.console
}

func (om *machine) saveConsole() error {
	var err error
	om.console, err = om.cluster.flight.api.GetConsoleOutput(om.ID())
	if err != nil {
		return fmt.Errorf("Error retrieving console log for %v: %v", om.ID(), err)
	}

	path := filepath.Join(om.dir, "console.txt")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString(om.console)

	return nil
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
