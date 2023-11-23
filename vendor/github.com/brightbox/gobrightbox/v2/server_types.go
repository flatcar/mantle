package brightbox

import (
	"github.com/brightbox/gobrightbox/v2/enums/servertypestatus"
	"github.com/brightbox/gobrightbox/v2/enums/storagetype"
)

//go:generate ./generate_enum servertypestatus experimental available deprecated
//go:generate ./generate_enum storagetype local network

// ServerType represents a Server Type
// https://api.gb1.brightbox.com/1.0/#server_type
type ServerType struct {
	ResourceRef
	ID          string
	Name        string
	Status      servertypestatus.Enum
	Cores       uint
	RAM         uint
	Handle      string
	DiskSize    uint             `json:"disk_size"`
	StorageType storagetype.Enum `json:"storage_type"`
}
