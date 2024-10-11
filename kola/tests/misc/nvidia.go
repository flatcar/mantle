package misc

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/kola"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/util"
)

const (
	CmdTimeout = time.Second * 300
)

var plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "kola/tests/misc")

func init() {
	register.Register(&register.Test{
		Name:        "cl.misc.nvidia",
		Run:         verifyNvidiaInstallation,
		ClusterSize: 1,
		Distros:     []string{"cl"},
		// This test is to test the NVIDIA installation, limited to AZURE for now
		Platforms:     []string{"azure"},
		Architectures: []string{"amd64"},
		Flags:         []register.Flag{register.NoEnableSelinux},
		SkipFunc:      skipOnNonGpu,
	})
}

func skipOnNonGpu(version semver.Version, channel, arch, platform string) bool {
	// N stands for GPU instance obviously :)
	if platform == "azure" && strings.Contains(kola.AzureOptions.Size, "N") {
		return false
	}
	return true
}

func verifyNvidiaInstallation(c cluster.TestCluster) {
	m := c.Machines()[0]

	nvidiaStatusRetry := func() error {
		out, err := c.SSH(m, "systemctl status nvidia.service")
		if !bytes.Contains(out, []byte("active (exited)")) {
			return fmt.Errorf("nvidia.service: %q: %v", out, err)
		}
		return nil
	}

	if err := util.Retry(40, 15*time.Second, nvidiaStatusRetry); err != nil {
		c.Fatal(err)
	}
	c.AssertCmdOutputContains(m, "/opt/bin/nvidia-smi", "Tesla")
}
