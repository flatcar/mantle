// Copyright 2016 CoreOS, Inc.
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

package locksmith

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"

	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/kola/tests/etcd"
	"github.com/flatcar-linux/mantle/lang/worker"
	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/conf"
	"github.com/flatcar-linux/mantle/util"
)

func init() {
	register.Register(&register.Test{
		Name:        "cl.locksmith.airlock",
		Run:         airlock,
		ClusterSize: 1,
		UserData: conf.Butane(`---
variant: flatcar
version: 1.0.0
systemd:
  units:
    - name: update-engine.service
      enabled: true
    - name: etcd-member.service
      enabled: true
    - name: locksmithd.service
      enabled: true
      dropins:
        - name: custom.conf
          contents: |
            [Service]
            Environment=LOCKSMITHD_ENDPOINT=http://127.0.0.1:3333
            Environment=LOCKSMITHD_GROUP=default
            Environment=LOCKSMITHD_ID=12345
    - name: airlock.service
      enabled: true
      contents: |
        [Install]
        WantedBy=multi-user.target
        [Unit]
        After=etcd-member.service
        Requires=etcd-member.service
        [Service]
        Type=fork
        ExecStartPre=-/usr/bin/docker stop airlock
        ExecStartPre=-/usr/bin/docker pull quay.io/coreos/airlock:main
        ExecStart=/usr/bin/docker \
          run \
          --rm \
          --network host \
          --name airlock \
          -v "/opt/config.toml:/etc/airlock/config.toml:ro,z" \
          quay.io/coreos/airlock:main \
          airlock serve -vv
storage:
  files:
    - path: /opt/config.toml
      contents:
        inline: |
          # Main service configuration
          [service]
          address = "127.0.0.1"
          port = 3333
          tls = false
          # Etcd-v3 client configuration
          [etcd3]
          endpoints = [ "http://127.0.0.1:2379" ]
          # Lock configuration, base reboot group
          [lock]
          default_group_name = "default"
          default_slots = 2
          # Lock configuration, additional reboot groups
          [[lock.groups]]
          name = "workers"
          [[lock.groups]]
          name = "controllers"
          slots = 1
      mode: 0644`),
		Distros:   []string{"cl"},
		Platforms: []string{"qemu"},
	})
	register.Register(&register.Test{
		Name:        "cl.locksmith.cluster",
		Run:         locksmithCluster,
		ClusterSize: 3,
		UserData: conf.ContainerLinuxConfig(`locksmith:
  reboot_strategy: etcd-lock
etcd:
  version:                     3.5.0
  listen_client_urls:          http://0.0.0.0:2379
  advertise_client_urls:       http://{PRIVATE_IPV4}:2379
  initial_advertise_peer_urls: http://{PRIVATE_IPV4}:2380
  listen_peer_urls:            http://{PRIVATE_IPV4}:2380
  discovery:                   $discovery
  enable_v2:                   true`),
		Distros: []string{"cl"},
	})
	register.Register(&register.Test{
		Name:        "coreos.locksmith.reboot",
		Run:         locksmithReboot,
		ClusterSize: 1,
		Distros:     []string{"cl"},
	})
	register.Register(&register.Test{
		Name:        "coreos.locksmith.tls",
		Run:         locksmithTLS,
		ClusterSize: 1,
		UserData: conf.Ignition(`{
  "ignition": { "version": "2.0.0" },
  "systemd": {
    "units": [
      {
        "name": "certgen.service",
        "contents": "[Unit]\nAfter=system-config.target\nAfter=time-sync.target\nWants=time-sync.target\n\n[Service]\nType=oneshot\nRemainAfterExit=yes\nExecStartPre=/bin/sh -c 'e=600; for i in $(seq $e); do echo Waiting for time sync $i/$e; timedatectl | grep -q \"System clock synchronized: yes\" && break; sleep 1; done'\nExecStartPre=/usr/bin/mkdir -p /etc/ssl/certs\nExecStart=/usr/bin/openssl req -config /etc/ssl/etcd.cnf -x509 -nodes -newkey rsa:4096 -sha512 -days 3 -extensions etcd_ca -subj '/CN=etcd CA' -out /etc/ssl/certs/ca-etcd-cert.pem -keyout /etc/ssl/certs/ca-etcd-key.pem\nExecStart=/usr/bin/openssl req -config /etc/ssl/etcd.cnf -nodes -newkey rsa:4096 -sha512 -days 3 -extensions etcd_server -subj '/CN=localhost' -out /etc/ssl/certs/etcd-csr.pem -keyout /etc/ssl/certs/etcd-key.pem\nExecStart=/usr/bin/openssl x509 -extfile /etc/ssl/etcd.cnf -extensions etcd_server -CA /etc/ssl/certs/ca-etcd-cert.pem -CAkey /etc/ssl/certs/ca-etcd-key.pem -CAcreateserial -sha512 -days 3 -req -in /etc/ssl/certs/etcd-csr.pem -out /etc/ssl/certs/etcd-cert.pem\nExecStart=/usr/bin/openssl req -config /etc/ssl/etcd.cnf -x509 -nodes -newkey rsa:4096 -sha512 -days 3 -extensions etcd_ca -subj '/CN=locksmith CA' -out /etc/ssl/certs/ca-locksmith-cert.pem -keyout /etc/ssl/certs/ca-locksmith-key.pem\nExecStart=/usr/bin/openssl req -config /etc/ssl/etcd.cnf -nodes -newkey rsa:4096 -sha512 -days 3 -extensions etcd_client -subj '/CN=locksmith client' -out /etc/ssl/certs/locksmith-csr.pem -keyout /etc/ssl/certs/locksmith-key.pem\nExecStart=/usr/bin/openssl x509 -extfile /etc/ssl/etcd.cnf -extensions etcd_client -CA /etc/ssl/certs/ca-locksmith-cert.pem -CAkey /etc/ssl/certs/ca-locksmith-key.pem -CAcreateserial -sha512 -days 3 -req -in /etc/ssl/certs/locksmith-csr.pem -out /etc/ssl/certs/locksmith-cert.pem\nExecStart=/usr/bin/chmod 0644 /etc/ssl/certs/ca-etcd-cert.pem /etc/ssl/certs/ca-etcd-key.pem /etc/ssl/certs/ca-locksmith-cert.pem /etc/ssl/certs/ca-locksmith-key.pem /etc/ssl/certs/etcd-cert.pem /etc/ssl/certs/etcd-key.pem /etc/ssl/certs/locksmith-cert.pem /etc/ssl/certs/locksmith-key.pem\nExecStart=/usr/bin/ln -fns ca-etcd-cert.pem /etc/ssl/certs/etcd.pem\nExecStart=/usr/bin/c_rehash"
      },
      {
        "name": "etcd-member.service",
        "dropins": [{
          "name": "environment.conf",
          "contents": "[Unit]\nAfter=certgen.service\nRequires=certgen.service\n[Service]\nEnvironment=ETCD_ADVERTISE_CLIENT_URLS=https://127.0.0.1:2379\nEnvironment=ETCD_LISTEN_CLIENT_URLS=https://127.0.0.1:2379\nEnvironment=ETCD_CERT_FILE=/etc/ssl/certs/etcd-cert.pem\nEnvironment=ETCD_KEY_FILE=/etc/ssl/certs/etcd-key.pem\nEnvironment=ETCD_TRUSTED_CA_FILE=/etc/ssl/certs/ca-locksmith-cert.pem\nEnvironment=ETCD_CLIENT_CERT_AUTH=true\nEnvironment=ETCD_ENABLE_V2=true"
        }]
      },
      {
        "name": "locksmithd.service",
        "enable": true,
        "dropins": [{
          "name": "environment.conf",
          "contents": "[Unit]\nAfter=etcd-member.service\nRequires=etcd-member.service\n[Service]\nEnvironment=LOCKSMITHD_ETCD_CERTFILE=/etc/ssl/certs/locksmith-cert.pem\nEnvironment=LOCKSMITHD_ETCD_KEYFILE=/etc/ssl/certs/locksmith-key.pem\nEnvironment=LOCKSMITHD_ETCD_CAFILE=/etc/ssl/certs/ca-etcd-cert.pem\nEnvironment=LOCKSMITHD_ENDPOINT=https://localhost:2379\nEnvironment=LOCKSMITHD_REBOOT_WINDOW_START=00:00\nEnvironment=LOCKSMITHD_REBOOT_WINDOW_LENGTH=23h59m"
        }]
      }
    ]
  },
  "storage": {
    "files": [
      {
        "filesystem": "root",
        "path": "/etc/coreos/update.conf",
        "contents": { "source": "data:,REBOOT_STRATEGY=etcd-lock%0A" },
        "mode": 420
      },
      {
        "filesystem": "root",
        "path": "/etc/ssl/etcd.cnf",
        "contents": { "source": "data:,%5Breq%5D%0Adistinguished_name=req%0A%5Betcd_ca%5D%0AbasicConstraints=CA:true%0AkeyUsage=keyCertSign,cRLSign%0AsubjectKeyIdentifier=hash%0A%5Betcd_client%5D%0AbasicConstraints=CA:FALSE%0AextendedKeyUsage=clientAuth%0AkeyUsage=digitalSignature,keyEncipherment%0A%5Betcd_server%5D%0AbasicConstraints=CA:FALSE%0AextendedKeyUsage=serverAuth%0AkeyUsage=digitalSignature,keyEncipherment%0AsubjectAltName=@sans%0A%5Bsans%5D%0ADNS.1=localhost%0AIP.1=127.0.0.1%0A" },
        "mode": 420
      }
    ]
  }
}`),
		Distros: []string{"cl"},
	})
}

