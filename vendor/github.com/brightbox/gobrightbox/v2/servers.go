package brightbox

import (
	"time"

	"github.com/brightbox/gobrightbox/v2/enums/serverstatus"
)

//go:generate ./generate_enum serverstatus creating active inactive deleting deleted failed unavailable
//go:generate ./generate_server_commands

// Server represents a Cloud Server
// https://api.gb1.brightbox.com/1.0/#server
// DeletedAt is nil if the server has not yet been deleted
type Server struct {
	ResourceRef
	ServerConsole
	ID                      string
	Name                    string
	Status                  serverstatus.Enum `json:"status"`
	Hostname                string
	Fqdn                    string
	UserData                string     `json:"user_data"`
	CreatedAt               *time.Time `json:"created_at"`
	DeletedAt               *time.Time `json:"deleted_at"`
	StartedAt               *time.Time `json:"started_at"`
	SnapshotsSchedule       string     `json:"snapshots_schedule"`
	SnapshotsScheduleNextAt *time.Time `json:"snapshots_schedule_next_at"`
	SnapshotsRetention      string     `json:"snapshots_retention"`
	Locked                  bool       `json:"locked"`
	CompatibilityMode       bool       `json:"compatibility_mode"`
	DiskEncrypted           bool       `json:"disk_encrypted"`
	Account                 *Account
	Image                   *Image
	Zone                    *Zone
	ServerType              *ServerType   `json:"server_type"`
	CloudIPs                []CloudIP     `json:"cloud_ips"`
	ServerGroups            []ServerGroup `json:"server_groups"`
	Snapshots               []Image
	Interfaces              []Interface
	Volumes                 []Volume
}

// ServerConsole is embedded into Server and contains the fields used in response
// to an ActivateConsoleForServer request.
type ServerConsole struct {
	ConsoleToken        *string    `json:"console_token"`
	ConsoleURL          *string    `json:"console_url"`
	ConsoleTokenExpires *time.Time `json:"console_token_expires"`
}

// ServerOptions is used in conjunction with CreateServer and UpdateServer to
// create and update servers.
type ServerOptions struct {
	ID                 string        `json:"-"`
	Image              *string       `json:"image,omitempty"`
	Name               *string       `json:"name,omitempty"`
	ServerType         *string       `json:"server_type,omitempty"`
	Zone               *string       `json:"zone,omitempty"`
	UserData           *string       `json:"user_data,omitempty"`
	SnapshotsRetention *string       `json:"snapshots_retention,omitempty"`
	SnapshotsSchedule  *string       `json:"snapshots_schedule,omitempty"`
	ServerGroups       []string      `json:"server_groups,omitempty"`
	CompatibilityMode  *bool         `json:"compatibility_mode,omitempty"`
	DiskEncrypted      *bool         `json:"disk_encrypted,omitempty"`
	CloudIP            *bool         `json:"cloud_ip,omitempty"`
	Volumes            []VolumeEntry `json:"volumes,omitempty"`
}

// ServerNewSize is used in conjunction with ResizeServer
// to specify the new Server type for the Server
type ServerNewSize struct {
	NewType string `json:"new_type"`
}

// VolumeEntry is used within ServerOptions to specify the boot
// volume for a server on creation. Either volume or image/disk size can
// be given
type VolumeEntry struct {
	Volume string `json:"volume,omitempty"`
	Size   uint   `json:"size,omitempty"`
	Image  string `json:"image,omitempty"`
}
