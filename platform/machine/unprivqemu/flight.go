// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

package unprivqemu

import (
	"os"

	"github.com/coreos/pkg/capnslog"

	"github.com/flatcar-linux/mantle/platform"
	"github.com/flatcar-linux/mantle/platform/machine/qemu"
)

const (
	Platform platform.Name = "qemu"
)

type flight struct {
	*platform.BaseFlight
	opts *qemu.Options

	diskImagePath string
	diskImageFile *os.File
}

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar-linux/mantle", "platform/machine/qemu")
)

func NewFlight(opts *qemu.Options) (platform.Flight, error) {
	bf, err := platform.NewBaseFlight(opts.Options, Platform, "")
	if err != nil {
		return nil, err
	}

	qf := &flight{
		BaseFlight:    bf,
		opts:          opts,
		diskImagePath: opts.DiskImage,
	}

	return qf, nil
}

// NewCluster creates a Cluster instance, suitable for running virtual
// machines in QEMU.
func (qf *flight) NewCluster(rconf *platform.RuntimeConfig) (platform.Cluster, error) {
	bc, err := platform.NewBaseCluster(qf.BaseFlight, rconf)
	if err != nil {
		return nil, err
	}

	qc := &Cluster{
		BaseCluster: bc,
		flight:      qf,
	}

	qf.AddCluster(qc)

	return qc, nil
}

func (qf *flight) Destroy() {
	if qf.diskImageFile != nil {
		qf.diskImageFile.Close()
	}
}
