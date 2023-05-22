// Copyright 2017 CoreOS, Inc.
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
	"strings"

	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/register"
	"github.com/flatcar/mantle/platform/conf"
)

var multipartMimeUserdata = `Content-Type: multipart/mixed; boundary="MIMEMULTIPART"
MIME-Version: 1.0

--MIMEMULTIPART
Content-Type: text/cloud-config; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="cloud-config"

ssh_authorized_keys:
  - ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEftQIHTRvUmyDCN7VGve4srz03Jmq6rPnqq+XMHMQUIL9c/b0l7B5tWfQvQecKyLte94HOPzAyMJlktWTVGQnY=

--MIMEMULTIPART
Content-Type: text/cloud-config; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="cloud-config"

hostname: "example"

--MIMEMULTIPART
Content-Type: text/cloud-config; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="cloud-config"

write_files:
-   encoding: b64
    content: NDI=
    path: /tmp/kola_b64
    permissions: '0644'
-   encoding: base64
    content: NDI=
    path: /tmp/kola_b64_1
    permissions: '0644'
-   encoding: gzip
    content: !!binary |
        H4sIAGUfoFQC/zMxAgCIsCQyAgAAAA==
    path: /tmp/kola_gzip
    permissions: '0644'
-   encoding: gz
    content: !!binary |
        H4sIAGUfoFQC/zMxAgCIsCQyAgAAAA==
    path: /tmp/kola_gzip_1
    permissions: '0644'
-   encoding: gz+base64
    content: H4sIAGUfoFQC/zMxAgCIsCQyAgAAAA==
    path: /tmp/kola_gzip_base64
    permissions: '0644'
-   encoding: gzip+base64
    content: H4sIAGUfoFQC/zMxAgCIsCQyAgAAAA==
    path: /tmp/kola_gzip_base64_1
    permissions: '0644'
-   encoding: gz+b64
    content: H4sIAGUfoFQC/zMxAgCIsCQyAgAAAA==
    path: /tmp/kola_gzip_base64_2
    permissions: '0644'

--MIMEMULTIPART
Content-Type: text/x-shellscript; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="create_file.ps1"

#!/bin/sh
touch /coreos-cloudinit_multipart.txt

--MIMEMULTIPART
Content-Type: text/cloud-config; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="cloud-config"

#test_to_check_if_cloud_config_can_contain_a_comment

--MIMEMULTIPART
Content-Type: text/plain; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="some_text.txt"

This is just some random text.

--MIMEMULTIPART
Content-Type: application/json; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="ignition.txt"

{
  "ignitionVersion": 1,
  "This ignition configuration will be ignored because it's just embedded": "only cloud-init will run",
  "ignition": {
    "version": "3.0.0"
  },
  "systemd": {
    "units": [{
      "name": "example.service",
      "enabled": true,
      "contents": "[Service]\nType=oneshot\nExecStart=/usr/bin/echo Hello World\n\n[Install]\nWantedBy=multi-user.target"
    }]
  }
}

--MIMEMULTIPART
Content-Type: text/plain; charset="us-ascii"
MIME-Version: 1.0
Content-Transfer-Encoding: 7bit
Content-Disposition: attachment; filename="incognito_cloud_config.txt"

#cloud-config

write_files:
-   encoding: b64
    content: NDI=
    path: /kola_undercover
    permissions: '0644'

--MIMEMULTIPART--
`

func init() {
	register.Register(&register.Test{
		Run:         CloudInitBasic,
		ClusterSize: 1,
		Name:        "cl.cloudinit.basic",
		UserData: conf.CloudConfig(`#cloud-config
hostname: "core1"
write_files:
  - path: "/foo"
    content: bar`),
		Distros:          []string{"cl"},
		ExcludePlatforms: []string{"qemu-unpriv"},
		// This should run on all clouds
	})
	register.Register(&register.Test{
		Run:         CloudInitScript,
		ClusterSize: 1,
		Name:        "cl.cloudinit.script",
		UserData: conf.Script(`#!/bin/bash
echo bar > /foo
mkdir -p ~core/.ssh
cat <<EOF >> ~core/.ssh/authorized_keys
@SSH_KEYS@
EOF
chown -R core.core ~core/.ssh
chmod 700 ~core/.ssh
chmod 600 ~core/.ssh/authorized_keys`),
		Distros:          []string{"cl"},
		ExcludePlatforms: []string{"qemu-unpriv"},
		// When cl.cloudinit.basic passed we don't need to run this on all clouds
		Platforms: []string{"qemu", "qemu-unpriv"},
	})
	register.Register(&register.Test{
		Run:         CloudInitMultipartMime,
		ClusterSize: 1,
		Name:        "cl.cloudinit.multipart-mime",
		UserData:    conf.MultipartMimeConfig(multipartMimeUserdata),
		Distros:     []string{"cl"},
		Platforms:   []string{"qemu", "qemu-unpriv"},
	})
}

func CloudInitBasic(c cluster.TestCluster) {
	m := c.Machines()[0]

	out := c.MustSSH(m, "cat /foo")
	if string(out) != "bar" {
		c.Fatalf("cloud-config produced unexpected value %q", out)
	}

	out = c.MustSSH(m, "hostnamectl")
	if !strings.Contains(string(out), "Static hostname: core1") {
		c.Fatalf("hostname wasn't set correctly:\n%s", out)
	}
}

func CloudInitScript(c cluster.TestCluster) {
	m := c.Machines()[0]

	out := c.MustSSH(m, "cat /foo")
	if string(out) != "bar" {
		c.Fatalf("userdata script produced unexpected value %q", out)
	}
}

func CloudInitMultipartMime(c cluster.TestCluster) {
	m := c.Machines()[0]

	expectKey := "AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEftQIHTRvUmyDCN7VGve4srz03Jmq6rPnqq+XMHMQUIL9c/b0l7B5tWfQvQecKyLte94HOPzAyMJlktWTVGQnY="

	// Test that the hostname was set by the first multipart mime part that declares the "hostname"
	// cloud-config option. The second one at the end should be ignored.
	out := c.MustSSH(m, "hostnamectl")
	if !strings.Contains(string(out), "Static hostname: example") {
		c.Fatalf("hostname wasn't set correctly:\n%s", out)
	}

	// we can ignore the output. If the command fails, MustSSH will fail the test.
	c.MustSSH(m, fmt.Sprintf("grep %s ~core/.ssh/authorized_keys", expectKey))

	out = c.MustSSH(m, "ls -l /tmp/kola_*| wc -l")
	if string(strings.Replace(string(out), "\n", "", -1)) != "7" {
		c.Fatalf("expected 7 files in /tmp, found %q", out)
	}

	// All files should have the same content (42). These files should have been created by the cloud-config part
	// that declares the write_files option.
	c.MustSSH(m, `for f in $(ls /tmp/kola_*); do OUT=$(cat $f); if [ "$OUT" != 42 ]; then exit 1; fi; done`)
	// Check that the x-shellscript part was executed.
	c.MustSSH(m, "test -f /coreos-cloudinit_multipart.txt")
	c.MustSSH(m, "test -f /kola_undercover")
}
