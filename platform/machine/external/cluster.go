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
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/conf"
)

type cluster struct {
	*platform.BaseCluster
	flight *flight
}

func (pc *cluster) NewMachine(userdata *conf.UserData) (platform.Machine, error) {
	conf, err := pc.RenderUserData(userdata, map[string]string{
		"$public_ipv4":  "${COREOS_CUSTOM_PUBLIC_IPV4}",
		"$private_ipv4": "${COREOS_CUSTOM_PRIVATE_IPV4}",
	})
	if err != nil {
		return nil, err
	}
	// This assumes that private and public IP addresses are the same (i.e., no public IP addr) on the interface that has the default route
	conf.AddSystemdUnit("coreos-metadata.service", `[Unit]
Description=Custom metadata agent
After=nss-lookup.target
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
Environment=OUTPUT=/run/metadata/flatcar
ExecStart=/usr/bin/mkdir --parent /run/metadata
ExecStart=/usr/bin/bash -c 'echo "COREOS_CUSTOM_PRIVATE_IPV4=$(ip addr show $(ip route get 1 | head -n 1 | cut -d ' ' -f 5) | grep -m 1 -Po "inet \K[\d.]+")\nCOREOS_CUSTOM_PUBLIC_IPV4=$(ip addr show $(ip route get 1 | head -n 1 | cut -d ' ' -f 5) | grep -m 1 -Po "inet \K[\d.]+")" > ${OUTPUT}'
ExecStartPost=/usr/bin/ln -fs /run/metadata/flatcar /run/metadata/coreos
`, false)

	var cons *console
	var pcons Console // need a nil interface value if unused
	var ipAddr string
	// Do not shadow assignments to err (i.e., use a, err := something) in the for loop
	// because the "continue" case needs to access the previous error to return it when the
	// maximal number of retries is reached or to print it at the beginning of the loop.
	for retry := 0; retry <= 2; retry++ {
		if err != nil {
			plog.Warningf("Retrying to provision a machine after error: %q", err)
		}
		// Stream the console somewhere temporary until we have a machine ID
		b := make([]byte, 5)
		rand.Read(b)
		randName := fmt.Sprintf("%x", b)
		consolePath := filepath.Join(pc.RuntimeConf().OutputDir, "console-"+pc.Name()[0:13]+"-"+randName+".txt")
		var f *os.File
		f, err = os.OpenFile(consolePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			return nil, err
		}
		cons = &console{
			pc:   pc,
			f:    f,
			done: make(chan interface{}),
		}
		pcons = cons

		// CreateDevice unconditionally closes console when done with it
		ipAddr, err = pc.createDevice(conf, pcons)
		if err != nil {
			continue // provisioning error
		}

		mach := &machine{
			cluster: pc,
			ipAddr:  ipAddr,
			console: cons,
			rand:    randName,
		}

		dir := filepath.Join(pc.RuntimeConf().OutputDir, mach.ID())
		if err = os.Mkdir(dir, 0777); err != nil {
			mach.Destroy()
			return nil, err
		}

		if cons != nil {
			if err = os.Rename(consolePath, filepath.Join(dir, "console.txt")); err != nil {
				mach.Destroy()
				return nil, err
			}
		}

		confPath := filepath.Join(dir, "user-data")
		if err = conf.WriteFile(confPath); err != nil {
			mach.Destroy()
			return nil, err
		}

		if mach.journal, err = platform.NewJournal(dir); err != nil {
			mach.Destroy()
			return nil, err
		}

		plog.Infof("Starting machine %v", mach.ID())
		if err = platform.StartMachine(mach, mach.journal); err != nil {
			mach.Destroy()
			continue // provisioning error
		}

		pc.AddMach(mach)

		return mach, nil

	}

	return nil, err
}

func setEnvCmd(varname, content string) string {
	quoted := strings.ReplaceAll(content, `'`, `'"'"'`)
	// include the final ; for concatenation with a following command
	return varname + `='` + quoted + `';`
}

func (pc *cluster) createDevice(conf *conf.Conf, console Console) (string, error) {
	plog.Info("Creating machine")
	consoleStarted := false
	defer func() {
		if console != nil && !consoleStarted {
			console.Close()
		}
	}()

	userdata := conf.String()
	session, err := pc.flight.ManagementSSHClient.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	output, err := session.Output(setEnvCmd("USERDATA", userdata) + pc.flight.ExternalOptions.ProvisioningCmds)
	if err != nil {
		return "", err
	}
	ipAddr := strings.TrimSpace(string(output))
	if net.ParseIP(ipAddr) == nil {
		return "", fmt.Errorf("script output %q is not a valid IP address", ipAddr)
	}
	plog.Infof("Got IP address %v", ipAddr)
	if console != nil {
		err := pc.startConsole(ipAddr, console)
		// console will be closed in any case
		consoleStarted = true
		if err != nil {
			err2 := pc.deleteDevice(ipAddr)
			if err2 != nil {
				return "", fmt.Errorf("couldn't delete device %s after error %q: %v", ipAddr, err, err2)
			}
			return "", err
		}
	}
	return ipAddr, nil
}

func (pc *cluster) deleteDevice(ipAddr string) error {
	plog.Infof("Deleting machine %v", ipAddr)
	session, err := pc.flight.ManagementSSHClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	err = session.Run(setEnvCmd("IPADDR", ipAddr) + pc.flight.ExternalOptions.DeprovisioningCmds)
	if err != nil {
		return err
	}
	return nil
}

func (pc *cluster) startConsole(ipAddr string, console Console) error {
	plog.Infof("Attaching serial console for %v", ipAddr)
	ready := make(chan error)

	runner := func() error {
		defer console.Close()
		session, err := pc.flight.ManagementSSHClient.NewSession()
		if err != nil {
			return fmt.Errorf("couldn't create SSH session for %s console: %v", ipAddr, err)
		}
		defer session.Close()

		reader, writer := io.Pipe()
		defer writer.Close()

		session.Stdin = reader
		session.Stdout = console
		if err := session.Start(setEnvCmd("IPADDR", ipAddr) + pc.flight.ExternalOptions.SerialConsoleCmd); err != nil {
			return fmt.Errorf("couldn't start provided serial attach command for %s console: %v", ipAddr, err)
		}

		// cause startConsole to return
		ready <- nil

		err = session.Wait()
		_, ok := err.(*ssh.ExitMissingError)
		if err != nil && !ok {
			plog.Errorf("%s console session failed: %v", ipAddr, err)
		}
		return nil
	}
	go func() {
		err := runner()
		if err != nil {
			ready <- err
		}
	}()

	return <-ready
}

func (pc *cluster) Destroy() {
	pc.BaseCluster.Destroy()
	pc.flight.DelCluster(pc)
}
