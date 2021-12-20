// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"strings"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/platform"
)

// GenPodmanScratchContainer creates a podman scratch container out of binaries from the host
func GenPodmanScratchContainer(c cluster.TestCluster, m platform.Machine, name string, binnames []string) {
	cmd := `tmpdir=$(mktemp -d); cd $tmpdir; echo -e "FROM scratch\nCOPY . /" > Dockerfile;
	        b=$(which %s); libs=$(sudo ldd $b | grep -o /lib'[^ ]*' | sort -u);
			sudo rsync -av --relative --copy-links $b $libs ./;
			sudo podman build --layers=false -t localhost/%s .`
	c.MustSSH(m, fmt.Sprintf(cmd, strings.Join(binnames, " "), name))
}
