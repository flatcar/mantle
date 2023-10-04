// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package misc

import (
	"github.com/coreos/go-semver/semver"
	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform/conf"
)

func init() {
	register.Register(&register.Test{
		Run:         fipsTest,
		ClusterSize: 1,
		Name:        `misc.fips`,
		MinVersion:  semver.Version{Major: 3549},
		Distros:     []string{"cl"},
		// This test is normally not related to the cloud environment
		Platforms: []string{"qemu", "qemu-unpriv"},
		UserData: conf.Butane(`---
version: 1.0.0
variant: flatcar
kernel_arguments:
  should_exist:
    - fips=1
storage:
  files:
    - path: /etc/system-fips
    - path: /etc/ssl/openssl.cnf
      overwrite: true
      mode: 0644
      contents:
        inline: |
          config_diagnostics = 1
          openssl_conf = openssl_init
          # includes the fipsmodule configuration
          .include /etc/ssl/fipsmodule.cnf
          [openssl_init]
          providers = provider_sect
          [provider_sect]
          fips = fips_sect
          base = base_sect
          [base_sect]
          activate = 1`),
	})

}

func fipsTest(c cluster.TestCluster) {
	m := c.Machines()[0]

	// It works because SHA is FIPS compliant.
	c.MustSSH(m, "echo Flatcar | openssl sha512 -")

	// Should exit with 0.
	c.MustSSH(m, "openssl list -provider fips")

	// It does not work because MD5 is not FIPS compliant.
	if _, err := c.SSH(m, "echo Flatcar | openssl md5 -"); err == nil {
		c.Fatal("MD5 hash algorithm should raise an error with FIPS mode.")
	}

	c.AssertCmdOutputContains(m, "cat /proc/sys/crypto/fips_enabled", "1")
}
