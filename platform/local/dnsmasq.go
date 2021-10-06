// Copyright 2021 Kinvolk GmbH
// Copyright 2015 CoreOS, Inc.
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

package local

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"text/template"

	"github.com/coreos/go-iptables/iptables"
	"github.com/coreos/pkg/capnslog"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/flatcar-linux/mantle/system/exec"
	"github.com/flatcar-linux/mantle/system/ns"
	"github.com/flatcar-linux/mantle/util"
)

var (
	// ErrIPClash is raised when a generated IP lives in an existing
	// root network namespace
	ErrIPClash = errors.New("IP lives in a host network")

	// ErrIncorrectSeed is raised when the seed used to generate veth pair
	// is in an incorrect format
	ErrIncorrectSeed = errors.New("seed must be a positive 2 bytes value")
)

type Interface struct {
	HardwareAddr net.HardwareAddr
	DHCPv4       []net.IPNet
	DHCPv6       []net.IPNet
	//SLAAC net.IPAddr
}

type Segment struct {
	BridgeName string
	BridgeIf   *Interface
	Interfaces []*Interface
	nextIf     int
	// Listener holds the unique TCP socket
	// created to ensure uniqueness of IP
	// it has to be closed once the kola instance
	// has terminated
	Listener net.Listener
}

type Dnsmasq struct {
	Segments []*Segment
	dnsmasq  *exec.ExecCmd
}

const (
	numInterfaces = 500 // affects dnsmasq startup time
	numSegments   = 1

	debugConfig = `
log-queries
log-dhcp
`

	quietConfig = `
quiet-dhcp
quiet-dhcp6
quiet-ra
`

	commonConfig = `
keep-in-foreground
leasefile-ro
log-facility=-
pid-file=

# hardcode DNS servers to avoid using systemd-resolved on the unreachable 127.0.0.53
dhcp-option=6,1.1.1.1,1.0.0.1,8.8.8.8
no-resolv
no-hosts

enable-ra

# point NTP at this host (0.0.0.0 and :: are special)
dhcp-option=option:ntp-server,0.0.0.0
dhcp-option=option6:ntp-server,[::]

{{range .Segments}}
domain={{.BridgeName}}.local

{{range .BridgeIf.DHCPv4}}
dhcp-range={{.IP}},static
{{end}}

{{range .BridgeIf.DHCPv6}}
dhcp-range={{.IP}},ra-names,slaac
{{end}}

{{range .Interfaces}}
dhcp-host={{.HardwareAddr}}{{template "ips" .DHCPv4}}{{template "ips" .DHCPv6}}
{{end}}
{{end}}

{{define "ips"}}{{range .}}{{printf ",%s" .IP}}{{end}}{{end}}
`
)

var plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "platform/local")

func newInterface(s byte, i uint16) *Interface {
	return &Interface{
		HardwareAddr: net.HardwareAddr{0x02, s, 0, 0, byte(i / 256), byte(i % 256)},
		DHCPv4: []net.IPNet{{
			IP:   net.IP{10, s, byte(i / 256), byte(i % 256)},
			Mask: net.CIDRMask(16, 32)}},
		DHCPv6: []net.IPNet{{
			IP:   net.IP{0xfd, s, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, byte(i / 256), byte(i % 256)},
			Mask: net.CIDRMask(64, 128)}},
	}
}

// configureNAT creates and append a NAT rule
// using the interface i as output
func configureNAT(i string) error {
	table, err := iptables.New()
	if err != nil {
		return fmt.Errorf("unable to get iptables: %w", err)
	}

	if err := table.AppendUnique("nat", "POSTROUTING", "-j", "MASQUERADE", "-o", i); err != nil {
		return fmt.Errorf("unable to append rule: %w", err)
	}

	return nil
}

// generateVethPair creates and returns a map holding
// a pair of veth based on a seed
// where:
// [0] is the network namespace veth
// [1] is the host namespace veth
// [2] is the next hop for adding route
func generateVethPair(seed int, hostNetworks []*net.IPNet) ([]string, error) {
	if seed < 0 || seed > 0xFFFF {
		return nil, ErrIncorrectSeed
	}

	// we basically use each single bit of the seed
	// to generate an unique IP range
	host := 0b10101100000100000000000000000000 // 172.16.0.0/12

	// 20 bits are free: 16 are used for the seed, 1 for the network
	// we can shift 1-4 as needed to minimize clashes (and could insert 3 static bits)
	seed <<= 4
	host |= seed
	ns := host | 1

	hostIP := make(net.IP, 4)
	nsIP := make(net.IP, 4)

	binary.BigEndian.PutUint32(hostIP, uint32(host))
	binary.BigEndian.PutUint32(nsIP, uint32(ns))

	for _, network := range hostNetworks {
		if network.Contains(hostIP) {
			return nil, ErrIPClash
		}
	}

	topology := make([]string, 3)
	topology[0] = fmt.Sprintf("%s|%s", fmt.Sprintf("kola-%d%d%d", nsIP[1], nsIP[2], nsIP[3]), fmt.Sprintf("%s/31", nsIP.String()))
	topology[1] = fmt.Sprintf("%s|%s", fmt.Sprintf("kola-%d%d%d", hostIP[1], hostIP[2], hostIP[3]), fmt.Sprintf("%s/31", hostIP.String()))
	topology[2] = fmt.Sprintf("%s/32", hostIP.String())

	return topology, nil
}

