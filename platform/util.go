// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package platform

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

// Manhole connects os.Stdin, os.Stdout, and os.Stderr to an interactive shell
// session on the Machine m. Manhole blocks until the shell session has ended.
// If os.Stdin does not refer to a TTY, Manhole returns immediately with a nil
// error.
func Manhole(m Machine) error {
	fd := int(os.Stdin.Fd())
	if !terminal.IsTerminal(fd) {
		return nil
	}

	tstate, _ := terminal.MakeRaw(fd)
	defer terminal.Restore(fd, tstate)

	client, err := m.SSHClient()
	if err != nil {
		return fmt.Errorf("SSH client failed: %v", err)
	}

	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("SSH session failed: %v", err)
	}

	defer session.Close()

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	modes := ssh.TerminalModes{
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}

	cols, lines, err := terminal.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	if err = session.RequestPty(os.Getenv("TERM"), lines, cols, modes); err != nil {
		return fmt.Errorf("failed to request pseudo terminal: %s", err)
	}

	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %s", err)
	}

	if err := session.Wait(); err != nil {
		return fmt.Errorf("failed to wait for session: %s", err)
	}

	return nil
}

// Enable SELinux on a machine (skip on machines without SELinux support)
func EnableSelinux(m Machine) error {
	_, stderr, err := m.SSH("sudo setenforce 1")
	if err != nil {
		return fmt.Errorf("Unable to enable SELinux: %s: %s", err, stderr)
	}

	// remove audit rules to get SELinux AVCs in the audit logs
	_, stderr, err = m.SSH("sudo rm -rf /etc/audit/rules.d/{80-selinux.rules,99-default.rules}; sudo systemctl restart audit-rules")
	if err != nil {
		return fmt.Errorf("unable to enable SELinux audit logs: %s: %s", err, stderr)
	}

	return nil
}

// Reboots a machine, stopping ssh first.
// Afterwards run CheckMachine to verify the system is back and operational.
func StartReboot(m Machine) error {
	// stop sshd so that commonMachineChecks will only work if the machine
	// actually rebooted
	out, stderr, err := m.SSH("sudo systemctl stop sshd.socket && sudo reboot")
	if _, ok := err.(*ssh.ExitMissingError); ok {
		// A terminated session is perfectly normal during reboot.
		err = nil
	}
	if err != nil {
		return fmt.Errorf("issuing reboot command failed: %s: %s: %s", out, err, stderr)
	}
	return nil
}

// RebootMachine will reboot a given machine, provided the machine's journal.
func RebootMachine(m Machine, j *Journal) error {
	if err := StartReboot(m); err != nil {
		return fmt.Errorf("machine %q failed to begin rebooting: %v", m.ID(), err)
	}
	return StartMachine(m, j)
}

// StartMachine will start a given machine, provided the machine's journal.
func StartMachine(m Machine, j *Journal) error {
	if err := j.Start(context.TODO(), m); err != nil {
		return fmt.Errorf("machine %q failed to start: %v", m.ID(), err)
	}
	if err := CheckMachine(context.TODO(), m); err != nil {
		return fmt.Errorf("machine %q failed basic checks: %v", m.ID(), err)
	}
	if !m.RuntimeConf().NoEnableSelinux {
		if err := EnableSelinux(m); err != nil {
			return fmt.Errorf("machine %q failed to enable selinux: %v", m.ID(), err)
		}
	}
	return nil
}

// GenerateFakeKey generates a SSH key pair, returns the public key, and
// discards the private key. This is useful for droplets that don't need a
// public key, since DO & Azure insists on requiring one.
func GenerateFakeKey() (string, error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	sshKey, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return "", err
	}
	return string(ssh.MarshalAuthorizedKey(sshKey)), nil
}