func locksmithReboot(c cluster.TestCluster) {
	// The machine should be able to reboot without etcd in the default mode
	m := c.Machines()[0]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	output, err := c.SSH(m, "sudo systemctl stop sshd.socket && locksmithctl send-need-reboot")
	if _, ok := err.(*ssh.ExitMissingError); ok {
		err = nil // A terminated session is perfectly normal during reboot.
	} else if err == io.EOF {
		err = nil // Sometimes copying command output returns EOF here.
	}
	if err != nil {
		c.Fatalf("failed to run \"locksmithctl send-need-reboot\": output: %q status: %q", output, err)
	}

	err = platform.CheckMachine(ctx, m)
	if err != nil {
		c.Fatalf("failed to check rebooted machine: %v", err)
	}

}

func locksmithCluster(c cluster.TestCluster) {
	machs := c.Machines()

	// Wait for all etcd cluster nodes to be ready.
	if err := etcd.GetClusterHealth(c, machs[0], len(machs)); err != nil {
		c.Fatalf("cluster health: %v", err)
	}

	c.MustSSH(machs[0], "locksmithctl status")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	wg := worker.NewWorkerGroup(ctx, len(machs))

	// reboot all the things
	for _, m := range machs {
		worker := func(ctx context.Context) error {
			cmd := "sudo systemctl stop sshd.socket && sudo locksmithctl send-need-reboot"
			output, err := c.SSH(m, cmd)
			if _, ok := err.(*ssh.ExitMissingError); ok {
				err = nil // A terminated session is perfectly normal during reboot.
			} else if err == io.EOF {
				err = nil // Sometimes copying command output returns EOF here.
			}
			if err != nil {
				return fmt.Errorf("failed to run %q: output: %q status: %q", cmd, output, err)
			}

			return platform.CheckMachine(ctx, m)
		}

		if err := wg.Start(worker); err != nil {
			c.Fatal(wg.WaitError(err))
		}
	}

	if err := wg.Wait(); err != nil {
		c.Fatal(err)
	}
}

