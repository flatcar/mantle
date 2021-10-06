// Copyright 2016 CoreOS, Inc.
// Copyright 2018 Red Hat
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
	"net"

	"github.com/coreos/pkg/capnslog"

	ctplatform "github.com/coreos/container-linux-config-transpiler/config/platform"
	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/api/esx"
)

const (
	Platform platform.Name = "esx"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "platform/machine/esx")
)

type flight struct {
	*platform.BaseFlight
	api       *esx.API
	ips       chan esx.IpPair
	staticIps bool
}

func nextIpAddress(orig net.IP) net.IP {
	ip := make([]byte, len(orig))
	copy(ip, orig)
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i] = ip[i] + byte(1)
		if ip[i] != 0 {
			break
		}
	}
	return ip
}

// NewFlight creates an instance of a Flight suitable for spawning
// clusters on VMware ESXi vSphere platform.
func NewFlight(opts *esx.Options) (platform.Flight, error) {
	api, err := esx.New(opts)
	if err != nil {
		return nil, err
	}

	bf, err := platform.NewBaseFlight(opts.Options, Platform, ctplatform.Custom)
	if err != nil {
		return nil, err
	}

	ef := &flight{
		BaseFlight: bf,
		api:        api,
		ips:        make(chan esx.IpPair, opts.StaticIPs),
		staticIps:  opts.StaticIPs != 0,
	}

	if ef.staticIps {
		public := net.ParseIP(opts.FirstStaticIp)
		private := net.ParseIP(opts.FirstStaticIpPrivate)
		for i := 0; i < opts.StaticIPs; i++ {
			if i > 0 {
				public = nextIpAddress(public)
				private = nextIpAddress(private)
			}
			plog.Debugf("Calculated available static IP addresses: %v and %v", public, private)
			ef.ips <- esx.IpPair{Public: public, Private: private, SubnetSize: opts.StaticSubnetSize,
				PrivateGw: net.ParseIP(opts.StaticGatewayIpPrivate), PublicGw: net.ParseIP(opts.StaticGatewayIp)}
		}
	}

	return ef, nil
}

// NewCluster creates an instance of a Cluster suitable for spawning
// instances on VMware ESXi vSphere platform.
func (ef *flight) NewCluster(rconf *platform.RuntimeConfig) (platform.Cluster, error) {
	bc, err := platform.NewBaseCluster(ef.BaseFlight, rconf)
	if err != nil {
		return nil, err
	}

	ec := &cluster{
		BaseCluster: bc,
		flight:      ef,
	}

	ef.AddCluster(ec)

	return ec, nil
}
