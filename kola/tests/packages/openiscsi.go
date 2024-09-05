package packages

import (
	"fmt"
	"strings"

	"github.com/flatcar/mantle/kola/cluster"
	"github.com/flatcar/mantle/kola/tests/util"
	"github.com/flatcar/mantle/platform"
)

var (
	getInitiatorClientScript = util.TrimLeftSpace(`
#!/bin/bash

set -euo pipefail

sudo systemctl start iscsid
for i in {0..9}; do
    if [[ ! -e /etc/iscsi/initiatorname.iscsi ]]; then
        sleep 1
        continue
    fi
    name=$(grep -F InitiatorName /etc/iscsi/initiatorname.iscsi | cut -d= -f2-)
    if [[ -z ${name} ]]; then
        echo "malformed initiator name config"
        exit 1
    fi
    echo "${name}"
    exit 0
done
echo "no initiator name config found"
exit 1
`)

	setupServerScript = util.TrimLeftSpace(`
#!/bin/bash

set -euo pipefail

initiator=${1}

mkdir -p /shared

cat <<EOF >/shared/init.script
cd /
backstores/fileio create test /shared/test.img 100m
iscsi/ create iqn.2006-04.com.example:test-target
cd iscsi/iqn.2006-04.com.example:test-target/tpg1/
luns/ create /backstores/fileio/test
set attribute generate_node_acls=1
acls/ create ${initiator}
EOF

docker_args=(
    --privileged
    --network host
    --mount type=bind,source=/sys,destination=/sys
    --mount type=bind,source=/shared,destination=/shared
    --mount type=bind,source=/run/dbus,destination=/run/dbus,readonly
    --mount type=bind,source=/usr/lib/modules,destination=/usr/lib/modules,readonly
    --rm
)

docker run "${docker_args[@]}" ghcr.io/flatcar/targetcli-fb bash -c 'targetcli </shared/init.script'
`)

	discoverClientScript = util.TrimLeftSpace(`
#!/bin/bash

set -euo pipefail

host_ip=${1}

target=$(iscsiadm --mode=discovery --type=sendtargets --portal="${host_ip}" | cut -d' ' -f 2-)
iscsiadm --mode=node --login --targetname="${target}" --portal="${host_ip}"

for i in {0..9}; do
    if [[ -e /dev/sda ]]; then
        break
    fi
    sleep 1
done

if [[ ! -e /dev/sda ]]; then
    echo "no /dev/sda device"
    exit 1
fi

mkfs.ext2 /dev/sda

mkdir -p /drive
mount -t ext2 /dev/sda /drive
echo "seems to be working" >/drive/test-file
umount /drive

systemctl enable iscsi
`)

	checkClientScript = util.TrimLeftSpace(`
#!/bin/bash

set -euo pipefail

if [[ ! -e /dev/sda ]]; then
    echo "no /dev/sda device after reboot"
    exit 1
fi

mkdir -p /drive
mount -t ext2 /dev/sda /drive

if [[ ! -e /drive/test-file ]]; then
    echo 'expected file missing'
    exit 1
fi

contents=$(cat /drive/test-file)
if [[ ${contents} != 'seems to be working' ]]; then
    echo "unexpected file contents: ${contents@Q}"
    exit 1
fi
umount /drive
`)
)

func openISCSI(c cluster.TestCluster) {
	// machine 0 will have the remote disk mounted
	//
	// machine 1 will be a disk provider
	client := c.Machines()[0]
	server := c.Machines()[1]

	for name, script := range map[string]string{
		"/get_initiator": getInitiatorClientScript,
		"/discover":      discoverClientScript,
		"/check":         checkClientScript,
	} {
		if err := platform.InstallFile(strings.NewReader(script), client, name); err != nil {
			c.Fatalf("failed to upload script %s to client: %v", name, err)
		}
	}
	if err := platform.InstallFile(strings.NewReader(setupServerScript), server, "/setup"); err != nil {
		c.Fatalf("failed to upload script /setup to server: %v", err)
	}

	c.MustSSH(client, "sudo chmod a+x /get_initiator /discover /check")
	c.MustSSH(server, "sudo chmod a+x /setup")

	initiatorName := c.MustSSH(client, `sudo /get_initiator`)
	c.MustSSH(server, fmt.Sprintf("sudo /setup '%s'", initiatorName))
	c.MustSSH(client, fmt.Sprintf("sudo /discover %s", server.PrivateIP()))
	if err := client.Reboot(); err != nil {
		c.Fatalf("failed to reboot the client: %v", err)
	}
	c.MustSSH(client, "sudo /check")
}
