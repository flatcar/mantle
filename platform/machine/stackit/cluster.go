package stackit

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/stackit"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/stackitcloud/stackit-sdk-go/services/iaas"
	"k8s.io/utils/ptr"
	"os"
	"path/filepath"
)

type cluster struct {
	*platform.BaseCluster
	flight  *flight
	network *stackit.Network
	imageId *string
	keypair *stackit.Keypair
}

func (bc *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	ctx := context.TODO()

	userDataConf, err := bc.RenderUserData(userdata, map[string]string{
		"$public_ipv4":  "${COREOS_CUSTOM_PUBLIC_IPV4}",
		"$private_ipv4": "${COREOS_CUSTOM_PRIVATE_IPV4}",
	})
	if err != nil {
		return nil, err
	}

	// Hack to workaround CT inheritance.
	// Can be dropped once we remove CT dependency.
	// https://github.com/flatcar/Flatcar/issues/1386
	// * The sed 32 is to remove the "/32" from the "${IP}/32" notation
	// * The 'ip addr add' is a hack: the private link is around but the address is not assigned, on other OS there is a
	// "linode helper" that does this for you. (You can find the same thing on Cluster API template).
	userDataConf.AddSystemdUnitDropin("coreos-metadata.service", "00-custom-metadata.conf", `[Service]
ExecStartPost=/usr/bin/sed -i "s/STACKIT/CUSTOM/" /run/metadata/flatcar
ExecStartPost=/usr/bin/sed -i "s/PRIVATE_IPV4_0/PRIVATE_IPV4/" /run/metadata/flatcar
ExecStartPost=/usr/bin/sed -i "s/PUBLIC_IPV4_0/PUBLIC_IPV4/" /run/metadata/flatcar
ExecStartPost=/usr/bin/sed -i "s#/32##" /run/metadata/flatcar
ExecStartPost=/usr/bin/sh -c "ip addr add $(cat /run/metadata/flatcar | grep PRIVATE_IPV4 | cut -d '=' -f 2) dev eth0"
`)
	fmt.Printf("UserData: \n")
	fmt.Printf(userDataConf.String())
	byteConf, err := json.Marshal(userDataConf)
	if err != nil {
		return nil, fmt.Errorf("error marshaling config: %s", err)
	}
	if bc == nil {
		fmt.Printf("bc is null\n")
	}
	if bc.network == nil {
		fmt.Printf("bc is network is null\n")
	}
	if bc.keypair == nil {
		fmt.Printf("bc is keypair is null\n")
	}
	if byteConf != nil {
		fmt.Printf("conf is %s\n", string(byteConf))
	}

	//configTest := []byte(userDataConf.String())
	configTest := base64.StdEncoding.EncodeToString([]byte(userDataConf.String()))
	base64Config := []byte(configTest)

	secGroup, err := bc.flight.api.CreateSecurityGroup(ctx, "flatcar_security_group")
	if err != nil {
		return nil, fmt.Errorf("error creating security group: %s", err)
	}
	err = bc.flight.api.CreateSecurityGroupRule(ctx, *secGroup.Id)
	if err != nil {
		return nil, fmt.Errorf("error creating security group rule: %s", err)
	}

	ipAddress, err := bc.flight.api.CreateIPAddress(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating IP address: %s", err)
	}
	fmt.Printf("IP Address: %s\n", *ipAddress.Ip)

	instance, err := bc.flight.api.CreateServer(ctx, bc.vmname(), bc.network.NetworkId, bc.keypair.Name, &base64Config)
	if err != nil {
		fmt.Printf("error creating server: %s\n", err)
		return nil, err
	}

	err = bc.flight.api.AttachPublicIPAddress(ctx, *ipAddress.Id, *instance.Id)
	if err != nil {
		fmt.Printf("error attaching public IP address: %s\n", err)
	}

	err = bc.flight.api.AddSecurityGroup(ctx, *instance.Id, *secGroup.Id)
	if err != nil {
		fmt.Printf("error adding security group: %s\n", err)
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
		if err := bc.flight.api.DeleteNetwork(context.TODO(), *bc.network.NetworkId); err != nil {
			plog.Errorf("deleting network %v: %v", bc.network.Name, err)
		}
	}

	bc.flight.DelCluster(bc)
}
