// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
)

type cluster struct {
	*platform.BaseCluster
	flight *flight
}

func (bc *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	conf, err := bc.RenderUserData(userdata, map[string]string{
		"$public_ipv4":  "${COREOS_CUSTOM_PUBLIC_IPV4}",
		"$private_ipv4": "${COREOS_CUSTOM_PRIVATE_IPV4}",
	})
	if err != nil {
		return nil, err
	}

	conf.AddSystemdUnitDropin("coreos-metadata.service", "00-custom-metadata.conf", `[Service]
ExecStartPost=/usr/bin/sh -c 'ip=$(ip -json -4 addr show $(ip -json route get 1 | jq -r '.[0].dev') | jq -r .[0].addr_info.[0].local); printf "COREOS_CUSTOM_PRIVATE_IPV4=%%s\nCOREOS_CUSTOM_PUBLIC_IPV4=%%s\n" "$ip" "$ip" >> /run/metadata/flatcar'
`)

	var sshAuthorizedKeys string
	if !bc.RuntimeConf().NoSSHKeyInMetadata {
		keys, err := bc.Keys()
		if err != nil {
			return nil, err
		}
		keyStrings := make([]string, 0, len(keys))
		for _, key := range keys {
			keyStrings = append(keyStrings, key.String())
		}
		sshAuthorizedKeys = strings.Join(keyStrings, "\n")
	}

	instance, err := bc.flight.api.CreateInstance(context.TODO(), bc.vmname(), conf.String(), sshAuthorizedKeys)
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
	if err := os.Mkdir(mach.dir, 0777); err != nil {
		return nil, err
	}

	confPath := filepath.Join(mach.dir, "ignition.json")
	if err := conf.WriteFile(confPath); err != nil {
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

func (bc *cluster) vmname() string {
	b := make([]byte, 5)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", bc.Name()[0:13], b)
}

func (bc *cluster) Destroy() {
	bc.BaseCluster.Destroy()
	bc.flight.DelCluster(bc)
}
