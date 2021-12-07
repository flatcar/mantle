// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package misc

import (
	"fmt"
	"time"

	"github.com/coreos/go-omaha/omaha"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/platform/conf"
	"github.com/flatcar-linux/mantle/platform/machine/qemu"
)

func init() {
	register.Register(&register.Test{
		Run:              OmahaPing,
		ClusterSize:      0,
		Name:             "cl.omaha.ping",
		Platforms:        []string{"qemu"},
		ExcludePlatforms: []string{"qemu-unpriv"},
		Distros:          []string{"cl"},
	})
}

type pingServer struct {
	omaha.UpdaterStub

	ping chan struct{}
}

func (ps *pingServer) Ping(req *omaha.Request, app *omaha.AppRequest) {
	ps.ping <- struct{}{}
}

func OmahaPing(c cluster.TestCluster) {
	qc, ok := c.Cluster.(*qemu.Cluster)
	if !ok {
		c.Fatal("test only works in qemu")
	}

	omahaserver := qc.LocalCluster.OmahaServer

	svc := &pingServer{
		ping: make(chan struct{}),
	}

	omahaserver.Updater = svc

	hostport, err := qc.GetOmahaHostPort()
	if err != nil {
		c.Fatalf("couldn't get Omaha server address: %v", err)
	}
	config := fmt.Sprintf(`update:
  server: "http://%s/v1/update/"
`, hostport)

	m, err := c.NewMachine(conf.ContainerLinuxConfig(config))
	if err != nil {
		c.Fatalf("couldn't start machine: %v", err)
	}

	out, stderr, err := m.SSH("update_engine_client -check_for_update")
	if err != nil {
		c.Fatalf("couldn't check for update: %s, %s, %v", out, stderr, err)
	}

	tc := time.After(30 * time.Second)

	select {
	case <-tc:
		c.Fatal("timed out waiting for omaha ping")
	case <-svc.ping:
	}
}
