package brightbox

import (
	"time"

	"github.com/brightbox/gobrightbox/v2/enums/arch"
	"github.com/brightbox/gobrightbox/v2/enums/imagestatus"
	"github.com/brightbox/gobrightbox/v2/enums/sourcetrigger"
	"github.com/brightbox/gobrightbox/v2/enums/sourcetype"
)

//go:generate ./generate_enum imagestatus creating available deprecated unavailable deleting deleted failed
//go:generate ./generate_enum arch x86_64 i686
//go:generate ./generate_enum sourcetrigger manual schedule
//go:generate ./generate_enum sourcetype upload snapshot

// Image represents a Machine Image
// https://api.gb1.brightbox.com/1.0/#image
type Image struct {
	ResourceRef
	ID                string
	Name              string
	Username          string
	Status            imagestatus.Enum
	Locked            bool
	Description       string
	Source            string
	Arch              arch.Enum
	Official          bool
	Public            bool
	Owner             string
	SourceTrigger     sourcetrigger.Enum `json:"source_trigger"`
	SourceType        sourcetype.Enum    `json:"source_type"`
	VirtualSize       uint               `json:"virtual_size"`
	DiskSize          uint               `json:"disk_size"`
	MinRAM            *uint              `json:"min_ram"`
	CompatibilityMode bool               `json:"compatibility_mode"`
	LicenceName       string             `json:"licence_name"`
	CreatedAt         *time.Time         `json:"created_at"`
	Ancestor          *Image
}

// ImageOptions is used to create and update machine images
type ImageOptions struct {
	ID                string           `json:"-"`
	Name              *string          `json:"name,omitempty"`
	Username          *string          `json:"username,omitempty"`
	Description       *string          `json:"description,omitempty"`
	MinRAM            *uint            `json:"min_ram,omitempty"`
	Server            string           `json:"server,omitempty"`
	Volume            string           `json:"volume,omitempty"`
	Arch              arch.Enum        `json:"arch,omitempty"`
	Status            imagestatus.Enum `json:"status,omitempty"`
	Public            *bool            `json:"public,omitempty"`
	CompatibilityMode *bool            `json:"compatibility_mode,omitempty"`
	URL               string           `json:"http_url,omitempty"`
}
