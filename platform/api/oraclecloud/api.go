// Copyright The Mantle Authors.
// SPDX-License-Identifier: Apache-2.0

package oraclecloud

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flatcar/mantle/platform"
	"github.com/flatcar/mantle/util"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
)

var DefaultTags = map[string]string{
	"managed-by": "mantle",
}

// Options hold the specific Oracle Cloud Infrastructure options.
type Options struct {
	*platform.Options
	Tenancy              string
	User                 string
	Fingerprint          string
	PrivateKey           string
	PrivateKeyPassphrase string
	Region               string

	CompartmentID      string
	AvailabilityDomain string
	SubnetID           string
	ImageID            string
	Shape              string
	OCPUs              float32
	MemoryGB           float32
	Namespace          string
	Bucket             string
}

// API is a wrapper around Oracle Cloud Infrastructure clients.
type API struct {
	opts           *Options
	compute        core.ComputeClient
	objectStorage  objectstorage.ObjectStorageClient
	virtualNetwork core.VirtualNetworkClient
}

type Instance struct {
	core.Instance
	PublicIP  string
	PrivateIP string
}

func New(opts *Options) (*API, error) {
	provider := common.NewRawConfigurationProvider(opts.Tenancy, opts.User, opts.Region, opts.Fingerprint, opts.PrivateKey, nil)

	compute, err := core.NewComputeClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("creating compute client: %w", err)
	}

	virtualNetwork, err := core.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("creating virtual network client: %w", err)
	}
	objectStorage, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("creating object storage client: %w", err)
	}

	// DefaultRetryPolicy retries selected 409 responses, 429 responses, and
	// most 5xx responses. It makes up to eight attempts using exponential
	// backoff with jitter, capped at about 30 seconds per delay. It also handles
	// eventual-consistency failures with up to nine attempts. See:
	// https://pkg.go.dev/github.com/oracle/oci-go-sdk/v65/common#DefaultRetryPolicy
	retryPolicy := common.DefaultRetryPolicy()
	retryConfig := common.CustomClientConfiguration{
		RetryPolicy: &retryPolicy,
	}
	compute.SetCustomClientConfiguration(retryConfig)
	virtualNetwork.SetCustomClientConfiguration(retryConfig)
	objectStorage.SetCustomClientConfiguration(retryConfig)

	return &API{
		opts:           opts,
		compute:        compute,
		objectStorage:  objectStorage,
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
		return nil, fmt.Errorf("getting primary vnic: %w", err)
	}

	return &Instance{
		Instance:  *instance,
		PublicIP:  stringValue(vnic.PublicIp),
		PrivateIP: stringValue(vnic.PrivateIp),
	}, nil
}

