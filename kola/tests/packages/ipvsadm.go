// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package packages

import (
	"bytes"
	"fmt"

	"github.com/flatcar-linux/mantle/kola/cluster"
)

func ipvsadm(c cluster.TestCluster) {
	m := c.Machines()[0]

	// Test it runs at all
	out := c.MustSSH(m, "sudo ipvsadm")
	if !bytes.Contains(out, []byte(`IP Virtual Server version`)) {
		c.Fatalf("unexpected ipvsadm output: %v", string(out))
	}

	// Test by using the example from the man page
	cmd := `echo " 
	-A -t 207.175.44.110:80 -s rr
	-a -t 207.175.44.110:80 -r 192.168.10.1:80 -m
	-a -t 207.175.44.110:80 -r 192.168.10.2:80 -m
	-a -t 207.175.44.110:80 -r 192.168.10.3:80 -m
	-a -t 207.175.44.110:80 -r 192.168.10.4:80 -m
	-a -t 207.175.44.110:80 -r 192.168.10.5:80 -m
	" | sudo ipvsadm -R`
	c.MustSSH(m, cmd)

	// Test we can read back what we just did
	out = c.MustSSH(m, "sudo ipvsadm -Ln")
	if !bytes.Contains(out, []byte(`TCP  207.175.44.110:80 rr`)) {
		c.Fatalf("could not create virtual service %v", string(out))
	}
	for i := 1; i <= 5; i++ {
		ip := []byte(fmt.Sprintf("-> 192.168.10.%d:80", i))
		if !bytes.Contains(out, ip) {
			c.Fatalf("did not add real service %v", string(ip))
		}
	}

	// Test we can delete the service
	c.MustSSH(m, "sudo ipvsadm -D -t 207.175.44.110:80")

	// Ensure it was really deleted
	out = c.MustSSH(m, "sudo ipvsadm -Ln")
	if bytes.Contains(out, []byte(`TCP 207.175.44.110:80 rr`)) {
		c.Fatalf("could not delete virtual service")
	}
}
