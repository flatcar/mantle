// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package brightbox

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	brightbox "github.com/brightbox/gobrightbox/v2"
	"github.com/brightbox/gobrightbox/v2/clientcredentials"
	"github.com/brightbox/gobrightbox/v2/enums/arch"
	"github.com/brightbox/gobrightbox/v2/enums/imagestatus"
	"github.com/brightbox/gobrightbox/v2/enums/serverstatus"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/util"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/api/brightbox")
)

type Options struct {
	*platform.Options

	// ClientID is the ID of the API client.
	ClientID string
	// ClientSecret is the secret of the API client.
	ClientSecret string
	// Image is the image to deploy.
	Image string
	// ServerType is the amount of memory and type of server (e.g 2gb.ssd).
	ServerType string
}

type API struct {
	client *brightbox.Client
	opts   *Options
}

type Server struct{ *brightbox.Server }

func New(opts *Options) (*API, error) {
	// Setup OAuth2 authentication
	conf := &clientcredentials.Config{
		ID:     opts.ClientID,
		Secret: opts.ClientSecret,
	}

	ctx := context.Background()

	client, err := brightbox.Connect(ctx, conf)
	if err != nil {
		return nil, fmt.Errorf("connecting to Brightbox: %w", err)
	}

	return &API{
		client: client,
		opts:   opts,
	}, nil
}

func (a *API) AddKey(name, key string) error {
	return nil
}

// CreateServer using the Brightbox API.
func (a *API) CreateServer(ctx context.Context, name, userdata, cloudIP string) (*Server, error) {
	// Not in the spec, but userdata needs to be base64 encoded.
	userdata = base64.StdEncoding.EncodeToString([]byte(userdata))

	s, err := a.client.CreateServer(ctx, brightbox.ServerOptions{
		Image:      &a.opts.Image,
		Name:       &name,
		UserData:   &userdata,
		ServerType: &a.opts.ServerType,
	})
	if err != nil {
		return nil, fmt.Errorf("creating server from API: %w", err)
	}

	// If the cloud IP already exists, we reuse it - otherwise we create a new one.
	if cloudIP == "" {
		plog.Info("No cloud IP already available: creating a new one.")
		cip, err := a.client.CreateCloudIP(ctx, brightbox.CloudIPOptions{})
		if err != nil {
			return nil, fmt.Errorf("creating a cloud IP from API: %w", err)
		}

		cloudIP = cip.ID
	}

	// Let's assign this IP to this new server.
	if _, err := a.client.MapCloudIP(ctx, cloudIP, brightbox.CloudIPAttachment{Destination: s.ID}); err != nil {
		_ = a.DeleteServer(ctx, s.ID)
		return nil, fmt.Errorf("mapping cloud IP to server: %w", err)
	}

	// Refetch the server to get the new information regarding the freshly assigned cloud IP.
	s, err = a.client.Server(ctx, s.ID)
	if err != nil {
		_ = a.DeleteServer(ctx, s.ID)
		return nil, fmt.Errorf("getting server from API: %w", err)
	}

	return &Server{s}, nil
}

func (a *API) DeleteKey(name string) error {
	return nil
}

// DeleteImage will remove the image from Brightbox.
func (a *API) DeleteImage(ctx context.Context, id string) error {
	if _, err := a.client.DestroyImage(ctx, id); err != nil {
		return fmt.Errorf("destroying image from API: %w", err)
	}

	return nil
}

// DeleteCloudIP will remove a cloud IP from Brightbox.
func (a *API) DeleteCloudIP(ctx context.Context, id string) error {
	if _, err := a.client.DestroyCloudIP(ctx, id); err != nil {
		return fmt.Errorf("destroying cloud IP from API: %w", err)
	}

	return nil
}

func (a *API) DeleteServer(ctx context.Context, id string) error {
	// Let's first unassign the cloud IP.
	s, err := a.client.Server(ctx, id)
	if err != nil {
		return fmt.Errorf("getting server from API: %w", err)
	}

	var cloudIP string
	if s != nil && len(s.CloudIPs) >= 1 {
		cloudIP = s.CloudIPs[0].ID
	}

	if cloudIP != "" {
		if _, err := a.client.UnMapCloudIP(ctx, cloudIP); err != nil {
			return fmt.Errorf("unmaping cloud IP from API: %w", err)
		}
		plog.Info("Cloud IP released.")
	}

	if _, err := a.client.DestroyServer(ctx, id); err != nil {
		return fmt.Errorf("destroying server from API: %w", err)
	}

	return nil
}

func (a *API) GC(ctx context.Context, gracePeriod time.Duration) error {
	threshold := time.Now().Add(-gracePeriod)
	// TODO: CloudIP has no creation date for now.
	// We can't safely delete "old" cloud IPs.
	// NOTE: Currently, cloud IPs removal is implemented as an independant
	// 'ore' subcommand.

	servers, err := a.client.Servers(ctx)
	if err != nil {
		return fmt.Errorf("listing servers from API: %w", err)
	}

	for _, server := range servers {
		if server.Status == serverstatus.Deleted || server.CreatedAt.After(threshold) {
			continue
		}

		if err := a.DeleteServer(ctx, server.ID); err != nil {
			return fmt.Errorf("deleting server: %w", err)
		}
	}

	images, err := a.client.Images(ctx)
	if err != nil {
		return fmt.Errorf("listing servers from API: %w", err)
	}

	for _, image := range images {
		if image.Public || image.Status == imagestatus.Deleted || image.CreatedAt.After(threshold) {
			continue
		}

		if err := a.DeleteImage(ctx, image.ID); err != nil {
			return fmt.Errorf("deleting image: %w", err)
		}
	}

	return nil
}

func (a *API) GetConsoleOutput(id string) (string, error) {
	// NOTE: There is no way to get console output from the API.
	// A workaround would be the fetch the console_url + console_token to read from this
	// endpoint.
	return "", nil
}

// UploadImage will upload an image from the URL on Brightbox and wait for it to become
// available.
func (a *API) UploadImage(ctx context.Context, name, URL string) (string, error) {
	defaultUsername := "core"

	img, err := a.client.CreateImage(ctx, brightbox.ImageOptions{
		Name:     &name,
		URL:      URL,
		Username: &defaultUsername,
		Arch:     arch.X86_64,
	})
	if err != nil {
		return "", fmt.Errorf("creating image from API: %w", err)
	}

	// It usually takes around 20 seconds to extract the image.
	if err := util.WaitUntilReady(2*time.Minute, 5*time.Second, func() (bool, error) {
		image, err := a.client.Image(ctx, img.ID)
		if err != nil {
			return false, fmt.Errorf("getting image status: %w", err)
		}

		return image.Status == imagestatus.Available, nil
	}); err != nil {
		a.DeleteImage(ctx, img.ID)
		return "", fmt.Errorf("getting image active: %w", err)
	}

	return img.ID, nil
}

// RemoveCloudIPs remove any left overs IPs.
func (a *API) RemoveCloudIPs(ctx context.Context) error {
	cloudIPs, err := a.client.CloudIPs(ctx)
	if err != nil {
		return fmt.Errorf("getting cloud IPs: %w", err)
	}

	for _, cloudIP := range cloudIPs {
		if err := a.DeleteCloudIP(ctx, cloudIP.ID); err != nil {
			return fmt.Errorf("deleting cloud IP: %w", err)
		}
	}

	return nil
}