func (a *API) WaitForInstanceState(ctx context.Context, instanceID string, state core.InstanceLifecycleStateEnum) (*core.Instance, error) {
	var instance *core.Instance
	var terminalErr error
	err := util.Retry(60, 10*time.Second, func() error {
		resp, err := a.compute.GetInstance(ctx, core.GetInstanceRequest{
			InstanceId: common.String(instanceID),
		})
		if err != nil {
			if ctx.Err() != nil {
				terminalErr = ctx.Err()
				return nil
			}
			return fmt.Errorf("getting instance %q: %w", instanceID, err)
		}

		if resp.Instance.LifecycleState == state {
			instance = &resp.Instance
			return nil
		}

		switch resp.Instance.LifecycleState {
		case core.InstanceLifecycleStateTerminated, core.InstanceLifecycleStateTerminating:
			terminalErr = fmt.Errorf("instance %q entered state %q while waiting for %q", instanceID, resp.Instance.LifecycleState, state)
			return nil
		}

		return fmt.Errorf("instance %q is in state %q, waiting for %q", instanceID, resp.Instance.LifecycleState, state)
	})
	if terminalErr != nil {
		return nil, terminalErr
	}
	if err != nil {
		return nil, fmt.Errorf("waiting for instance %q to reach %q: %w", instanceID, state, err)
	}
	return instance, nil
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

func (a *API) UploadImage(ctx context.Context, name, path, objectName, sourceImageType string) (string, error) {
	if a.opts.CompartmentID == "" {
		return "", fmt.Errorf("compartment ID is required")
	}
	if a.opts.Bucket == "" {
		return "", fmt.Errorf("bucket is required")
	}

	namespace := a.opts.Namespace
	if namespace == "" {
		var err error
		namespace, err = a.GetNamespace(ctx)
		if err != nil {
			return "", err
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening image %q: %w", path, err)
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat image %q: %w", path, err)
	}

	if objectName == "" {
		objectName = filepath.Base(path)
	}

	if err := a.UploadObject(ctx, namespace, a.opts.Bucket, objectName, f, st.Size()); err != nil {
		return "", err
	}

	imageID, err := a.CreateImageFromObject(ctx, name, namespace, a.opts.Bucket, objectName, sourceImageType)
	if err != nil {
		if deleteErr := a.DeleteObject(ctx, namespace, a.opts.Bucket, objectName); deleteErr != nil {
			return "", fmt.Errorf("%w; additionally failed deleting uploaded object %q: %v", err, objectName, deleteErr)
		}
		return "", err
	}

	if _, err := a.WaitForImageState(ctx, imageID, core.ImageLifecycleStateAvailable); err != nil {
		return "", fmt.Errorf("waiting for image import %q: %w; uploaded object %q was left in bucket %q", imageID, err, objectName, a.opts.Bucket)
	}

	if err := a.DeleteObject(ctx, namespace, a.opts.Bucket, objectName); err != nil {
		return "", fmt.Errorf("deleting uploaded object %q: %w", objectName, err)
	}

	return imageID, nil
}

func (a *API) GetNamespace(ctx context.Context) (string, error) {
	resp, err := a.objectStorage.GetNamespace(ctx, objectstorage.GetNamespaceRequest{
		CompartmentId: common.String(a.opts.CompartmentID),
	})
	if err != nil {
		return "", fmt.Errorf("getting object storage namespace: %w", err)
	}
	if resp.Value == nil || *resp.Value == "" {
		return "", fmt.Errorf("object storage namespace is empty")
	}
	return *resp.Value, nil
}

func (a *API) UploadObject(ctx context.Context, namespace, bucket, objectName string, body io.ReadCloser, size int64) error {
	_, err := a.objectStorage.PutObject(ctx, objectstorage.PutObjectRequest{
		NamespaceName: common.String(namespace),
		BucketName:    common.String(bucket),
		ObjectName:    common.String(objectName),
		ContentLength: common.Int64(size),
		PutObjectBody: body,
		ContentType:   common.String("application/octet-stream"),
	})
	if err != nil {
		return fmt.Errorf("uploading object %q to bucket %q: %w", objectName, bucket, err)
	}
	return nil
}

func (a *API) DeleteObject(ctx context.Context, namespace, bucket, objectName string) error {
	_, err := a.objectStorage.DeleteObject(ctx, objectstorage.DeleteObjectRequest{
		NamespaceName: common.String(namespace),
		BucketName:    common.String(bucket),
		ObjectName:    common.String(objectName),
	})
	if err != nil {
		return fmt.Errorf("deleting object %q from bucket %q: %w", objectName, bucket, err)
	}
	return nil
}

func (a *API) CreateImageFromObject(ctx context.Context, name, namespace, bucket, objectName, sourceImageType string) (string, error) {
	sourceType, err := parseSourceImageType(sourceImageType)
	if err != nil {
		return "", err
	}

	resp, err := a.compute.CreateImage(ctx, core.CreateImageRequest{
		CreateImageDetails: core.CreateImageDetails{
			CompartmentId: common.String(a.opts.CompartmentID),
			DisplayName:   common.String(name),
			FreeformTags:  DefaultTags,
			ImageSourceDetails: core.ImageSourceViaObjectStorageTupleDetails{
				BucketName:             common.String(bucket),
				NamespaceName:          common.String(namespace),
				ObjectName:             common.String(objectName),
				OperatingSystem:        common.String("Flatcar"),
				OperatingSystemVersion: common.String("Container Linux"),
				SourceImageType:        sourceType,
			},
			LaunchMode: core.CreateImageDetailsLaunchModeParavirtualized,
		},
	})
	if err != nil {
		return "", fmt.Errorf("creating image from object %q: %w", objectName, err)
	}
	if resp.Image.Id == nil {
		return "", fmt.Errorf("created image response did not include an image ID")
	}
	return *resp.Image.Id, nil
}

func (a *API) WaitForImageState(ctx context.Context, imageID string, state core.ImageLifecycleStateEnum) (*core.Image, error) {
	var image *core.Image
	var terminalErr error
	err := util.Retry(240, 30*time.Second, func() error {
		resp, err := a.compute.GetImage(ctx, core.GetImageRequest{
			ImageId: common.String(imageID),
		})
		if err != nil {
			if ctx.Err() != nil {
				terminalErr = ctx.Err()
				return nil
			}
			return fmt.Errorf("getting image %q: %w", imageID, err)
		}

		if resp.Image.LifecycleState == state {
			image = &resp.Image
			return nil
		}

		switch resp.Image.LifecycleState {
		case core.ImageLifecycleStateDeleted, core.ImageLifecycleStateDisabled:
			terminalErr = fmt.Errorf("image %q entered state %q while waiting for %q", imageID, resp.Image.LifecycleState, state)
			return nil
		}

		return fmt.Errorf("image %q is in state %q, waiting for %q", imageID, resp.Image.LifecycleState, state)
	})
	if terminalErr != nil {
		return nil, terminalErr
	}
	if err != nil {
		return nil, fmt.Errorf("waiting for image %q to reach %q: %w", imageID, state, err)
	}
	return image, nil
}

func (a *API) GC(ctx context.Context, gracePeriod time.Duration) error {
	createdCutoff := time.Now().Add(-gracePeriod)

	if err := a.gcImages(ctx, createdCutoff); err != nil {
		return fmt.Errorf("failed to gc images: %w", err)
	}

	if err := a.gcInstances(ctx, createdCutoff); err != nil {
		return fmt.Errorf("failed to gc instances: %w", err)
	}

	return nil
}

func (a *API) gcInstances(ctx context.Context, createdCutoff time.Time) error {

	var page *string
	for {
		resp, err := a.compute.ListInstances(ctx, core.ListInstancesRequest{
			CompartmentId: common.String(a.opts.CompartmentID),
			Page:          page,
		})
		if err != nil {
			return fmt.Errorf("listing instances: %w", err)
		}

		for _, instance := range resp.Items {
			if instance.LifecycleState == core.InstanceLifecycleStateTerminated || instance.LifecycleState == core.InstanceLifecycleStateTerminating {
				continue
			}
			if instance.FreeformTags["managed-by"] != "mantle" {
				continue
			}
			if instance.TimeCreated == nil || instance.TimeCreated.After(createdCutoff) {
				continue
			}
			if instance.Id == nil {
				continue
			}

			if err := a.TerminateInstance(ctx, *instance.Id); err != nil {
				return fmt.Errorf("terminating instance %q: %w", *instance.Id, err)
			}
		}

		if resp.OpcNextPage == nil {
			break
		}
		page = resp.OpcNextPage
	}

	return nil
}

func (a *API) gcImages(ctx context.Context, createdCutoff time.Time) error {
	var page *string
	for {
		resp, err := a.compute.ListImages(ctx, core.ListImagesRequest{
			CompartmentId: common.String(a.opts.CompartmentID),
			Page:          page,
		})
		if err != nil {
			return fmt.Errorf("listing images: %w", err)
		}

		for _, image := range resp.Items {
			if image.LifecycleState == core.ImageLifecycleStateDeleted {
				continue
			}
			if image.FreeformTags["managed-by"] != "mantle" {
				continue
			}
			if image.TimeCreated == nil || image.TimeCreated.After(createdCutoff) {
				continue
			}
			if image.Id == nil {
				continue
			}

			_, err := a.compute.DeleteImage(ctx, core.DeleteImageRequest{
				ImageId: image.Id,
			})
			if err != nil {
				return fmt.Errorf("deleting image %q: %w", *image.Id, err)
			}
		}

		if resp.OpcNextPage == nil {
			break
		}
		page = resp.OpcNextPage
	}

	return nil
}

func parseSourceImageType(sourceImageType string) (core.ImageSourceDetailsSourceImageTypeEnum, error) {
	switch strings.ToUpper(sourceImageType) {
	case "", "QCOW2":
		return core.ImageSourceDetailsSourceImageTypeQcow2, nil
	case "VMDK":
		return core.ImageSourceDetailsSourceImageTypeVmdk, nil
	default:
		return "", fmt.Errorf("unsupported source image type %q", sourceImageType)
	}
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
