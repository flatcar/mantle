package stackit

import (
	"context"
	"fmt"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/stackit"
	"golang.org/x/crypto/ssh"
)

type machine struct {
	cluster *cluster
	mach    *stackit.Server
	dir     string
	journal *platform.Journal
	console string
}

// ID returns the ID of the machine.
func (bm *machine) ID() string {
	return *bm.mach.Server.Id
}

// IP returns the IP of the machine.
func (bm *machine) IP() string {
	fmt.Printf("machine info: %+v\n", *bm.mach.Server)
	if bm.mach.Server.HasErrorMessage() {
		fmt.Printf("Error message: %+v\n", bm.mach.Server.ErrorMessage)
	}
	if bm.mach.Nics != nil && len(*bm.mach.Nics) > 0 {
		for _, nic := range *bm.mach.Nics {
			if nic.HasPublicIp() {
				return *nic.PublicIp
			}
			return ""
		}
	}
	return ""
}

// PrivateIP returns the private IP of the machine.
func (bm *machine) PrivateIP() string {
	fmt.Printf("machine info: %+v\n", *bm.mach.Server)
	if bm.mach.Server.HasErrorMessage() {
		fmt.Printf("Error message: %+v\n", bm.mach.Server.ErrorMessage)
	}
	if bm.mach.Nics != nil && len(*bm.mach.Nics) > 0 {
		for _, nic := range *bm.mach.Nics {
			return *nic.Ipv4
		}
	}
	return ""
}

func (bm *machine) RuntimeConf() *platform.RuntimeConfig {
	return bm.cluster.RuntimeConf()
}

func (bm *machine) SSHClient() (*ssh.Client, error) {
	fmt.Printf("SSH IP: %s\n", bm.IP())
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

	if err := bm.cluster.flight.api.DeleteServer(context.TODO(), *bm.mach.Id); err != nil {
		plog.Errorf("deleting server %v: %v", bm.ID(), err)
	}

	for _, secGroup := range bm.mach.GetSecurityGroups() {
		err := bm.cluster.flight.api.DeleteSecurityGroup(context.TODO(), secGroup)
		plog.Errorf("deleting server %v security group: %v", bm.ID(), err)
	}

	for _, nic := range bm.mach.GetNics() {
		if nic.HasPublicIp() {
			err := bm.cluster.flight.api.DeleteIPAddress(context.TODO(), nic.GetPublicIp())
			plog.Errorf("deleting server %v public IP: %v", bm.ID(), err)
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
	if bm.journal != nil {
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
