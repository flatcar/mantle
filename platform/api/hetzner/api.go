// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package hetzner

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/version"

	"github.com/apricote/hcloud-upload-image/hcloudimages"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var (
	DefaultLabels = map[string]string{
		"managed-by": "mantle",
	}
)

// Options hold the specific Hetzner options.
type Options struct {
	*platform.Options
	// Image is the ID of the Hetzner image to deploy.
	Image string
	// ServerType is the type of the instance (e.g. cx22).
	ServerType string
	// Location is used as "default" zone for all requests.
	Location string
	// Token is used to construct the Hetzner client
	Token string
}

// API is a wrapper around Hetzner instance API.
type API struct {
	client       *hcloud.Client
	imagesclient *hcloudimages.Client

	serverType *hcloud.ServerType
	image      *hcloud.Image
	location   *hcloud.Location
}

type Server struct{ *hcloud.Server }

type SSHKey struct{ *hcloud.SSHKey }

type Network struct{ *hcloud.Network }

// New returns a Hetzner API instance.
func New(opts *Options) (*API, error) {
	client := hcloud.NewClient(
		hcloud.WithToken(opts.Token),
		hcloud.WithApplication("flatcar-mantle", version.Version),
		hcloud.WithPollBackoffFunc(hcloud.ExponentialBackoff(2, 500*time.Millisecond)),
	)

	imagesclient := hcloudimages.NewClient(client)

	ctx := context.Background()

	serverType, _, err := client.ServerType.Get(ctx, opts.ServerType)
	if err != nil {
		return nil, fmt.Errorf("verifying server type: %w", err)
	}

	var image *hcloud.Image
	if opts.Image != "" {
		image, _, err = client.Image.GetForArchitecture(ctx, opts.Image, serverType.Architecture)
		if err != nil {
			return nil, fmt.Errorf("verifying image: %w", err)
		}
	}

	location, _, err := client.Location.Get(ctx, opts.Location)
	if err != nil {
		return nil, fmt.Errorf("verifying location: %w", err)
	}

	return &API{
		client:       client,
		imagesclient: imagesclient,

		serverType: serverType,
		image:      image,
		location:   location,
	}, nil
}

// CreateServer using the Hetzner API.
func (a *API) CreateServer(ctx context.Context, name, userdata string, sshKey *SSHKey, network *Network) (*Server, error) {
	opts := hcloud.ServerCreateOpts{
		Name:       name,
		ServerType: a.serverType,
		Image:      a.image,
		Location:   a.location,
		UserData:   userdata,
	}
	if sshKey != nil {
		opts.SSHKeys = []*hcloud.SSHKey{sshKey.SSHKey}
	}
	if network != nil {
		opts.Networks = []*hcloud.Network{network.Network}
	}

	result, _, err := a.client.Server.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to request new server: %w", err)
	}

	err = a.client.Action.WaitFor(ctx, append(result.NextActions, result.Action)...)
	if err != nil {
		return nil, fmt.Errorf("failed to create new server: %w", err)
	}

	s := result.Server

	if network != nil && len(s.PrivateNet) == 0 {
		// In some cases the private net might not be set on the server right after creation.
		s, _, err = a.client.Server.GetByID(ctx, result.Server.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated server: %w", err)
		}
	}

	return &Server{s}, nil
}

func (a *API) DeleteServer(ctx context.Context, id int64) error {
	server, _, err := a.client.Server.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find server: %w", err)
	}

	if server == nil {
		// No server with that ID exists in this project
		return nil
	}

	result, _, err := a.client.Server.DeleteWithResult(ctx, server)
	if err != nil {
		return fmt.Errorf("failed to request server delete: %w", err)
	}

	err = a.client.Action.WaitFor(ctx, result.Action)
	if err != nil {
		return fmt.Errorf("failed to delete server: %w", err)
	}

	return nil
}

func (a *API) CreateSSHKey(ctx context.Context, name, publicKey string) (*SSHKey, error) {
	sshKey, _, err := a.client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      name,
		PublicKey: publicKey,
		Labels:    DefaultLabels,
	})
	if err != nil {
		return nil, err
	}

	return &SSHKey{sshKey}, nil
}

func (a *API) DeleteSSHKey(ctx context.Context, key *SSHKey) error {
	_, err := a.client.SSHKey.Delete(ctx, key.SSHKey)
	return err
}

func (a *API) CreateNetwork(ctx context.Context, name string) (*Network, error) {
	_, ipnet, err := net.ParseCIDR("10.0.0.0/16")
	if err != nil {
		return nil, fmt.Errorf("parsing CIDR for network: %w", err)
	}

	network, _, err := a.client.Network.Create(ctx, hcloud.NetworkCreateOpts{
		Name:    name,
		IPRange: ipnet,
		Subnets: []hcloud.NetworkSubnet{{
			Type:        hcloud.NetworkSubnetTypeCloud,
			IPRange:     ipnet,
			NetworkZone: a.location.NetworkZone,
		}},
		Labels: DefaultLabels,
	})
	if err != nil {
		return nil, fmt.Errorf("creating network: %w", err)
	}

	return &Network{network}, nil
}

