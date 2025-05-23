// Copyright 2018 Red Hat
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package misc

import (
	"fmt"

	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
)

var (
	urlsToFetch = []string{
		"https://www.example.com/",
		"https://www.wikipedia.org/",
		"https://start.fedoraproject.org/",
	}
)

func init() {
	register.Register(&register.Test{
		Run:            TestTLSFetchURLs,
		ClusterSize:    1,
		Name:           "coreos.tls.fetch-urls",
		ExcludeDistros: []string{"rhcos", "fcos"}, // wget not included in *COS
		// This test is normally not related to the cloud environment (and we have other tests for networking)
		Platforms: []string{"qemu", "qemu-unpriv"},
	})
}

func TestTLSFetchURLs(c cluster.TestCluster) {
	m := c.Machines()[0]

	for _, url := range urlsToFetch {
		c.MustSSH(m, fmt.Sprintf("curl -s -S -m 30 --retry 2 %s", url))
		c.MustSSH(m, fmt.Sprintf("wget -nv -T 30 -t 2 --delete-after %s 2> >(grep -v -- '->' >&2)", url))
	}
}
