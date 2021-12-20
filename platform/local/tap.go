// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"bytes"
	"os"
	"syscall"
	"unsafe"

	"github.com/vishvananda/netlink"
)

const (
	tunDevice = "/dev/net/tun"
)

// Tun/Tap device that is compatible with the netlink library.
type TunTap struct {
	*netlink.LinkAttrs
	*os.File
}

func (tt *TunTap) Attrs() *netlink.LinkAttrs {
	return tt.LinkAttrs
}

func (tt *TunTap) Type() string {
	return "tun"
}

type ifreqFlags struct {
	IfrnName  [syscall.IFNAMSIZ]byte
	IfruFlags uint16
}

func ioctl(fd, request, argp uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, argp)
	if errno != 0 {
		return errno
	}
	return nil
}

func fromZeroTerm(s []byte) string {
	return string(bytes.TrimRight(s, "\000"))
}

func newTunTap(name string, flags uint16) (*TunTap, error) {
	dev, err := os.OpenFile(tunDevice, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	var ifr ifreqFlags
	copy(ifr.IfrnName[:len(ifr.IfrnName)-1], []byte(name+"\000"))
	ifr.IfruFlags = flags | syscall.IFF_NO_PI

	err = ioctl(dev.Fd(), syscall.TUNSETIFF, uintptr(unsafe.Pointer(&ifr)))
	if err != nil {
		return nil, err
	}

	ifname := fromZeroTerm(ifr.IfrnName[:len(ifr.IfrnName)-1])
	iflink, err := netlink.LinkByName(ifname)
	if err != nil {
		dev.Close()
		return nil, err
	}

	tt := TunTap{
		File:      dev,
		LinkAttrs: iflink.Attrs(),
	}

	return &tt, nil
}

func AddLinkTap(name string) (*TunTap, error) {
	return newTunTap(name, syscall.IFF_TAP)
}

func AddLinkTun(name string) (*TunTap, error) {
	return newTunTap(name, syscall.IFF_TUN)
}
