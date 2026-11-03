package stackit

import (
	"context"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/stackit"
	"golang.org/x/crypto/ssh"
)

type machine struct {
	cluster    *cluster
	mach       *stackit.Server
	publicIPID string
	dir        string
	journal    *platform.Journal
	console    string
}

func (bm *machine) ID() string {
	return *bm.mach.Server.Id
}

func (bm *machine) IP() string {
	for _, nic := range bm.mach.Nics {
		if nic.HasPublicIp() {
			return *nic.PublicIp
		}
	}
	return ""
}

func (bm *machine) PrivateIP() string {
	for _, nic := range bm.mach.Nics {
		return *nic.Ipv4
	}
	return ""
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
	// TODO: Add "saveConsole" logic here when STACKIT API will support fetching the console output.
	ctx := context.TODO()
	api := bm.cluster.flight.api

	// Order matters: detach the IP first, then wait for the server to be
	// gone, and only then delete the IP. Deleting the IP while the NIC still
	// references it fails and leaves the IP behind for the GC.
	if bm.publicIPID != "" {
		if err := api.DetachPublicIPAddress(ctx, bm.publicIPID, bm.ID()); err != nil {
			plog.Errorf("detaching public ip from server %v: %v", bm.ID(), err)
		}
	}

	if err := api.DeleteServer(ctx, bm.ID()); err != nil {
		plog.Errorf("deleting server %v: %v", bm.ID(), err)
	}

	if bm.publicIPID != "" {
		if err := api.DeleteIPAddress(ctx, bm.publicIPID); err != nil {
			plog.Errorf("deleting public ip %v: %v", bm.publicIPID, err)
		}
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
