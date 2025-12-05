package stackit

import (
	"context"
	"time"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/stackit"
	"github.com/flatcar/mantle/util"
	"golang.org/x/crypto/ssh"
)

type machine struct {
	cluster *cluster
	mach    *stackit.Server
	dir     string
	journal *platform.Journal
	console string
}

func (bm *machine) ID() string {
	return *bm.mach.Server.Id
}

func (bm *machine) IP() string {
	if bm.mach.Nics != nil && len(*bm.mach.Nics) > 0 {
		for _, nic := range *bm.mach.Nics {
			if nic.HasPublicIp() {
				return *nic.PublicIp
			}
		}
	}
	return ""
}

func (bm *machine) PrivateIP() string {
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
	server, err := bm.cluster.flight.api.GetServer(ctx, *bm.mach.Id)
	if err != nil {
		plog.Errorf("error getting server: %s", err)
	}

	for _, nic := range *server.Nics {
		for _, securityGroupID := range *nic.SecurityGroups {
			if err := bm.cluster.flight.api.RemoveSecurityGroupFromServer(ctx, server.GetId(), securityGroupID); err != nil {
				plog.Errorf("error removing security group from server: %s", err)
			}
			securityGroup, err := bm.cluster.flight.api.GetSecurityGroup(ctx, securityGroupID)
			if err != nil {
				plog.Errorf("error getting security group: %s", err)
			}
			for _, securityGroupRule := range securityGroup.GetRules() {
				if err := bm.cluster.flight.api.DeleteSecurityGroupRule(ctx, securityGroupID, *securityGroupRule.Id); err != nil {
					plog.Error("error deleting security group rule: %s", err)
				}
			}
			if err := util.Retry(5, 10*time.Second, func() error {
				return bm.cluster.flight.api.DeleteSecurityGroup(ctx, securityGroupID)
			}); err != nil {
				plog.Errorf("error deleting security group: %s", err)
			}
		}
		networkID := nic.GetNetworkId()
		if err := bm.cluster.flight.api.RemoveNetworkFromServer(ctx, server.GetId(), networkID); err != nil {
			plog.Errorf("error removing server from network: %s", err)
		}
		if nic.HasPublicIp() {
			if err := bm.cluster.flight.api.DeleteIPAddressByIP(ctx, nic.GetPublicIp()); err != nil {
				plog.Errorf("error deleting public ip: %s", err)
			}
		}
	}

	if err := bm.cluster.flight.api.DeleteServer(context.TODO(), *bm.mach.Id); err != nil {
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
