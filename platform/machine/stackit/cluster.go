package stackit

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/stackit"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/util"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"k8s.io/utils/ptr"
)

type cluster struct {
	*platform.BaseCluster
	flight  *flight
	network *stackit.Network
	keypair *stackit.Keypair
}

func (bc *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	ctx := context.TODO()

	userDataConf, err := bc.RenderUserData(userdata, map[string]string{
		"$public_ipv4":  "${COREOS_OPENSTACK_IPV4_PUBLIC}",
		"$private_ipv4": "${COREOS_OPENSTACK_IPV4_PRIVATE}",
	})
	if err != nil {
		return nil, err
	}

	base64Config := make([]byte, base64.StdEncoding.EncodedLen(len(userDataConf.Bytes())))
	base64.StdEncoding.Encode(base64Config, userDataConf.Bytes())

	secGroup, err := bc.flight.api.CreateSecurityGroup(ctx, "flatcar_security_group")
	if err != nil {
		return nil, fmt.Errorf("error creating security group: %w", err)
	}
	err = bc.flight.api.CreateSecurityGroupRuleTCP(ctx, *secGroup.Id)
	if err != nil {
		return nil, fmt.Errorf("error creating security group rule: %w", err)
	}
	err = bc.flight.api.CreateSecurityGroupRuleUDP(ctx, *secGroup.Id)
	if err != nil {
		return nil, fmt.Errorf("error creating security group rule: %w", err)
	}

	ipAddress, err := bc.flight.api.CreateIPAddress(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating IP address: %w", err)
	}

	var keyPairName iaas.CreateServerPayloadGetKeypairNameAttributeType
	if bc.keypair != nil {
		keyPairName = bc.keypair.Name
	}
	securityGoups := &[]string{*secGroup.Id}
	instance, err := bc.flight.api.CreateServer(ctx, bc.vmname(), bc.network.Id, securityGoups, keyPairName, &base64Config)
	if err != nil {
		return nil, fmt.Errorf("creating server: %w", err)
	}

	err = bc.flight.api.AttachPublicIPAddress(ctx, *ipAddress.Id, *instance.Id)
	if err != nil {
		return nil, fmt.Errorf("attaching public IP address: %w", err)
	}

	// The API does sometimes need a couple of seconds to report the attached IP address
	err = util.Retry(5, 2*time.Second, func() error {
		instance, err = bc.flight.api.GetServer(ctx, *instance.Id)
		if err != nil {
			return fmt.Errorf("error getting server: %w", err)
		}
		if !instance.HasNics() {
			return fmt.Errorf("no NICs available")
		}
		hasPublicIP := false
		for _, nic := range instance.GetNics() {
			if nic.HasPublicIp() {
				hasPublicIP = true
			}
		}
		if hasPublicIP {
			return nil
		}
		return fmt.Errorf("server does not have a public IP address")
	})
	if err != nil {
		return nil, err
	}

	mach := &machine{
		cluster: bc,
		mach:    instance,
	}

	m := mach
	defer func() {
		if m != nil {
			m.Destroy()
		}
	}()

	mach.dir = filepath.Join(bc.RuntimeConf().OutputDir, mach.ID())
	if err := os.MkdirAll(mach.dir, 0777); err != nil {
		return nil, err
	}

	configPath := filepath.Join(mach.dir, "ignition.json")
	if err := userDataConf.WriteFile(configPath); err != nil {
		return nil, err
	}

	if mach.journal, err = platform.NewJournal(mach.dir); err != nil {
		return nil, err
	}

	if err := platform.StartMachine(mach, mach.journal); err != nil {
		return nil, err
	}

	m = nil
	bc.AddMach(mach)

	return mach, nil

}

func (bc *cluster) vmname() iaas.CreateServerPayloadGetNameAttributeType {
	b := make([]byte, 5)
	rand.Read(b)
	return ptr.To(fmt.Sprintf("%s-%x", bc.Name()[0:13], b))
}

func (bc *cluster) Destroy() {
	bc.BaseCluster.Destroy()
	if bc.network != nil {
		if err := bc.flight.api.DeleteNetwork(context.TODO(), *bc.network.Id); err != nil {
			plog.Errorf("deleting network %v: %v", *bc.network.Name, err)
		}
	}

	bc.flight.DelCluster(bc)
}
