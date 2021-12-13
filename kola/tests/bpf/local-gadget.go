// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package bpf

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/flatcar-linux/mantle/kola"
	"github.com/flatcar-linux/mantle/kola/cluster"
	"github.com/flatcar-linux/mantle/kola/register"
	"github.com/flatcar-linux/mantle/kola/tests/docker"
	"github.com/flatcar-linux/mantle/platform/conf"
)

type gadget struct {
	// Arch holds the gadget architecture (arm or amd).
	Arch string
	// Version holds the version of the gadget (v0.4.1 for example).
	Version string
	// Sum holds the SHA512 checksum to verify binary.
	Sum string
	// Cmd holds the command for the given gadget.
	Cmd string
}

var (
	// binaries holds the map of binaries for each supported
	// architecture.
	binaries = map[string]gadget{
		"arm64": gadget{
			Version: "0.4.1",
			Sum:     "4b5761dd08afea378e7e58a5a76b76c727ed59d327e042a42dd72330cac7bcd516da7574f69d22a4076c07dcafa639d08aaced56cf624816c5815226b33a0961",
		},
		"amd64": gadget{
			Version: "0.4.1",
			Sum:     "d68c2c55ac3d783f52aaf9b2e360955fa542f48901b6dc7fc4d52cd91e22f5c9839e6dd89e575923f117752e82f052d397c002cdd9acbcc3adfb893b8a18cdd8",
		},
	}
	config = `storage:
  files:
    - path: /opt/local-gadget.tar.gz
      filesystem: root
      mode: 0755
      contents:
        remote:
          url: https://github.com/kinvolk/inspektor-gadget/releases/download/v{{ .Version }}/local-gadget-linux-{{ .Arch }}.tar.gz
          verification:
            hash:
              function: sha512
              sum: {{ .Sum }}
    - path: /opt/local-gadget.cmd
      filesystem: root
      mode: 0644
      contents:
        inline: |
          {{ .Cmd }}
systemd:
  units:
    - name: prepare-local-gadget.service
      enabled: true
      contents: |
        [Unit]
        Description=Unpack local-gadget to /opt/bin/
        ConditionPathExists=!/opt/bin/local-gadget
        [Service]
        Type=oneshot
        RemainAfterExit=true
        Restart=on-failure
        ExecStartPre=/usr/bin/mkdir --parents /opt/bin
        ExecStartPre=/usr/bin/tar -v --extract --file /opt/local-gadget.tar.gz --directory /opt/bin --no-same-owner
        ExecStart=/usr/bin/rm /opt/local-gadget.tar.gz
        [Install]
        WantedBy=multi-user.target
    - name: local-gadget.service
      enabled: true
      contents: |
        [Unit]
        Description=Run local-gadget
        After=prepare-local-gadget.service
        Requires=prepare-local-gadget.service
        [Service]
        User=root
        Type=fork
        RemainAfterExit=true
        Restart=on-failure
        ExecStart=/opt/bin/local-gadget
        StandardInput=file:/opt/local-gadget.cmd
        StandardOutput=file:/tmp/local-gadget.res
        [Install]
        WantedBy=multi-user.target`
)

func init() {
	register.Register(&register.Test{
		Run:     localGadgetTest,
		Name:    `bpf.local-gadget`,
		Distros: []string{"cl"},
		// ESX is excluded because initramfs has no network access
		// so it's not able to download local-gadget binary.
		ExcludePlatforms: []string{"esx"},
		// required while SELinux policy is not correcly updated to support
		// `bpf` and `perfmon` permission.
		Flags: []register.Flag{register.NoEnableSelinux},
		// current LTS has DOCKER_API_VERSION=1.40 which is too old for local-gadget docker client.
		// "client version 1.41 is too new. Maximum supported API version is 1.40"
		ExcludeChannels: []string{"lts"},
	})
}

func localGadgetTest(c cluster.TestCluster) {
	arch := strings.SplitN(kola.QEMUOptions.Board, "-", 2)[0]
	gadget := binaries[arch]

	gadget.Arch = arch

	tmpl, err := template.New("user-data").Parse(config)
	if err != nil {
		c.Fatalf("parsing user-data: %w", err)
	}

	c.Run("dns gadget", func(c cluster.TestCluster) {
		gadget.Cmd = `create dns kola-dns --container-selector=shell01
          list-traces
          stream -f kola-dns`

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, gadget); err != nil {
			c.Fatalf("rendering user-data: %w", err)
		}

		node, err := c.NewMachine(conf.ContainerLinuxConfig(buf.String()))
		if err != nil {
			c.Fatalf("creating node: %v", err)
		}

		docker.GenDockerImage(c, node, "dig", []string{"dig"})

		if _, err := c.SSH(node, "docker run --rm --name shell01 dig dig flatcar-linux.org"); err != nil {
			c.Fatalf("unable to run docker cmd: %v", err)
		}

		out, err := c.SSH(node, "grep -m 1 pkt_type /tmp/local-gadget.res | jq -r .name")
		if err != nil {
			c.Fatalf("unable to get local-gadget res: %v", err)
		}

		name := string(out)

		if name != "flatcar-linux.org." {
			c.Fatalf("should have 'flatcar-linux.org.', got: %v", err)
		}
	})
}