func locksmithTLS(c cluster.TestCluster) {
	m := c.Machines()[0]
	lCmd := "sudo locksmithctl --endpoint https://localhost:2379 --etcd-cafile /etc/ssl/certs/ca-etcd-cert.pem --etcd-certfile /etc/ssl/certs/locksmith-cert.pem --etcd-keyfile /etc/ssl/certs/locksmith-key.pem "

	// First verify etcd has a valid TLS connection ready
	// Retry a few times in case the system clock is adjusted by a few seconds
	// causing the certificate to be rejected during the first tries
	retryClock := func() error {
		output, err := c.SSH(m, "openssl s_client -showcerts -verify_return_error -verify_ip 127.0.0.1 -verify_hostname localhost -connect localhost:2379 -cert /etc/ssl/certs/locksmith-cert.pem -key /etc/ssl/certs/locksmith-key.pem 0</dev/null 2>&1")
		if err != nil || !bytes.Contains(output, []byte("Verify return code: 0")) {
			return fmt.Errorf("openssl s_client: %q: %v", output, err)
		}
		return nil
	}
	if err := util.Retry(5, 12*time.Second, retryClock); err != nil {
		c.Fatal(err)
	}

	// Also verify locksmithctl understands the TLS connection
	c.MustSSH(m, lCmd+"status")

	// Stop locksmithd
	c.MustSSH(m, "sudo systemctl stop locksmithd.service")

	// Set the lock while locksmithd isn't looking
	c.MustSSH(m, lCmd+"lock")

	// Verify it is locked
	output, err := c.SSH(m, lCmd+"status")
	if err != nil || !bytes.HasPrefix(output, []byte("Available: 0\nMax: 1")) {
		c.Fatalf("locksmithctl status (locked): %q: %v", output, err)
	}

	// Start locksmithd
	c.MustSSH(m, "sudo systemctl start locksmithd.service")

	// Verify it is unlocked (after locksmithd wakes up again)
	checker := func() error {
		output, err := c.SSH(m, lCmd+"status")
		if err != nil || !bytes.HasPrefix(output, []byte("Available: 1\nMax: 1")) {
			return fmt.Errorf("locksmithctl status (unlocked): %q: %v", output, err)
		}
		return nil
	}
	if err := util.Retry(10, 12*time.Second, checker); err != nil {
		c.Fatal(err)
	}
}

func airlock(c cluster.TestCluster) {
	m := c.Machines()[0]

	c.MustSSH(m, "locksmithctl exp --id kola-test-id --group default --endpoint http://localhost:3333 lock")
	c.AssertCmdOutputContains(m, "docker exec airlock airlock exp get slots", "kola-test-id")
}