// setupLink creates a new network device, assigns it
// an address and finally set it up
func setupLink(l netlink.Link, addr string, add bool) error {
	if add {
		if err := netlink.LinkAdd(l); err != nil {
			return fmt.Errorf("unable to add link: %w", err)
		}
	}

	ad, err := netlink.ParseAddr(addr)
	if err != nil {
		return fmt.Errorf("unable to parse addr: %w", err)
	}

	if err := netlink.AddrAdd(l, ad); err != nil {
		return fmt.Errorf("unable to add address: %w", err)
	}

	if err := netlink.LinkSetUp(l); err != nil {
		return fmt.Errorf("unable to set link up: %w", err)
	}

	return nil
}

func newSegment(s byte) (*Segment, error) {
	seg := &Segment{
		BridgeName: fmt.Sprintf("br%d", s),
		BridgeIf:   newInterface(s, 1),
	}

	for i := uint16(2); i < 2+numInterfaces; i++ {
		seg.Interfaces = append(seg.Interfaces, newInterface(s, i))
	}

	br := netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:         seg.BridgeName,
			HardwareAddr: seg.BridgeIf.HardwareAddr,
		},
	}

	if err := netlink.LinkAdd(&br); err != nil {
		return nil, fmt.Errorf("LinkAdd() failed: %v", err)
	}

	for _, addr := range seg.BridgeIf.DHCPv4 {
		nladdr := netlink.Addr{IPNet: &addr}
		if err := netlink.AddrAdd(&br, &nladdr); err != nil {
			return nil, fmt.Errorf("DHCPv4 AddrAdd() failed: %v", err)
		}
	}

	for _, addr := range seg.BridgeIf.DHCPv6 {
		nladdr := netlink.Addr{IPNet: &addr}
		if err := netlink.AddrAdd(&br, &nladdr); err != nil {
			return nil, fmt.Errorf("DHCPv6 AddrAdd() failed: %v", err)
		}
	}

	if err := netlink.LinkSetUp(&br); err != nil {
		return nil, fmt.Errorf("LinkSetUp() failed: %v", err)
	}

	// we first create an unique virtual ethernet pair in the root network namespace
	// we use linux network random port attribution to assert the uniqueness of the IP range in order to avoid IP
	// range clashes
	root, err := netns.GetFromPid(1)
	if err != nil {
		return nil, fmt.Errorf("unable to get NS from PID 1: %w", err)
	}

	rootExit, err := ns.Enter(root)
	if err != nil {
		return nil, fmt.Errorf("unable to enter root namespace: %w", err)
	}

	var (
		listener net.Listener
		// isValid indicates whether or not the generated IP
		// is valid - meaning no conflict with existing
		// IP network
		isValid bool
		pair    []string
	)

	for !isValid {
		listener, err = net.Listen("tcp", ":0")
		if err != nil {
			return nil, fmt.Errorf("unable to listen on random port: %w", err)
		}

		_, port, err := net.SplitHostPort(listener.Addr().String())
		if err != nil {
			return nil, fmt.Errorf("unable to split address: %w", err)
		}

		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("unable to convert port to int: %w", err)
		}

		ips := make([]*net.IPNet, 0)
		links, err := netlink.LinkList()
		if err != nil {
			return nil, fmt.Errorf("unable to fetch network link list: %w", err)
		}

		for _, link := range links {
			addresses, err := netlink.AddrList(link, netlink.FAMILY_V4)
			if err != nil {
				return nil, fmt.Errorf("unable to list addresses for device: %w", err)
			}

			for _, address := range addresses {
				ips = append(ips, address.IPNet)
			}
		}

		pair, err = generateVethPair(p, ips)
		if err != nil {
			if errors.Is(err, ErrIPClash) {
				isValid = false
				plog.Debugf("failed to use port %d as unique seed for the veth pair due to address clash, retrying", p)
				if err := listener.Close(); err != nil {
					return nil, fmt.Errorf("unable to close TCP listener: %w", err)
				}
				continue
			} else {
				return nil, fmt.Errorf("unable to generate veth pair: %w", err)
			}
		}

		isValid = true
	}

	if err := rootExit(); err != nil {
		return nil, fmt.Errorf("unable to exit root namespace: %w", err)
	}

	// keep the created listener for destroying later
	seg.Listener = listener

	peer0 := strings.Split(pair[0], "|")
	peer1 := strings.Split(pair[1], "|")

	attr := netlink.NewLinkAttrs()
	attr.Name = peer0[0]
	veth := &netlink.Veth{
		PeerName:  peer1[0],
		LinkAttrs: attr,
	}
	if err := setupLink(veth, peer0[1], true); err != nil {
		return nil, fmt.Errorf("unable to set up link: %w", err)
	}

	peer, err := netlink.LinkByName(peer1[0])
	if err != nil {
		return nil, fmt.Errorf("unable to get link by name: %w", err)
	}

	// move to root network namespace
	if err := netlink.LinkSetNsPid(peer, 1); err != nil {
		return nil, fmt.Errorf("unable to set link into a new ns: %w\n", err)
	}

	gtw, _, err := net.ParseCIDR(pair[2])
	if err != nil {
		return nil, fmt.Errorf("unable to parse CIDR address: %w", err)
	}

	_, dst, err := net.ParseCIDR("0.0.0.0/0")
	if err != nil {
		return nil, fmt.Errorf("unable to parse CIDR address: %w", err)
	}

	if err := netlink.RouteAdd(&netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		Dst:       dst,
		LinkIndex: veth.Attrs().Index,
		Gw:        gtw,
	}); err != nil {
		return nil, fmt.Errorf("unable to add default route: %w", err)
	}

	if err := configureNAT(peer0[0]); err != nil {
		return nil, fmt.Errorf("unable to configure NAT: %w", err)
	}

	root, err = netns.GetFromPid(1)
	if err != nil {
		return nil, fmt.Errorf("unable to get NS from PID 1: %w", err)
	}

	rootExit, err = ns.Enter(root)
	if err != nil {
		return nil, fmt.Errorf("unable to enter root namespace: %w", err)
	}

	peer, err = netlink.LinkByName(peer1[0])
	if err != nil {
		return nil, fmt.Errorf("unable to get link by name: %w", err)
	}

	if err := setupLink(peer, peer1[1], false); err != nil {
		return nil, fmt.Errorf("unable to set up link: %w", err)
	}

	// `ip route get 1.0.0.0` in order to get device to configure NAT on
	d := net.IPv4(1, 0, 0, 0)

	routes, err := netlink.RouteGet(d)
	if err != nil {
		return nil, fmt.Errorf("unable to get routes: %w", err)
	}

	if len(routes) < 0 {
		return nil, fmt.Errorf("at least one default route is required")
	}

	route := routes[0]
	l, err := netlink.LinkByIndex(route.LinkIndex)
	if err != nil {
		return nil, fmt.Errorf("unable to get link by index: %w", err)
	}

	if err := configureNAT(l.Attrs().Name); err != nil {
		return nil, fmt.Errorf("unable to configure NAT: %w", err)
	}

	if err := rootExit(); err != nil {
		return nil, fmt.Errorf("unable to exit root namespace: %w", err)
	}

	return seg, nil
}

