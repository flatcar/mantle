// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package brightbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/brightbox"
)

type machine struct {
	cluster *cluster
	mach    *brightbox.Server
	dir     string
	journal *platform.Journal
	console string
}

// ID returns the ID of the machine.
func (bm *machine) ID() string {
	return bm.mach.Server.ID
}

// IP returns the IP of the machine.
// The machine should only get one "cloud" IP.
func (bm *machine) IP() string {
	if bm.mach.Server != nil && len(bm.mach.Server.CloudIPs) >= 1 {
		return bm.mach.Server.CloudIPs[0].PublicIPv4
	}

	return ""
}

func (bm *machine) PrivateIP() string {
	// Return the first IPv4 address, assuming it's the private one.
	for _, iface := range bm.mach.Server.Interfaces {
		return iface.IPv4Address
	}

	// Otherwise returns the public one in last resort.
	return bm.IP()
}

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
	// Keep the cloud IP ID to add it to the available pool after
	// machine deletion.
	var cloudIP string
	if bm.mach.Server != nil && len(bm.mach.Server.CloudIPs) >= 1 {
		cloudIP = bm.mach.Server.CloudIPs[0].ID
	}

	if err := bm.saveConsole(); err != nil {
		plog.Errorf("Error saving console for instance %v: %v", bm.ID(), err)
	}

	if err := bm.cluster.flight.api.DeleteServer(context.TODO(), bm.ID()); err != nil {
		plog.Errorf("deleting server %v: %v", bm.ID(), err)
	}

	if bm.journal != nil {
		bm.journal.Destroy()
	}

	bm.cluster.DelMach(bm)

	if cloudIP != "" {
		plog.Infof("Adding Cloud IP to the pool: %s", cloudIP)
		bm.cluster.flight.cloudIPs <- cloudIP
	}
}

func (bm *machine) ConsoleOutput() string {
	return bm.console
}

func (bm *machine) saveConsole() error {
	var err error
	bm.console, err = bm.cluster.flight.api.GetConsoleOutput(bm.ID())
	if err != nil {
		return fmt.Errorf("Error retrieving console log for %v: %v", bm.ID(), err)
	}

	path := filepath.Join(bm.dir, "console.txt")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString(bm.console)

	return nil
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
