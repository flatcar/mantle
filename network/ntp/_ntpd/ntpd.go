// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"os"
	"time"

	"github.com/coreos/pkg/capnslog"

	"github.com/flatcar-linux/mantle/network/ntp"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "main")
	now  = flag.String("now", "", "Internal time for the server.")
	leap = flag.String("leap", "", "Handle a leap second.")
)

func main() {
	flag.Parse()
	capnslog.SetFormatter(capnslog.NewStringFormatter(os.Stderr))
	capnslog.SetGlobalLogLevel(capnslog.INFO)

	var l, n time.Time
	var err error
	if *now != "" {
		n, err = time.Parse(time.UnixDate, *now)
		if err != nil {
			plog.Fatalf("Parsing --now failed: %v", err)
		}
	}
	if *leap != "" {
		l, err = time.Parse(time.UnixDate, *leap)
		if err != nil {
			plog.Fatalf("Parsing --leap failed: %v", err)
		}
		if (l.Truncate(24*time.Hour) != l) || (l.UTC().Day() != 1) {
			plog.Fatalf("Invalid --leap time: %s", l)
		}
	}

	s, err := ntp.NewServer(":123")
	if err != nil {
		plog.Fatalf("Listen failed: %v", err)
	}

	if !n.IsZero() {
		s.SetTime(n)
	}
	if !l.IsZero() {
		s.SetLeapSecond(l, ntp.LEAP_ADD)
	}

	s.Serve()
}
