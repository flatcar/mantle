// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package system

import (
	"net"
	"os"
	"strings"
)

// FullHostname is a best effort attempt to resolve the canonical FQDN of
// the host. On failure it will fall back to a reasonable looking default
// such as 'localhost.' or 'hostname.invalid.'
func FullHostname() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "localhost" || hostname == "(none)" {
		return "localhost."
	}
	fullname, err := net.LookupCNAME(hostname)
	if err != nil {
		fullname = hostname
		if !strings.Contains(fullname, ".") {
			fullname += ".invalid."
		}
	}
	return fullname
}
