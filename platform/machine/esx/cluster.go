// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package esx

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/esx"
	"github.com/flatcar-linux/mantle/platform/conf"
)

type cluster struct {
	*platform.BaseCluster
	flight *flight
}

func (ec *cluster) vmname() string {
	b := make([]byte, 5)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", ec.Name(), b)
}

func (ec *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	conf, err := ec.RenderUserData(userdata, map[string]string{
		"$public_ipv4":  "${COREOS_CUSTOM_PUBLIC_IPV4}",
		"$private_ipv4": "${COREOS_CUSTOM_PRIVATE_IPV4}",
	})
	if err != nil {
		return nil, err
	}

	var ipPairMaybe *esx.IpPair
	if ec.flight.staticIps {
		plog.Debugf("Trying to get static IP addresses...")
		ips := <-ec.flight.ips
		ipPairMaybe = &ips
		plog.Debugf("Got static IP addresses %v and %v", ips.Public, ips.Private)
		networkdConfig := fmt.Sprintf(`[Match]
Virtualization=vmware
Name=ens192

[Network]
DHCP=no
DNS=1.1.1.1
DNS=1.0.0.1

[Address]
Address=%s/%d

[Address]
Address=%s/%d

[Route]
Destination=0.0.0.0/0
Gateway=%s

[Route]
Destination=10.0.0.0/8
Gateway=%s
`, ips.Public, ips.SubnetSize, ips.Private, ips.SubnetSize, ips.PublicGw, ips.PrivateGw)
		// If no Ignition config is given, Ignition will use the default.ign which activates cloud-init
		// and cloud-init will use the guestinfo variables to setup the static IPs
		if conf.IsIgnition() {
			conf.AddFile("/etc/systemd/network/00-vmware.network", "root", networkdConfig, 0644)
		}
	}

	// This assumes that private IPs are in the form of 10.x.x.x
	conf.AddSystemdUnit("coreos-metadata.service", `[Unit]
Description=VMware metadata agent
After=nss-lookup.target
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
Environment=OUTPUT=/run/metadata/flatcar
ExecStart=/usr/bin/mkdir --parent /run/metadata
ExecStart=/usr/bin/bash -c 'echo "COREOS_CUSTOM_PRIVATE_IPV4=$(ip addr show ens192 | grep "inet 10." | grep -Po "inet \K[\d.]+")\nCOREOS_CUSTOM_PUBLIC_IPV4=$(ip addr show ens192 | grep -v "inet 10." | grep -Po "inet \K[\d.]+")" > ${OUTPUT}'
ExecStartPost=/usr/bin/ln -fs /run/metadata/flatcar /run/metadata/coreos
`, false)

	instance, err := ec.flight.api.CreateDevice(ec.vmname(), conf, ipPairMaybe)
	if err != nil {
		if ipPairMaybe != nil {
			plog.Debugf("Setting static IP addresses %v and %v as available", (*ipPairMaybe).Public, (*ipPairMaybe).Private)
			ec.flight.ips <- *ipPairMaybe
		}
		return nil, err
	}

	mach := &machine{
		cluster: ec,
		mach:    instance,
		ipPair:  ipPairMaybe,
	}

	mach.dir = filepath.Join(ec.RuntimeConf().OutputDir, mach.ID())
	if err := os.Mkdir(mach.dir, 0777); err != nil {
		mach.Destroy()
		return nil, err
	}

	confPath := filepath.Join(mach.dir, "user-data")
	if err := conf.WriteFile(confPath); err != nil {
		mach.Destroy()
		return nil, err
	}

	if mach.journal, err = platform.NewJournal(mach.dir); err != nil {
		mach.Destroy()
		return nil, err
	}

	if err := platform.StartMachine(mach, mach.journal); err != nil {
		mach.Destroy()
		return nil, err
	}

	ec.AddMach(mach)

	return mach, nil
}

func (ec *cluster) Destroy() {
	ec.BaseCluster.Destroy()
	ec.flight.DelCluster(ec)
}
