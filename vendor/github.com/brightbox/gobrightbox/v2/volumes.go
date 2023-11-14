package brightbox

import (
	"context"
	"path"
	"time"

	"github.com/brightbox/gobrightbox/v2/enums/filesystemtype"
	"github.com/brightbox/gobrightbox/v2/enums/storagetype"
	"github.com/brightbox/gobrightbox/v2/enums/volumestatus"
	"github.com/brightbox/gobrightbox/v2/enums/volumetype"
)

//go:generate ./generate_enum volumestatus creating attached detached deleting deleted failed
//go:generate ./generate_enum filesystemtype xfs ext4
//go:generate ./generate_enum volumetype image volume raw

// Volume represents a Brightbox Volume
// https://api.gb1.brightbox.com/1.0/#volume
type Volume struct {
	ResourceRef
	ID               string
	Name             string
	Status           volumestatus.Enum
	Description      string
	DeleteWithServer bool `json:"delete_with_server"`
	Boot             bool
	Encrypted        bool
	FilesystemLabel  string              `json:"filesystem_label"`
	FilesystemType   filesystemtype.Enum `json:"filesystem_type"`
	Locked           bool
	Serial           string
	Size             uint
	Source           string
	SourceType       volumetype.Enum  `json:"source_type"`
	StorageType      storagetype.Enum `json:"storage_type"`
	CreatedAt        *time.Time       `json:"created_at"`
	DeletedAt        *time.Time       `json:"deleted_at"`
	UpdatedAt        *time.Time       `json:"updated_at"`
	Server           *Server
	Account          *Account
	Image            *Image
}

// VolumeOptions is used to create and update volumes
// create and update servers.
type VolumeOptions struct {
	ID               string              `json:"-"`
	Name             *string             `json:"name,omitempty"`
	Description      *string             `json:"description,omitempty"`
	Serial           *string             `json:"serial,omitempty"`
	DeleteWithServer *bool               `json:"delete_with_server,omitempty"`
	FilesystemLabel  *string             `json:"filesystem_label,omitempty"`
	FilesystemType   filesystemtype.Enum `json:"filesystem_type,omitempty"`
	Size             *uint               `json:"size,omitempty"`
	Image            *string             `json:"image,omitempty"`
	Encrypted        *bool               `json:"encrypted,omitempty"`
}

// VolumeAttachment is used in conjunction with AttachVolume and DetachVolume
type VolumeAttachment struct {
	Server string `json:"server"`
	Boot   bool   `json:"boot"`
}

// VolumeNewSize is used in conjunction with ResizeVolume
// to specify the change in the disk size
type VolumeNewSize struct {
	From uint `json:"from"`
	To   uint `json:"to"`
}

// AttachVolume issues a request to attach the volume to a particular server and
// optionally mark it as the boot volume
func (c *Client) AttachVolume(ctx context.Context, identifier string, attachment VolumeAttachment) (*Volume, error) {
	return apiPost[Volume](
		ctx,
		c,
		path.Join(volumeAPIPath, identifier, "attach"),
		attachment,
	)
}

// DetachVolume issues a request to disconnect a volume from a server
func (c *Client) DetachVolume(ctx context.Context, identifier string) (*Volume, error) {
	return apiPost[Volume](
		ctx,
		c,
		path.Join(volumeAPIPath, identifier, "detach"),
		nil,
	)
}
