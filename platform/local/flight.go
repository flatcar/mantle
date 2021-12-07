// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"fmt"
	"sync/atomic"

	"github.com/coreos/go-omaha/omaha"
	"github.com/vishvananda/netns"

	"github.com/flatcar-linux/mantle/lang/destructor"
	"github.com/flatcar-linux/mantle/network"
	"github.com/flatcar-linux/mantle/network/ntp"
	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/system/ns"
)

const (
	listenPortBase = 30000
)

type LocalFlight struct {
	destructor.MultiDestructor
	*platform.BaseFlight
	Dnsmasq    *Dnsmasq
	SimpleEtcd *SimpleEtcd
	NTPServer  *ntp.Server
	nshandle   netns.NsHandle
	listenPort int32
}

func NewLocalFlight(opts *platform.Options, platformName platform.Name) (*LocalFlight, error) {
	nshandle, err := ns.Create()
	if err != nil {
		return nil, fmt.Errorf("creating new ns handle failed: %v", err)
	}

	nsdialer := network.NewNsDialer(nshandle)
	bf, err := platform.NewBaseFlightWithDialer(opts, platformName, "custom", nsdialer)
	if err != nil {
		nshandle.Close()
		return nil, fmt.Errorf("creating new base flight failed: %v", err)
	}

	lf := &LocalFlight{
		BaseFlight: bf,
		nshandle:   nshandle,
		listenPort: listenPortBase,
	}
	lf.AddDestructor(lf.BaseFlight)
	lf.AddCloser(&lf.nshandle)

	// dnsmasq and etcd must be launched in the new namespace
	nsExit, err := ns.Enter(lf.nshandle)
	if err != nil {
		lf.Destroy()
		return nil, fmt.Errorf("entering new ns failed: %v", err)
	}
	defer nsExit()

	lf.Dnsmasq, err = NewDnsmasq()
	if err != nil {
		lf.Destroy()
		return nil, fmt.Errorf("creating new dnsmasq failed: %v", err)
	}
	lf.AddDestructor(lf.Dnsmasq)

	lf.SimpleEtcd, err = NewSimpleEtcd()
	if err != nil {
		lf.Destroy()
		return nil, fmt.Errorf("creating new simple etcd failed: %v", err)
	}
	lf.AddDestructor(lf.SimpleEtcd)

	lf.NTPServer, err = ntp.NewServer(":123")
	if err != nil {
		lf.Destroy()
		return nil, fmt.Errorf("creating new ntp server failed: %v", err)
	}
	lf.AddCloser(lf.NTPServer)
	go lf.NTPServer.Serve()

	return lf, nil
}

func (lf *LocalFlight) NewCluster(rconf *platform.RuntimeConfig) (*LocalCluster, error) {
	lc := &LocalCluster{
		flight: lf,
	}

	var err error
	lc.BaseCluster, err = platform.NewBaseCluster(lf.BaseFlight, rconf)
	if err != nil {
		lc.Destroy()
		return nil, err
	}
	lc.AddDestructor(lc.BaseCluster)

	// Omaha server must be launched in the new namespace
	nsExit, err := ns.Enter(lf.nshandle)
	if err != nil {
		lc.Destroy()
		return nil, err
	}
	defer nsExit()

	omahaServer, err := omaha.NewTrivialServer(fmt.Sprintf(":%d", lf.newListenPort()))
	if err != nil {
		lc.Destroy()
		return nil, err
	}
	lc.OmahaServer = OmahaWrapper{TrivialServer: omahaServer}
	lc.AddDestructor(lc.OmahaServer)
	go lc.OmahaServer.Serve()

	// does not lf.AddCluster() since we are not the top-level object

	return lc, nil
}

func (lf *LocalFlight) newListenPort() int {
	return int(atomic.AddInt32(&lf.listenPort, 1))
}

func (lf *LocalFlight) Destroy() {
	lf.MultiDestructor.Destroy()
}
