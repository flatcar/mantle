// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package akamai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/linode/linodego"

	"github.com/coreos/pkg/capnslog"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/api/akamai")
	tags = []string{"mantle"}
)

// Server is a wrapper around Akamai instance Server struct.
type Server struct {
	*linodego.Instance
}

// CreateServer is running a few steps:
// 1. Create the linode (i.e) the instance but it does not boot it
// 2. Create a custom disk that uses the Flatcar image previously uploaded
// 3. Resize the disk to match the expected disk size of the instance
// 4. Create a custom configuration to directly boot on the disk
// 5. Boot the instance
func (a *API) CreateServer(ctx context.Context, name, userData string) (*Server, error) {
	booted := false
	opts := linodego.InstanceCreateOptions{
		Label:  name,
		Region: a.opts.Region,
		Type:   a.opts.Type,
		Tags:   tags,
		Metadata: &linodego.InstanceMetadataOptions{
			UserData: base64.StdEncoding.EncodeToString([]byte(userData)),
		},
		Booted:    &booted,
		PrivateIP: true,
	}

	plog.Infof("Creating the instance")
	instance, err := a.client.CreateInstance(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("creating instance: %w", err)
	}

	t, err := a.client.GetType(ctx, a.opts.Type)
	if err != nil {
		return nil, fmt.Errorf("getting instance type: %w", err)
	}

	// RootPass is required when you create a disk from an image. It is not used by Flatcar.
	instanceDiskCreateOptions := linodego.InstanceDiskCreateOptions{
		Size:     t.Disk,
		Label:    name,
		Image:    a.opts.Image,
		RootPass: "ThisPasswordIsNotUsedButRequiredByAkamai1234",
	}

	disk, err := a.client.CreateInstanceDisk(ctx, instance.ID, instanceDiskCreateOptions)
	if err != nil {
		return nil, fmt.Errorf("creating instance disk: %w", err)
	}

	plog.Infof("Waiting for the instance to be ready: %v", instance.ID)
	disk, err = a.client.WaitForInstanceDiskStatus(ctx, instance.ID, disk.ID, linodego.DiskReady, 100)
	if err != nil {
		return nil, fmt.Errorf("waiting for instance disk to be ready: %w", err)
	}

	root := "sda"
	instanceConfigCreateOptions := linodego.InstanceConfigCreateOptions{
		Label:    "default",
		Comments: "Created by Mantle",
		Devices: linodego.InstanceConfigDeviceMap{
			SDA: &linodego.InstanceConfigDevice{
				DiskID: disk.ID,
			},
		},
		Helpers: &linodego.InstanceConfigHelpers{
			DevTmpFsAutomount: false,
			Network:           false,
			ModulesDep:        false,
			Distro:            false,
			UpdateDBDisabled:  true,
		},
		Kernel:     "linode/direct-disk",
		RootDevice: &root,
	}

	cfg, err := a.client.CreateInstanceConfig(ctx, instance.ID, instanceConfigCreateOptions)
	if err != nil {
		return nil, fmt.Errorf("creating instance configuration: %w", err)
	}

	plog.Infof("Booting the instance")
	if err := a.client.BootInstance(ctx, instance.ID, cfg.ID); err != nil {
		return nil, fmt.Errorf("booting the instance: %w", err)
	}

	return &Server{instance}, nil
}

func (a *API) DeleteServer(ctx context.Context, id string) error {
	instanceID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("converting instance ID to integer: %w", err)
	}

	if err := a.client.DeleteInstance(ctx, instanceID); err != nil {
		return fmt.Errorf("deleting the instance: %w", err)
	}

	return nil
}

func (a *API) DeleteImage(ctx context.Context, id string) error {
	if err := a.client.DeleteImage(ctx, id); err != nil {
		return fmt.Errorf("deleting the image: %w", err)
	}

	return nil
}

func (a *API) GC(ctx context.Context, gracePeriod time.Duration) error {
	threshold := time.Now().Add(-gracePeriod)

	t := strings.Join(tags, ",")
	f := map[string]string{
		"tags": t,
	}
	filter, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("marshalling filter: %w", err)
	}

	plog.Infof("listing instances with filter: %s", string(filter))

	instances, err := a.client.ListInstances(ctx, &linodego.ListOptions{
		Filter: string(filter),
	})
	if err != nil {
		return fmt.Errorf("getting instances list: %w", err)
	}

	for _, instance := range instances {
		if instance.Created.After(threshold) {
			continue
		}

		plog.Infof("deleting instance: %d", instance.ID)
		if err := a.DeleteServer(ctx, strconv.Itoa(instance.ID)); err != nil {
			return err
		}

	}

	images, err := a.client.ListImages(ctx, &linodego.ListOptions{
		Filter: string(filter),
	})
	if err != nil {
		return fmt.Errorf("getting images list: %w", err)
	}

	for _, image := range images {
		if image.Created.After(threshold) {
			continue
		}

		plog.Infof("deleting image: %s", image.ID)
		if err := a.DeleteImage(ctx, image.ID); err != nil {
			return err
		}
	}

	return nil
}
