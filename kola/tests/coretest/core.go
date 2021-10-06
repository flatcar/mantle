package coretest

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pborman/uuid"

	"github.com/flatcar-linux/mantle/kola/register"
)

const (
	CmdTimeout                  = time.Second * 20
	DbusTimeout                 = time.Second * 20
	DockerTimeout               = time.Second * 60
	PortTimeout                 = time.Second * 3
	UpdateEnginePubKey          = "/usr/share/update_engine/update-payload-key.pub.pem"
	UpdateEnginePubKeyV1        = "d410d94dc56a1cba8df71c94ea6925811e44b09416f66958ab7a453f0731d80e"
	UpdateEnginePubKeyV2        = "a76a22e6afcdfbc55dd2953aa950c7ec93b254774fca02d13ec52c59672e5982"
	UpdateEnginePubKeyFlatcarV1 = "b59a0fa528fec10d706e5d48030218199568769e385dcf473aa696763331a353"
	UpdateEnginePubKeyFlatcarV2 = "b6a8227f835a4a56988241eaeeda57a19ef5ff413c8533fd791827be2c15dd6c"
)

// RHCOS services we expect disabled/inactive
var offServices = []string{
	"dnsmasq.service",
	"iscsid.service",
	"iscsid.socket",
	"iscsiuio.service",
	"nfs-blkmap.service",
	"nfs-idmapd.service",
	"nfs-mountd.service",
	"nfs-server.service",
	"nis-domainname.service",
	"rbdmap.service",
	"rdisc.service",
	"rpc-statd.service",
	"rpcbind.service",
	"rpcbind.socket",
	"sssd.service",
	"tcsd.service",
}

var symlinkMap = map[string]string{
	"/etc/coreos":             "/etc/flatcar",
	"/usr/bin/coreos-install": "/usr/bin/flatcar-install",
	"/usr/lib/coreos":         "/usr/lib/flatcar",
	"/usr/share/coreos":       "/usr/share/flatcar",
}

func init() {
	register.Register(&register.Test{
		Name:        "cl.basic",
		Run:         LocalTests,
		ClusterSize: 1,
		NativeFuncs: map[string]func() error{
			"CloudConfig":      TestCloudinitCloudConfig,
			"Script":           TestCloudinitScript,
			"PortSSH":          TestPortSsh,
			"DbusPerms":        TestDbusPerms,
			"Symlink":          TestSymlinkResolvConf,
			"SymlinkFlatcar":   TestSymlinkFlatcar,
			"UpdateEngineKeys": TestInstalledUpdateEngineRsaKeys,
			"ServicesActive":   TestServicesActive,
			"ReadOnly":         TestReadOnlyFs,
			"RandomUUID":       TestFsRandomUUID,
			"Useradd":          TestUseradd,
			"MachineID":        TestMachineID,
		},
		Distros: []string{"cl"},
	})
	register.Register(&register.Test{
		Name:        "rhcos.basic",
		Run:         LocalTests,
		ClusterSize: 1,
		NativeFuncs: map[string]func() error{
			"PortSSH":          TestPortSsh,
			"DbusPerms":        TestDbusPerms,
			"ServicesActive":   TestServicesActiveCoreOS,
			"ServicesDisabled": TestServicesDisabledRHCOS,
			"ReadOnly":         TestReadOnlyFs,
			"Useradd":          TestUseradd,
			"MachineID":        TestMachineID,
		},
		Distros: []string{"rhcos"},
	})
	register.Register(&register.Test{
		Name:        "fcos.basic",
		Run:         LocalTests,
		ClusterSize: 1,
		NativeFuncs: map[string]func() error{
			"PortSSH":        TestPortSsh,
			"DbusPerms":      TestDbusPerms,
			"ServicesActive": TestServicesActiveCoreOS,
			"ReadOnly":       TestReadOnlyFs,
			"Useradd":        TestUseradd,
			"MachineID":      TestMachineID,
		},
		Distros: []string{"fcos"},
	})

	// tests requiring network connection to internet
	register.Register(&register.Test{
		Name:        "cl.internet",
		Run:         InternetTests,
		ClusterSize: 1,
		NativeFuncs: map[string]func() error{
			"UpdateEngine": TestUpdateEngine,
			"DockerPing":   TestDockerPing,
			"DockerEcho":   TestDockerEcho,
			"NTPDate":      TestNTPDate,
		},
		Distros: []string{"cl"},
	})
}

func TestPortSsh() error {
	//t.Parallel()
	err := CheckPort("tcp", "127.0.0.1:22", PortTimeout)
	if err != nil {
		return err
	}
	return nil
}

