// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

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
