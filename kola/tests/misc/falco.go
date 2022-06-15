package misc

import (
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
)

func init() {
	register.Register(&register.Test{
		Run:         loadFalco,
		ClusterSize: 1,
		Name:        "cl.misc.falco",
		Distros:     []string{"cl"},
		Platforms:   []string{"qemu"},
		// selinux blocks insmod from within container
		Flags: []register.Flag{register.NoEnableSelinux},
	})
}

func loadFalco(c cluster.TestCluster) {
	// load the falco binary
	// TODO: first supported version will be 0.33.0, but use master tag for now
	c.MustSSH(c.Machines()[0], "docker run --rm --privileged -v /root/.falco:/root/.falco -v /proc:/host/proc:ro -v /boot:/host/boot:ro -v /lib/modules:/host/lib/modules:ro -v /usr:/host/usr:ro -v /etc:/host/etc:ro falcosecurity/falco-driver-loader:master")
	// Build must succeed and falco must be running
	c.MustSSH(c.Machines()[0], "dmesg | grep falco")
	c.MustSSH(c.Machines()[0], "lsmod | grep falco")
}
