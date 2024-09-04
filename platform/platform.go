// Copyright 2017 CoreOS, Inc.
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

package platform

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/coreos/pkg/capnslog"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"

	"github.com/flatcar/mantle/platform/conf"
	"github.com/flatcar/mantle/util"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform")
)

// Name is a unique identifier for a platform.
type Name string

// Machine represents a Container Linux instance.
type Machine interface {
	// ID returns the plaform-specific machine identifier.
	ID() string

	// IP returns the machine's public IP.
	IP() string

	// PrivateIP returns the machine's private IP.
	PrivateIP() string

	// RuntimeConf returns the cluster's runtime configuration.
	RuntimeConf() *RuntimeConfig

	// SSHClient establishes a new SSH connection to the machine.
	SSHClient() (*ssh.Client, error)

	// PasswordSSHClient establishes a new SSH connection using the provided credentials.
	PasswordSSHClient(user string, password string) (*ssh.Client, error)

	// SSH runs a single command over a new SSH connection.
	SSH(cmd string) ([]byte, []byte, error)

	// Reboot restarts the machine and waits for it to come back.
	Reboot() error

	// Destroy terminates the machine and frees associated resources. It should log
	// any failures; since they are not actionable, it does not return an error.
	Destroy()

	// ConsoleOutput returns the machine's console output if available,
	// or an empty string.  Only expected to be valid after Destroy().
	ConsoleOutput() string

	// JournalOutput returns the machine's journal output if available,
	// or an empty string.  Only expected to be valid after Destroy().
	JournalOutput() string

	// Board returns the machine's board
	Board() string
}

// Cluster represents a cluster of machines within a single Flight.
type Cluster interface {
	// Platform returns the name of the platform.
	Platform() Name

	// Name returns a unique name for the Cluster.
	Name() string

	// NewMachine creates a new Container Linux machine.
	NewMachine(userdata *conf.UserData) (Machine, error)

	// Machines returns a slice of the active machines in the Cluster.
	Machines() []Machine

	// GetDiscoveryURL returns a new etcd discovery URL.
	GetDiscoveryURL(size int) (string, error)

	// Destroy terminates each machine in the cluster and frees any other
	// associated resources. It should log any failures; since they are not
	// actionable, it does not return an error
	Destroy()

	// ConsoleOutput returns a map of console output from destroyed
	// cluster machines.
	ConsoleOutput() map[string]string

	// JournalOutput returns a map of journal output from destroyed
	// cluster machines.
	JournalOutput() map[string]string

	// IgnitionVersion returns the version of Ignition supported by the
	// cluster
	IgnitionVersion() string

	// RuntimeConf returns a pointer to the runtime configuration.
	RuntimeConf() *RuntimeConfig
}

// Flight represents a group of Clusters within a single platform.
type Flight interface {
	// NewCluster creates a new Cluster.
	NewCluster(rconf *RuntimeConfig) (Cluster, error)

	// Name returns a unique name for the Flight.
	Name() string

	// Platform returns the name of the platform.
	Platform() Name

	// Clusters returns a slice of the active Clusters.
	Clusters() []Cluster

	// Destroy terminates each cluster and frees any other associated
	// resources.  It should log any failures; since they are not
	// actionable, it does not return an error.
	Destroy()

	GetBaseFlight() *BaseFlight
}

// SystemdDropin is a userdata type agnostic struct representing a systemd dropin
type SystemdDropin struct {
	Unit     string
	Name     string
	Contents string
}

// Options contains the base options for all clusters.
type Options struct {
	BaseName        string
	Distribution    string
	IgnitionVersion string
	SystemdDropins  []SystemdDropin

	// OSContainer is an image pull spec that can be given to the pivot service
	// in RHCOS machines to perform machine content upgrades.
	// When specified additional files & units will be automatically generated
	// inside of RenderUserData
	OSContainer string

	// Board is the board used by the image
	Board string

	// Toggle to instantiate a secureboot instance.
	EnableSecureboot bool

	// How many times to retry establishing an SSH connection when
	// creating a journal or when doing a machine check.
	SSHRetries int
	// A duration of a single try of establishing the connection
	// when creating a journal or when doing a machine check.
	SSHTimeout time.Duration
}

// RuntimeConfig contains cluster-specific configuration.
type RuntimeConfig struct {
	OutputDir string

	NoSSHKeyInUserData bool          // don't inject SSH key into Ignition/cloud-config
	NoSSHKeyInMetadata bool          // don't add SSH key to platform metadata
	NoEnableSelinux    bool          // don't enable selinux when starting or rebooting a machine
	NoDisableUpdates   bool          // don't disable usage of the public update server
	AllowFailedUnits   bool          // don't fail CheckMachine if a systemd unit has failed
	SSHRetries         int           // see SSHRetries field in Options
	SSHTimeout         time.Duration // see SSHTimeout field in Options

	// DefaultUser is the user used for SSH connection, it will be created via Ignition when possible.
	DefaultUser string
}

// Wrap a StdoutPipe as a io.ReadCloser
type sshPipe struct {
	s   *ssh.Session
	c   *ssh.Client
	err *bytes.Buffer
	io.Reader
}

func (p *sshPipe) Close() error {
	if err := p.s.Wait(); err != nil {
		return fmt.Errorf("%s: %s", err, p.err)
	}
	if err := p.s.Close(); err != nil {
		return err
	}
	return p.c.Close()
}