func NewDnsmasq() (*Dnsmasq, error) {
	dm := &Dnsmasq{}
	for s := byte(0); s < numSegments; s++ {
		seg, err := newSegment(s)
		if err != nil {
			return nil, fmt.Errorf("Network setup failed: %v", err)
		}
		dm.Segments = append(dm.Segments, seg)
	}

	// setup lo
	lo, err := netlink.LinkByName("lo")
	if err != nil {
		return nil, fmt.Errorf("Network loopback setup failed: %v", err)
	}
	err = netlink.LinkSetUp(lo)
	if err != nil {
		return nil, fmt.Errorf("Network loopback setup failed: %v", err)
	}

	dm.dnsmasq = exec.Command("dnsmasq", "--conf-file=-")
	cfg, err := dm.dnsmasq.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := dm.dnsmasq.StdoutPipe()
	if err != nil {
		return nil, err
	}
	dm.dnsmasq.Stderr = dm.dnsmasq.Stdout
	go util.LogFrom(capnslog.INFO, out)

	if err = dm.dnsmasq.Start(); err != nil {
		cfg.Close()
		return nil, err
	}

	plog.Debugf("dnsmasq PID (manual cleanup needed if --remove=false): %v", dm.dnsmasq.Pid())

	var configTemplate *template.Template

	if plog.LevelAt(capnslog.DEBUG) {
		configTemplate = template.Must(
			template.New("dnsmasq").Parse(debugConfig + commonConfig))
	} else {
		configTemplate = template.Must(
			template.New("dnsmasq").Parse(quietConfig + commonConfig))
	}

	if err = configTemplate.Execute(cfg, dm); err != nil {
		cfg.Close()
		dm.Destroy()
		return nil, err
	}
	cfg.Close()

	return dm, nil
}

func (dm *Dnsmasq) GetInterface(bridge string) (in *Interface) {
	for _, seg := range dm.Segments {
		if bridge == seg.BridgeName {
			if seg.nextIf >= len(seg.Interfaces) {
				panic("Not enough interfaces!")
			}
			in = seg.Interfaces[seg.nextIf]
			seg.nextIf++
			return
		}
	}
	panic("Not a valid bridge!")
}

func (dm *Dnsmasq) Destroy() {
	if err := dm.dnsmasq.Kill(); err != nil {
		plog.Errorf("Error killing dnsmasq: %v", err)
	}

	for _, seg := range dm.Segments {
		if err := seg.Listener.Close(); err != nil {
			plog.Errorf("unable to close segment listener: %v", err)
		}
	}
}
