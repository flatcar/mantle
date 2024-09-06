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

package qemu

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pborman/uuid"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/platform/local"
	"github.com/flatcar/mantle/system/ns"
)

// Cluster is a local cluster of QEMU-based virtual machines.
//
// XXX: must be exported so that certain QEMU tests can access struct members
// through type assertions.
type Cluster struct {
	flight *flight

	mu sync.Mutex
	*local.LocalCluster
}

func (qc *Cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	options := platform.MachineOptions{
		ExtraPrimaryDiskSize: qc.flight.opts.ExtraBaseDiskSize,
		// Use for 'kola spawn'; test cases should pass true through
		// NewMachineWithOptions()
		EnableTPM: qc.flight.opts.EnableTPM,
		VNC:       qc.flight.opts.VNC,
	}
	return qc.NewMachineWithOptions(userdata, options)
}

func (qc *Cluster) NewMachineWithOptions(userdata *conf.UserData, options platform.MachineOptions) (platform.Machine, error) {
	id := uuid.New()

	dir := filepath.Join(qc.RuntimeConf().OutputDir, id)
	if err := os.Mkdir(dir, 0777); err != nil {
		return nil, err
	}

	// hacky solution for cloud config ip substitution
	// NOTE: escaping is not supported
	qc.mu.Lock()
	netif := qc.flight.Dnsmasq.GetInterface("br0")
	ip := strings.Split(netif.DHCPv4[0].String(), "/")[0]

	conf, err := qc.RenderUserData(userdata, map[string]string{
		"$public_ipv4":  "${COREOS_CUSTOM_PUBLIC_IPV4}",
		"$private_ipv4": "${COREOS_CUSTOM_PRIVATE_IPV4}",
	})
	if err != nil {
		qc.mu.Unlock()
		return nil, err
	}
	qc.mu.Unlock()

	conf.AddSystemdUnit("coreos-metadata.service", `[Unit]
Description=QEMU metadata agent
After=nss-lookup.target
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
Environment=OUTPUT=/run/metadata/flatcar
ExecStart=/usr/bin/mkdir --parent /run/metadata
ExecStart=/usr/bin/bash -c 'echo "COREOS_CUSTOM_PRIVATE_IPV4=`+ip+`\nCOREOS_CUSTOM_PUBLIC_IPV4=`+ip+`\n" > ${OUTPUT}'
ExecStartPost=/usr/bin/ln -fs /run/metadata/flatcar /run/metadata/coreos
`, false)

	// confPath is relative to the machine folder
	var confPath string
	if conf.IsIgnition() {
		confPath = "ignition.json"
		if err := conf.WriteFile(filepath.Join(dir, confPath)); err != nil {
			return nil, err
		}
	} else {
		confPath, err = local.MakeConfigDrive(conf, dir)
		if err != nil {
			return nil, err
		}
	}

	journal, err := platform.NewJournal(dir)
	if err != nil {
		return nil, err
	}

	qm := &machine{
		qc:          qc,
		id:          id,
		netif:       netif,
		journal:     journal,
		consolePath: "console.txt",
		subDir:      dir,
	}

	var swtpm *local.SoftwareTPM
	if options.EnableTPM {
		swtpm, err = local.NewSwtpm(qm.subDir, "tpm")
		if err != nil {
			return nil, fmt.Errorf("starting swtpm: %v", err)
		}
		options.SoftwareTPMSocket = swtpm.SocketRelativePathFromTestDir()
		defer func() {
			if swtpm != nil {
				swtpm.Stop()
			}
		}()
	}

	// This uses path arguments with path values being
	// relative to the folder created for this machine
	firmware, err := filepath.Abs(qc.flight.opts.Firmware)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize firmware path: %v", err)
	}
	ovmfVars := ""
	if qc.flight.opts.OVMFVars != "" {
		ovmfVars, err = platform.CreateOvmfVarsCopy(qm.subDir, qc.flight.opts.OVMFVars)
		if err != nil {
			return nil, err
		}
		defer func() {
			if ovmfVars != "" {
				os.Remove(path.Join(qm.subDir, ovmfVars))
			}
		}()
	}

	qmCmd, extraFiles, err := platform.CreateQEMUCommand(qc.flight.opts.Board, qm.id, firmware, ovmfVars, qm.consolePath, confPath, qc.flight.diskImagePath, qc.flight.opts.EnableSecureboot, conf.IsIgnition(), options)
	if err != nil {
		return nil, err
	}

	for _, file := range extraFiles {
		defer file.Close()
	}
	qmMac := qm.netif.HardwareAddr.String()

	qc.mu.Lock()

	tap, err := qc.NewTap("br0")
	if err != nil {
		qc.mu.Unlock()
		return nil, err
	}
	defer tap.Close()
	fdnum := 3 + len(extraFiles)
	qmCmd = append(qmCmd, "-netdev", fmt.Sprintf("tap,id=tap,fd=%d", fdnum),
		"-device", platform.Virtio(qc.flight.opts.Board, "net", "netdev=tap,mac="+qmMac))
	fdnum += 1
	extraFiles = append(extraFiles, tap.File)

	plog.Debugf("NewMachine: %q, cwd: %q, %q, %q", qmCmd, qm.subDir, qm.IP(), qm.PrivateIP())

	// Set qemu's current working directory to the machine folder
	// so that we can use short relative links for the UNIX sockets
	// without hitting the 108 char limit.
	qm.qemu = qm.qc.NewCommand(qm.subDir, qmCmd[0], qmCmd[1:]...)

	qc.mu.Unlock()

	cmd := qm.qemu.(*ns.Cmd)
	cmd.Stderr = os.Stderr

	cmd.ExtraFiles = append(cmd.ExtraFiles, extraFiles...)

	if err = qm.qemu.Start(); err != nil {
		return nil, err
	}

	// from this point on Destroy() is responsible for cleaning up swtpm
	qm.swtpm, swtpm = swtpm, nil
	qm.ovmfVars, ovmfVars = ovmfVars, ""
	plog.Debugf("qemu PID (manual cleanup needed if --remove=false): %v", qm.qemu.Pid())

	if err := platform.StartMachine(qm, qm.journal); err != nil {
		qm.Destroy()
		return nil, err
	}

	qc.AddMach(qm)

	return qm, nil
}

func (qc *Cluster) Destroy() {
	qc.LocalCluster.Destroy()
	qc.flight.DelCluster(qc)
}