func (a *API) DeleteNetwork(ctx context.Context, network *Network) error {
	_, err := a.client.Network.Delete(ctx, network.Network)
	if err != nil {
		return fmt.Errorf("deleting network: %w", err)
	}

	return nil
}

func (a *API) UploadImage(ctx context.Context, name, path, board string) (int64, error) {
	opts := hcloudimages.UploadOptions{
		ImageCompression: hcloudimages.CompressionBZ2,
		Description:      hcloud.Ptr(name),
		Labels:           DefaultLabels,
	}

	parsedUrl, err := url.Parse(path)
	if err != nil {
		return 0, fmt.Errorf("parsing path: %w", err)
	}

	if parsedUrl.Scheme != "" {
		opts.ImageURL = parsedUrl
	} else {
		opts.ImageReader, err = os.Open(path)
		if err != nil {
			return 0, fmt.Errorf("opening local path: %w", err)
		}
	}

	switch board {
	case "amd64-usr":
		opts.Architecture = hcloud.ArchitectureX86
	case "arm64-usr":
		opts.Architecture = hcloud.ArchitectureARM
	}

	image, err := a.imagesclient.Upload(ctx, opts)
	if err != nil {
		return 0, fmt.Errorf("uploading the image: %w", err)
	}

	return image.ID, nil
}

// GC is the garbage collection (when we want to clear the project from resources created by Mantle (servers, images, etc.))
func (a *API) GC(ctx context.Context, gracePeriod time.Duration) error {
	createdCutoff := time.Now().Add(-gracePeriod)

	if err := a.gcServers(ctx, createdCutoff); err != nil {
		return fmt.Errorf("failed to gc servers: %w", err)
	}

	if err := a.gcImages(ctx, createdCutoff); err != nil {
		return fmt.Errorf("failed to gc servers: %w", err)
	}

	if err := a.gcSSHKeys(ctx, createdCutoff); err != nil {
		return fmt.Errorf("failed to gc ssh keys: %w", err)
	}

	if err := a.gcNetworks(ctx, createdCutoff); err != nil {
		return fmt.Errorf("failed to gc networks: %w", err)
	}

	return nil
}

func (a *API) gcServers(ctx context.Context, createdCutoff time.Time) error {
	servers, err := a.client.Server.AllWithOpts(ctx, hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: labelSelector(DefaultLabels),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list current servers: %w", err)
	}

	for _, server := range servers {
		if server.Created.After(createdCutoff) {
			continue
		}

		// Delete in series, could be made faster by triggering batches of deletes and then wait on the actions in parallel
		result, _, err := a.client.Server.DeleteWithResult(ctx, server)
		if err != nil {
			return fmt.Errorf("failed to request server delete: %w", err)
		}

		err = a.client.Action.WaitFor(ctx, result.Action)
		if err != nil {
			return fmt.Errorf("failed to delete server: %w", err)
		}
	}

	return nil
}

func (a *API) gcImages(ctx context.Context, createdCutoff time.Time) error {
	images, err := a.client.Image.AllWithOpts(ctx, hcloud.ImageListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: labelSelector(DefaultLabels),
		},
		Type: []hcloud.ImageType{hcloud.ImageTypeSnapshot},
	})
	if err != nil {
		return fmt.Errorf("failed to list current images: %w", err)
	}

	for _, image := range images {
		if image.Created.After(createdCutoff) {
			continue
		}

		_, err := a.client.Image.Delete(ctx, image)
		if err != nil {
			return fmt.Errorf("failed to delete image: %w", err)
		}
	}

	return nil
}

func (a *API) gcSSHKeys(ctx context.Context, createdCutoff time.Time) error {
	sshKeys, err := a.client.SSHKey.AllWithOpts(ctx, hcloud.SSHKeyListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: labelSelector(DefaultLabels),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list current ssh keys: %w", err)
	}

	for _, sshKey := range sshKeys {
		if sshKey.Created.After(createdCutoff) {
			continue
		}

		_, err := a.client.SSHKey.Delete(ctx, sshKey)
		if err != nil {
			return fmt.Errorf("failed to delete ssh key: %w", err)
		}
	}

	return nil
}

func (a *API) gcNetworks(ctx context.Context, createdCutoff time.Time) error {
	networks, err := a.client.Network.AllWithOpts(ctx, hcloud.NetworkListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: labelSelector(DefaultLabels),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list current networks: %w", err)
	}

	for _, network := range networks {
		if network.Created.After(createdCutoff) {
			continue
		}

		_, err := a.client.Network.Delete(ctx, network)
		if err != nil {
			return fmt.Errorf("failed to delete network: %w", err)
		}
	}

	return nil
}

// GetConsoleOutput returns the console output using API calls or other.
func (a *API) GetConsoleOutput(id string) (string, error) {
	// Hetzner Cloud API does not have an easy way to retrieve this.
	// There is only the VNC console that could be used to get the last x lines of output.
	return "", nil
}

func labelSelector(labels map[string]string) string {
	selectors := make([]string, 0, len(labels))

	for k, v := range labels {
		selectors = append(selectors, fmt.Sprintf("%s=%s", k, v))
	}

	// Reproducible result for tests
	sort.Strings(selectors)

	return strings.Join(selectors, ",")
}
