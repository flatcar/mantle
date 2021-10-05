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

package external

import (
	"github.com/coreos/pkg/capnslog"

	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"

	ctplatform "github.com/coreos/container-linux-config-transpiler/config/platform"
	"github.com/flatcar-linux/mantle/network"
	"github.com/flatcar-linux/mantle/platform"
)

const (
	Platform platform.Name = "external"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "platform/machine/external")
)

type flight struct {
	*platform.BaseFlight
	ManagementSSHClient *ssh.Client
	ExternalOptions     *Options
}

type Options struct {
	*platform.Options
	ManagementHost     string
	ManagementUser     string
	ManagementPassword string
	ManagementSocks    string
	// Executed on the Management Node
	ProvisioningCmds   string
	SerialConsoleCmd   string
	DeprovisioningCmds string
}

func newManagementSSHClient(opts *Options) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: opts.ManagementUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(opts.ManagementPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if opts.ManagementSocks != "" {
		dialer, err := proxy.SOCKS5("tcp", opts.ManagementSocks, nil, nil)
		if err != nil {
			return nil, err
		}
		conn, err := dialer.Dial("tcp", opts.ManagementHost)
		if err != nil {
			return nil, err
		}
		ncc, chans, reqs, err := ssh.NewClientConn(conn, opts.ManagementHost, config)
		if err != nil {
			return nil, err
		}
		return ssh.NewClient(ncc, chans, reqs), nil
	}
	client, err := ssh.Dial("tcp", opts.ManagementHost, config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func NewFlight(opts *Options) (platform.Flight, error) {
	managementSSHClient, err := newManagementSSHClient(opts)
	if err != nil {
		return nil, err
	}

	retryDialer := network.RetryDialer{
		Dialer:  managementSSHClient,
		Retries: network.DefaultRetries,
	}
	bf, err := platform.NewBaseFlightWithDialer(opts.Options, Platform, ctplatform.Custom, &retryDialer)
	if err != nil {
		return nil, err
	}

	pf := &flight{
		BaseFlight:          bf,
		ExternalOptions:     opts,
		ManagementSSHClient: managementSSHClient,
	}

	return pf, nil
}

func (pf *flight) NewCluster(rconf *platform.RuntimeConfig) (platform.Cluster, error) {
	bc, err := platform.NewBaseCluster(pf.BaseFlight, rconf)
	if err != nil {
		return nil, err
	}

	pc := &cluster{
		BaseCluster: bc,
		flight:      pf,
	}

	pf.AddCluster(pc)

	return pc, nil
}

func (pf *flight) Destroy() {
	pf.BaseFlight.Destroy()
	pf.ManagementSSHClient.Close()
}