// Copy a file between two machines in a cluster.
func TransferFile(src Machine, srcPath string, dst Machine, dstPath string) error {
	srcPipe, err := ReadFile(src, srcPath)
	if err != nil {
		return err
	}
	defer srcPipe.Close()

	if err := InstallFile(srcPipe, dst, dstPath); err != nil {
		return err
	}
	return nil
}

// ReadFile returns a io.ReadCloser that streams the requested file. The
// caller should close the reader when finished.
func ReadFile(m Machine, path string) (io.ReadCloser, error) {
	client, err := m.SSHClient()
	if err != nil {
		return nil, fmt.Errorf("failed creating SSH client: %v", err)
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed creating SSH session: %v", err)
	}

	// connect session stdout
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, err
	}

	// collect stderr
	errBuf := bytes.NewBuffer(nil)
	session.Stderr = errBuf

	// stream file to stdout
	err = session.Start(fmt.Sprintf("sudo cat %s", path))
	if err != nil {
		session.Close()
		client.Close()
		return nil, err
	}

	// pass stdoutPipe as a io.ReadCloser that cleans up the ssh session
	// on when closed.
	return &sshPipe{session, client, errBuf, stdoutPipe}, nil
}

// InstallFile copies data from in to the path to on m.
func InstallFile(in io.Reader, m Machine, to string) error {
	dir := filepath.Dir(to)
	out, stderr, err := m.SSH(fmt.Sprintf("sudo mkdir -p %s", dir))
	if err != nil {
		return fmt.Errorf("failed creating directory %s: %s: %s", dir, stderr, err)
	}

	client, err := m.SSHClient()
	if err != nil {
		return fmt.Errorf("failed creating SSH client: %v", err)
	}

	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed creating SSH session: %v", err)
	}

	defer session.Close()

	// write file to fs from stdin
	session.Stdin = in
	out, err = session.CombinedOutput(fmt.Sprintf("sudo install -m 0755 /dev/stdin %s", to))
	if err != nil {
		return fmt.Errorf("failed executing install: %q: %v", out, err)
	}

	return nil
}

// NewMachines spawns n instances in cluster c, with
// each instance passed the same userdata.
func NewMachines(c Cluster, userdata *conf.UserData, n int) ([]Machine, error) {
	var wg sync.WaitGroup

	mchan := make(chan Machine, n)
	errchan := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, err := c.NewMachine(userdata)
			if err != nil {
				errchan <- err
			}
			if m != nil {
				mchan <- m
			}
		}()
	}

	wg.Wait()
	close(mchan)
	close(errchan)

	machs := []Machine{}

	for m := range mchan {
		machs = append(machs, m)
	}

	if firsterr, ok := <-errchan; ok {
		for _, m := range machs {
			m.Destroy()
		}
		return nil, firsterr
	}

	return machs, nil
}

// CheckMachine tests a machine for various error conditions such as ssh
// being available and no systemd units failing at the time ssh is reachable.
// It also ensures the remote system is running Flatcar Container Linux.
//
// TODO(mischief): better error messages.
func CheckMachine(ctx context.Context, m Machine) error {
	// ensure ssh works and the system is ready
	sshChecker := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		out, stderr, err := m.SSH("systemctl is-system-running")
		if !bytes.Contains([]byte("initializing starting running stopping"), out) {
			return nil // stop retrying if the system went haywire, e.g., "degraded"
		}
		jobs := ""
		if bytes.Contains([]byte("starting"), out) {
			startingOut, startingStderr, startingErr := m.SSH("systemctl list-jobs")
			jobs = fmt.Sprintf(", systemctl list-jobs returned stdout: %q, stderr: %q, err: %v", startingOut, startingStderr, startingErr)
		}
		// For "running" the exit code is 0 thus err is nil but not for, e.g., "starting" where the exit code is 1
		if err != nil {
			return fmt.Errorf("failure checking if machine is running: systemctl is-system-running returned stdout: %q, stderr: %q, err: %v%s", out, stderr, err, jobs)
		}
		return nil
	}

	rc := m.RuntimeConf()
	if err := util.Retry(rc.SSHRetries, rc.SSHTimeout, sshChecker); err != nil {
		return fmt.Errorf("ssh unreachable or system not ready: %v", err)
	}

	// ensure we're talking to a Container Linux system
	out, stderr, err := m.SSH("grep ^ID= /etc/os-release")
	if err != nil {
		return fmt.Errorf("no /etc/os-release file: %v: %s", err, stderr)
	}

	if !bytes.Equal(out, []byte("ID=flatcar")) {
		return fmt.Errorf("not a Flatcar Container Linux instance")
	}

	if !m.RuntimeConf().AllowFailedUnits {
		// ensure no systemd units failed during boot
		out, stderr, err = m.SSH("systemctl --no-legend --state failed list-units")
		if err != nil {
			return fmt.Errorf("systemctl: %s: %v: %s", out, err, stderr)
		}
		if len(out) > 0 {
			unit := strings.Fields(string(out))
			info := ""
			if 0 < len(unit) {
				j, _, _ := m.SSH(fmt.Sprintf("journalctl -b -u %s", unit[0]))
				s, _, _ := m.SSH(fmt.Sprintf("systemctl status %s", unit[0]))
				info = fmt.Sprintf("\nstatus: %s\njournal:%s", s, j)
			}
			return fmt.Errorf("some systemd units failed:\n%s%s", out, info)
		}
	}

	return ctx.Err()
}
