package stackit

import (
	"context"
	"fmt"

	"github.com/coreos/pkg/capnslog"
	ctplatform "github.com/flatcar/container-linux-config-transpiler/config/platform"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/platform/api/stackit"
)

const (
	Platform platform.Name = "stackit"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/machine/stackit")
)

type flight struct {
	*platform.BaseFlight
	api *stackit.API

	keypair *stackit.Keypair
}

func NewFlight(opts *stackit.Options) (platform.Flight, error) {
	api, err := stackit.New(opts)
	if err != nil {
		return nil, fmt.Errorf("creating STACKIT API client: %w", err)
	}

	// TODO: Rework the Base Flight to remove the CT dependency.
	base, err := platform.NewBaseFlight(opts.Options, Platform, ctplatform.OpenStackMetadata)
	if err != nil {
		return nil, fmt.Errorf("creating base flight: %w", err)
	}

	bf := &flight{
		BaseFlight: base,
		api:        api,
	}

	keys, err := bf.Keys()
	if err != nil {
		bf.Destroy()
		return nil, err
	}
	if len(keys) > 0 {
		bf.keypair, err = api.CreateKeyPair(context.TODO(), bf.Name(), keys[0].String())
		if err != nil {
			bf.Destroy()
			return nil, err
		}
	}

	return bf, nil
}

func (bf *flight) NewCluster(rconf *platform.RuntimeConfig) (platform.Cluster, error) {
	bc, err := platform.NewBaseCluster(bf.BaseFlight, rconf)
	if err != nil {
		return nil, fmt.Errorf("creating stackit base cluster: %w", err)
	}

	c := &cluster{
		BaseCluster: bc,
		flight:      bf,
	}

	if !rconf.NoSSHKeyInMetadata {
		c.keypair = bf.keypair
	}

	c.network, err = bf.api.CreateNetwork(context.Background(), bc.Name())
	if err != nil {
		return nil, fmt.Errorf("creating network for cluster: %w", err)
	}

	bf.AddCluster(c)

	return c, nil
}

func (bf *flight) Destroy() {
	if bf.keypair != nil {
		_ = bf.api.DeleteKeyPair(context.TODO(), *bf.keypair.Name)
	}

	bf.BaseFlight.Destroy()
}
