// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0
package scaleway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/pkg/capnslog"
	"github.com/flatcar/mantle/util"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

const (
	userDataKey = "cloud-init"
)

var (
	plog = capnslog.NewPackageLogger("github.com/flatcar/mantle", "platform/api/scaleway")
	tags = []string{"mantle"}
)

// Server is a wrapper around Scaleway instance Server struct.
type Server struct {
	*instance.Server
}

// CreateServer acts in four steps:
// 1. Create the server
// 2. Set the metadata on the server
// 3. Start the server
// 4. Get the server to update the IP fields
func (a *API) CreateServer(ctx context.Context, name, userData string) (*Server, error) {
	volumeName := "flatcar_production_scaleway_image.qcow2"
	res, err := a.instance.CreateServer(&instance.CreateServerRequest{
		Name: name,
		Tags: tags,
		Volumes: map[string]*instance.VolumeServerTemplate{
			"0": {
				BaseSnapshot: &a.opts.Image,
				VolumeType:   instance.VolumeVolumeTypeLSSD,
				Name:         &volumeName,
			},
		},
		CommercialType: a.opts.InstanceType,
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("creating server: %w", err)
	}

	if res.Server == nil {
		return nil, errors.New("unable to get server from API response")
	}

	id := res.Server.ID

	plog.Infof("setting server userdata: %s", id)
	if err := a.instance.SetServerUserData(&instance.SetServerUserDataRequest{
		ServerID: id,
		Key:      userDataKey,
		Content:  strings.NewReader(userData),
	}, scw.WithContext(ctx)); err != nil {
		return nil, fmt.Errorf("setting user-data configuration: %w", err)
	}

	timeout := 2 * time.Minute
	if err := a.instance.ServerActionAndWait(&instance.ServerActionAndWaitRequest{
		ServerID: id,
		Action:   instance.ServerActionPoweron,
		Timeout:  &timeout,
	}, scw.WithContext(ctx)); err != nil {
		return nil, fmt.Errorf("starting server: %w", err)
	}
	plog.Infof("server started: %s", id)

	// This is required to get an IP.
	sres, err := a.instance.GetServer(&instance.GetServerRequest{
		ServerID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("refreshing server data: %w", err)
	}

	if sres.Server == nil {
		return nil, errors.New("unable to refresh server data from API response")
	}

	return &Server{sres.Server}, nil
}

// DeleteServer acts in three steps:
// 1. Power off the instance
// 2. Actually delete the server
// 3. Delete the associated volumes
func (a *API) DeleteServer(ctx context.Context, id string) error {
	timeout := 2 * time.Minute
	if err := a.instance.ServerActionAndWait(&instance.ServerActionAndWaitRequest{
		ServerID: id,
		Action:   instance.ServerActionPoweroff,
		Timeout:  &timeout,
	}, scw.WithContext(ctx)); err != nil {
		return fmt.Errorf("stopping server: %w", err)
	}

	res, err := a.instance.GetServer(&instance.GetServerRequest{
		ServerID: id,
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("refreshing server data: %w", err)
	}

	if res.Server == nil {
		return errors.New("unable to refresh server data from API response")
	}

	if err := a.instance.DeleteServer(&instance.DeleteServerRequest{
		ServerID: id,
	}, scw.WithContext(ctx)); err != nil {
		return fmt.Errorf("deleting server: %w", err)
	}

	for _, volume := range res.Server.Volumes {
		if err := a.instance.DeleteVolume(&instance.DeleteVolumeRequest{
			VolumeID: volume.ID,
		}, scw.WithContext(ctx)); err != nil {
			return fmt.Errorf("deleting volume: %w", err)
		}
	}

	return nil
}

func (a *API) DeleteSnapshot(ctx context.Context, id string) error {
	if err := a.instance.DeleteSnapshot(&instance.DeleteSnapshotRequest{
		SnapshotID: id,
	}, scw.WithContext(ctx)); err != nil {
		return fmt.Errorf("deleting snapshot: %w", err)
	}

	return nil
}

// CreateSnapshot will create the snapshot used by the instance.
// It waits for the snapshot to be available before returning its ID.
func (a *API) CreateSnapshot(ctx context.Context, bucket, key string) (string, error) {
	res, err := a.instance.CreateSnapshot(&instance.CreateSnapshotRequest{
		Name:       key,
		Bucket:     &bucket,
		Key:        &key,
		VolumeType: instance.SnapshotVolumeTypeLSSD,
		Tags:       &tags,
	}, scw.WithContext(ctx))
	if err != nil {
		return "", fmt.Errorf("creating snapshot: %v", err)
	}

	if res.Snapshot == nil {
		return "", errors.New("unable to get snapshot from API response")
	}

	snapshot := res.Snapshot
	plog.Infof("snaphost created: %s", snapshot.ID)

	if err := util.WaitUntilReady(5*time.Minute, 5*time.Second, func() (bool, error) {
		res, err := a.instance.GetSnapshot(&instance.GetSnapshotRequest{
			SnapshotID: snapshot.ID,
		}, scw.WithContext(ctx))
		if err != nil {
			return false, fmt.Errorf("getting snapshot: %v", err)
		}

		if res.Snapshot == nil {
			return false, errors.New("unable to get snapshot from API response")
		}

		return res.Snapshot.State == instance.SnapshotStateAvailable, nil
	}); err != nil {
		return "", fmt.Errorf("getting snapshot available: %w", err)
	}

	plog.Infof("snaphost ready: %s", snapshot.ID)
	return snapshot.ID, nil
}

func (a *API) GC(ctx context.Context, gracePeriod time.Duration) error {
	threshold := time.Now().Add(-gracePeriod)

	servers, err := a.instance.ListServers(&instance.ListServersRequest{
		Tags: tags,
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("listing servers: %w", err)
	}

	if servers.Servers == nil {
		return errors.New("unable to list servers from API response")
	}

	for _, server := range servers.Servers {
		if server.CreationDate.After(threshold) {
			continue
		}

		plog.Infof("deleting server: %s", server.ID)
		if err := a.DeleteServer(ctx, server.ID); err != nil {
			return err
		}
	}

	// SDK / APIs are not consistent regarding the tags.
	// CreateServer -> []string
	// CreateSnapshot -> *[]string
	// ListSnapshot -> *string
	t := strings.Join(tags, " ")
	snapshots, err := a.instance.ListSnapshots(&instance.ListSnapshotsRequest{
		Tags: &t,
	}, scw.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("listing snapshots: %w", err)
	}

	if servers.Servers == nil {
		return errors.New("unable to list snapshots from API response")
	}

	for _, snapshot := range snapshots.Snapshots {
		if snapshot.CreationDate.After(threshold) {
			continue
		}

		plog.Infof("deleting snapshot: %s", snapshot.ID)
		if err := a.DeleteSnapshot(ctx, snapshot.ID); err != nil {
			return err
		}
	}

	return nil
}
