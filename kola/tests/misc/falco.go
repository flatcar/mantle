package misc

import (
	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
)

func init() {
	register.Register(&register.Test{
		Run:         loadFalco,
		ClusterSize: 1,
		Name:        "cl.misc.falco",
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu"},
		// falco builder container can't handle our arm64 config (yet)
		Architectures: []string{"amd64"},
		// selinux blocks insmod from within container
		Flags:    []register.Flag{register.NoEnableSelinux},
		SkipFunc: kola.SkipSecureboot,
	})
}

func loadFalco(c cluster.TestCluster) {
	// load the falco binary
	c.MustSSH(c.Machines()[0], "docker run --rm --privileged -v /root/.falco:/root/.falco -v /proc:/host/proc:ro -v /boot:/host/boot:ro -v /lib/modules:/host/lib/modules:ro -v /usr:/host/usr:ro -v /etc:/host/etc:ro falcosecurity/falco-driver-loader:master kmod")
	// Build must succeed and falco must be running
	c.MustSSH(c.Machines()[0], "dmesg | grep falco")
	c.MustSSH(c.Machines()[0], "lsmod | grep falco")
}
