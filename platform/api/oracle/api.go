// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package oracle

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/flatcar/mantle/platform"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
)

var DefaultTags = map[string]string{
	"managed-by": "mantle",
}

// Options hold the specific Oracle Cloud Infrastructure options.
type Options struct {
	*platform.Options
	ConfigFile         string
	Profile            string
	CompartmentID      string
	AvailabilityDomain string
	SubnetID           string
	ImageID            string
	Shape              string
	OCPUs              float32
	MemoryGB           float32
}

// API is a wrapper around Oracle Cloud Infrastructure clients.
type API struct {
	opts           *Options
	compute        core.ComputeClient
	virtualNetwork core.VirtualNetworkClient
}

type Instance struct {
	core.Instance
	PublicIP  string
	PrivateIP string
}

func New(opts *Options) (*API, error) {
	configFile, err := expandPath(opts.ConfigFile)
	if err != nil {
		return nil, err
	}

	provider := common.CustomProfileConfigProvider(configFile, opts.Profile)

	compute, err := core.NewComputeClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("creating compute client: %w", err)
	}

	virtualNetwork, err := core.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("creating virtual network client: %w", err)
	}

	return &API{
		opts:           opts,
		compute:        compute,
		virtualNetwork: virtualNetwork,
	}, nil
}

func (a *API) CreateInstance(ctx context.Context, name, userData string) (*Instance, error) {
	encodedUserData := base64.StdEncoding.EncodeToString([]byte(userData))
	shapeConfig := core.LaunchInstanceShapeConfigDetails{
		Ocpus:       common.Float32(a.opts.OCPUs),
		MemoryInGBs: common.Float32(a.opts.MemoryGB),
	}

	resp, err := a.compute.LaunchInstance(ctx, core.LaunchInstanceRequest{
		LaunchInstanceDetails: core.LaunchInstanceDetails{
			AvailabilityDomain: common.String(a.opts.AvailabilityDomain),
			CompartmentId:      common.String(a.opts.CompartmentID),
			DisplayName:        common.String(name),
			FreeformTags:       DefaultTags,
			Metadata: map[string]string{
				"user_data": encodedUserData,
			},
			Shape:       common.String(a.opts.Shape),
			ShapeConfig: &shapeConfig,
			SourceDetails: core.InstanceSourceViaImageDetails{
				ImageId: common.String(a.opts.ImageID),
			},
			CreateVnicDetails: &core.CreateVnicDetails{
				AssignPublicIp: common.Bool(true),
				DisplayName:    common.String(name),
				FreeformTags:   DefaultTags,
				SubnetId:       common.String(a.opts.SubnetID),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("launching instance: %w", err)
	}

	instance, err := a.WaitForInstanceState(ctx, *resp.Instance.Id, core.InstanceLifecycleStateRunning)
	if err != nil {
		return nil, err
	}

	vnic, err := a.PrimaryVNIC(ctx, *instance.Id)
	if err != nil {
		return nil, err
	}

	return &Instance{
		Instance:  *instance,
		PublicIP:  stringValue(vnic.PublicIp),
		PrivateIP: stringValue(vnic.PrivateIp),
	}, nil
}

func (a *API) WaitForInstanceState(ctx context.Context, instanceID string, state core.InstanceLifecycleStateEnum) (*core.Instance, error) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(10 * time.Minute)
	defer timeout.Stop()

	for {
		resp, err := a.compute.GetInstance(ctx, core.GetInstanceRequest{
			InstanceId: common.String(instanceID),
		})
		if err != nil {
			return nil, fmt.Errorf("getting instance %q: %w", instanceID, err)
		}

		if resp.Instance.LifecycleState == state {
			return &resp.Instance, nil
		}

		switch resp.Instance.LifecycleState {
		case core.InstanceLifecycleStateTerminated, core.InstanceLifecycleStateTerminating:
			return nil, fmt.Errorf("instance %q entered state %q while waiting for %q", instanceID, resp.Instance.LifecycleState, state)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout.C:
			return nil, fmt.Errorf("timed out waiting for instance %q to reach %q", instanceID, state)
		case <-ticker.C:
		}
	}
}

func (a *API) PrimaryVNIC(ctx context.Context, instanceID string) (*core.Vnic, error) {
	attachments, err := a.compute.ListVnicAttachments(ctx, core.ListVnicAttachmentsRequest{
		CompartmentId: common.String(a.opts.CompartmentID),
		InstanceId:    common.String(instanceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing VNIC attachments for instance %q: %w", instanceID, err)
	}

	for _, attachment := range attachments.Items {
		if attachment.VnicId == nil {
			continue
		}

		resp, err := a.virtualNetwork.GetVnic(ctx, core.GetVnicRequest{
			VnicId: attachment.VnicId,
		})
		if err != nil {
			return nil, fmt.Errorf("getting VNIC %q: %w", *attachment.VnicId, err)
		}

		if resp.Vnic.IsPrimary != nil && *resp.Vnic.IsPrimary {
			return &resp.Vnic, nil
		}
	}

	return nil, fmt.Errorf("no primary VNIC found for instance %q", instanceID)
}

func (a *API) TerminateInstance(ctx context.Context, instanceID string) error {
	preserveBootVolume := false
	_, err := a.compute.TerminateInstance(ctx, core.TerminateInstanceRequest{
		InstanceId:         common.String(instanceID),
		PreserveBootVolume: &preserveBootVolume,
	})
	if err != nil {
		return fmt.Errorf("terminating instance %q: %w", instanceID, err)
	}

	return nil
}

func expandPath(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}
	if len(path) > 1 && path[1] != '/' {
		return "", fmt.Errorf("unsupported path %q", path)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	if len(path) == 1 {
		return home, nil
	}

	return filepath.Join(home, path[2:]), nil
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
