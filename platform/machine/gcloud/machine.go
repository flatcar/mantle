// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package gcloud

import (
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"

	"github.com/flatcar-linux/mantle/platform"
)

type machine struct {
	gc      *cluster
	name    string
	intIP   string
	extIP   string
	dir     string
	journal *platform.Journal
	console string
}

func (gm *machine) ID() string {
	return gm.name
}

func (gm *machine) IP() string {
	return gm.extIP
}

func (gm *machine) PrivateIP() string {
	return gm.intIP
}

func (gm *machine) RuntimeConf() platform.RuntimeConfig {
	return gm.gc.RuntimeConf()
}

func (gm *machine) SSHClient() (*ssh.Client, error) {
	return gm.gc.SSHClient(gm.IP())
}

func (gm *machine) PasswordSSHClient(user string, password string) (*ssh.Client, error) {
	return gm.gc.PasswordSSHClient(gm.IP(), user, password)
}

func (gm *machine) SSH(cmd string) ([]byte, []byte, error) {
	return gm.gc.SSH(gm, cmd)
}

func (gm *machine) Reboot() error {
	return platform.RebootMachine(gm, gm.journal)
}

func (gm *machine) Destroy() {
	if err := gm.saveConsole(); err != nil {
		plog.Errorf("Error saving console for instance %v: %v", gm.ID(), err)
	}

	if err := gm.gc.flight.api.TerminateInstance(gm.name); err != nil {
		plog.Errorf("Error terminating instance %v: %v", gm.ID(), err)
	}

	if gm.journal != nil {
		gm.journal.Destroy()
	}

	gm.gc.DelMach(gm)
}

func (gm *machine) ConsoleOutput() string {
	return gm.console
}

func (gm *machine) saveConsole() error {
	var err error
	gm.console, err = gm.gc.flight.api.GetConsoleOutput(gm.name)
	if err != nil {
		return err
	}

	path := filepath.Join(gm.dir, "console.txt")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString(gm.console)

	return nil
}

func (gm *machine) JournalOutput() string {
	if gm.journal == nil {
		return ""
	}

	data, err := gm.journal.Read()
	if err != nil {
		plog.Errorf("Reading journal for instance %v: %v", gm.ID(), err)
	}
	return string(data)
}

func (gm *machine) Board() string {
	return gm.gc.flight.Options().Board
}