func TestUpdateEngine() error {
	//t.Parallel()

	errc := make(chan error, 1)
	go func() {
		c := exec.Command("update_engine_client", "-status")
		err := c.Run()
		errc <- err
	}()

	select {
	case <-time.After(CmdTimeout):
		return fmt.Errorf("update_engine_client timed out after %s.", CmdTimeout)
	case err := <-errc:
		if err != nil {
			return err
		}
		return nil
	}

	// FIXME(marineam): Test DBus directly
}

func TestDockerEcho() error {
	//t.Parallel()
	errc := make(chan error, 1)
	go func() {
		c := exec.Command("docker", "run", "ghcr.io/kinvolk/busybox", "echo")
		err := c.Run()
		errc <- err
	}()
	select {
	case <-time.After(DockerTimeout):
		return fmt.Errorf("DockerEcho timed out after %s.", DockerTimeout)
	case err := <-errc:
		if err != nil {
			return fmt.Errorf("DockerEcho: %v", err)
		}
		return nil
	}
}

func TestDockerPing() error {
	//t.Parallel()
	errc := make(chan error, 1)
	go func() {
		c := exec.Command("docker", "run", "ghcr.io/kinvolk/busybox", "ping", "-c4", "coreos.com")
		err := c.Run()
		errc <- err
	}()
	select {
	case <-time.After(DockerTimeout):
		return fmt.Errorf("DockerPing timed out after %s.", DockerTimeout)
	case err := <-errc:
		if err != nil {
			return err
		}
		return nil
	}
}

func TestNTPDate() error {
	//t.Parallel()
	errc := make(chan error, 1)
	go func() {
		c := exec.Command("ntpdate", "-d", "-s", "-u", "pool.ntp.org")
		err := c.Run()
		errc <- err
	}()
	select {
	case <-time.After(CmdTimeout):
		return fmt.Errorf("ntpdate timed out after %s.", CmdTimeout)
	case err := <-errc:
		if err != nil {
			return err
		}
		return nil
	}
}

// This execs gdbus, because we need to change uses to test perms.
func TestDbusPerms() error {
	c := exec.Command(
		"sudo", "-u", "core",
		"gdbus", "call", "--system",
		"--dest", "org.freedesktop.systemd1",
		"--object-path", "/org/freedesktop/systemd1",
		"--method", "org.freedesktop.systemd1.Manager.RestartUnit",
		"ntpd.service", "replace",
	)
	out, err := c.CombinedOutput()

	if err != nil {
		if !strings.Contains(string(out), "org.freedesktop.DBus.Error.AccessDenied") &&
			!strings.Contains(string(out), "org.freedesktop.DBus.Error.InteractiveAuthorizationRequired") {
			return err
		}
	} else {
		return fmt.Errorf("We were able to call RestartUnit as a non-root user.")
	}

	c = exec.Command(
		"sudo", "-u", "core",
		"gdbus", "call", "--system",
		"--dest", "org.freedesktop.systemd1",
		"--object-path", "/org/freedesktop/systemd1/unit/ntpd_2eservice",
		"--method", "org.freedesktop.DBus.Properties.GetAll",
		"org.freedesktop.systemd1.Unit",
	)

	out, err = c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Err:%s\n Out:%v", err, out)
	}
	return nil
}

func TestSymlinkResolvConf() error {
	//t.Parallel()
	f, err := os.Lstat("/etc/resolv.conf")
	if err != nil {
		return fmt.Errorf("SymlinkResolvConf: %v", err)
	}
	if !IsLink(f) {
		return fmt.Errorf("/etc/resolv.conf is not a symlink.")
	}
	return nil
}

func TestSymlinkFlatcar() error {
	testSymlink := func(coreosPath, flatcarPath string) error {
		f, err := os.Lstat(coreosPath)
		if err != nil {
			return fmt.Errorf("unable to lstat on %s: %v", coreosPath, err)
		}
		if !IsLink(f) {
			return fmt.Errorf("coreos path %s is not a symlink.", coreosPath)
		}
		resolved, err := os.Readlink(coreosPath)
		if err != nil {
			return fmt.Errorf("unable to resolve symlink %s.", f)
		}
		// Works for us but in general the canonical path should be resolved.
		resolved = strings.TrimPrefix(resolved, "./")
		if resolved != filepath.Base(flatcarPath) {
			return fmt.Errorf("resolved path %s does not point to %s", resolved, filepath.Base(flatcarPath))
		}
		fr, err := os.Lstat(flatcarPath)
		if err != nil {
			return fmt.Errorf("unable to stat on %s: %v", flatcarPath, err)
		}
		if !fr.Mode().IsDir() && !fr.Mode().IsRegular() {
			return fmt.Errorf("path %s is not file or directory.", flatcarPath)
		}

		return nil
	}

	for cp, fp := range symlinkMap {
		if err := testSymlink(cp, fp); err != nil {
			return err
		}
	}

	return nil
}

func TestInstalledUpdateEngineRsaKeys() error {
	//t.Parallel()
	fileHash, err := Sha256File(UpdateEnginePubKey)
	if err != nil {
		return err
	}

	switch string(fileHash) {
	case UpdateEnginePubKeyV1, UpdateEnginePubKeyV2, UpdateEnginePubKeyFlatcarV1, UpdateEnginePubKeyFlatcarV2:
		return nil
	default:
		return fmt.Errorf("%s:%s unexpected hash.", UpdateEnginePubKey, fileHash)
	}
}

func TestServicesActive() error {
	return servicesActive([]string{
		"multi-user.target",
		"docker.socket",
		"systemd-timesyncd.service",
		"update-engine.service",
	})
}

func TestServicesActiveCoreOS() error {
	return servicesActive([]string{
		"multi-user.target",
	})
}

func servicesActive(units []string) error {
	//t.Parallel()
	for _, unit := range units {
		c := exec.Command("systemctl", "is-active", unit)
		err := c.Run()
		if err != nil {
			return fmt.Errorf("Services Active: %v", err)
		}
	}
	return nil
}

func TestServicesDisabledRHCOS() error {
	err := servicesInactive(offServices)
	if err != nil {
		return err
	}

	err = servicesDisabled(offServices)
	if err != nil {
		return err
	}
	return nil
}

func servicesInactive(units []string) error {
	for _, unit := range units {
		c := exec.Command("systemctl", "is-active", unit)
		err := c.Run()
		if err == nil {
			return fmt.Errorf("Service Incorrectly Active: %q", unit)
		}
	}
	return nil
}

func servicesDisabled(units []string) error {
	for _, unit := range units {
		c := exec.Command("systemctl", "is-enabled", unit)
		out, err := c.Output()
		if err == nil {
			// "is-enabled" can return 0 in some cases when the output is not
			// explicitly "disabled".  In the case of the RHCOS services
			// that are checked, we expect some to report "static"
			outString := strings.TrimSuffix(string(out), "\n")
			if (outString != "disabled") && (outString != "static") {
				return fmt.Errorf("Service Incorrectly Enabled: %q", unit)
			}
		}
	}
	return nil
}

func TestReadOnlyFs() error {
	mountModes := make(map[string]bool)
	mounts, err := GetMountTable()
	if err != nil {
		return err
	}
	for _, m := range mounts {
		mountModes[m.MountPoint] = m.Options[0] == "ro"
	}
	if mp, ok := mountModes["/usr"]; ok {
		if mp {
			return nil
		} else {
			return fmt.Errorf("/usr is not mounted read-only.")
		}
	} else if mp, ok := mountModes["/"]; ok {
		if mp {
			return nil
		} else {
			return fmt.Errorf("/ is not mounted read-only.")
		}
	}
	return fmt.Errorf("could not find /usr or / mount points.")
}

// Test that the root disk's GUID was set to a random one on first boot.
func TestFsRandomUUID() error {
	c := exec.Command("sh", "-ec", "sudo blkid -o value -s PTUUID /dev/$(lsblk -no PKNAME $(findmnt -vno SOURCE /))")
	out, err := c.Output()
	if err != nil {
		return fmt.Errorf("findmnt: %v", err)
	}

	got, err := uuid.ParseBytes(bytes.TrimSpace(out))
	if err != nil {
		return fmt.Errorf("malformed GUID: %v", err)
	}

	defaultGUID := uuid.Parse("00000000-0000-0000-0000-000000000001")
	if uuid.Equal(defaultGUID, got) {
		return fmt.Errorf("unexpected default GUID found")
	}

	return nil
}

// Test "Add User Manually", from https://coreos.com/os/docs/latest/adding-users.html
func TestUseradd() error {
	u := "user1"
	c := exec.Command("sudo", "useradd", "-p", "*", "-U", "-m", u, "-G", "sudo")
	err := c.Run()
	if err != nil {
		return fmt.Errorf("useradd: %v", err)
	}

	// verify
	c = exec.Command("id", u)
	err = c.Run()
	if err != nil {
		return fmt.Errorf("id %s: %v", u, err)
	}

	return nil
}

// Test that /etc/machine-id isn't empty or COREOS_BLANK_MACHINE_ID
func TestMachineID() error {
	id := MachineID()
	if id == "" {
		return fmt.Errorf("machine-id is empty")
	} else if id == "COREOS_BLANK_MACHINE_ID" {
		return fmt.Errorf("machine-id is %s", id)
	}
	return nil
}
