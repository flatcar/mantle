package brightbox

import (
	"context"
	"path"
	"time"
)

// ServerGroup represents a server group
// https://api.gb1.brightbox.com/1.0/#server_group
type ServerGroup struct {
	ResourceRef
	ID             string
	Name           string
	Description    string
	Default        bool
	Fqdn           string
	CreatedAt      *time.Time `json:"created_at"`
	Account        *Account
	FirewallPolicy *FirewallPolicy `json:"firewall_policy"`
	Servers        []Server
}

// ServerGroupOptions is used in combination with CreateServerGroup and
// UpdateServerGroup to create and update server groups
type ServerGroupOptions struct {
	ID          string  `json:"-"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ServerGroupMember is used to add, remove and move a server between server groups
type ServerGroupMember struct {
	Server string `json:"server"`
}

// ServerGroupMemberList is used to add, remove and move servers between server groups
type ServerGroupMemberList struct {
	Servers []ServerGroupMember `json:"servers"`
}

// AddServersToServerGroup adds servers to an existing server group
func (c *Client) AddServersToServerGroup(ctx context.Context, identifier string, attachment ServerGroupMemberList) (*ServerGroup, error) {
	return apiPost[ServerGroup](
		ctx,
		c,
		path.Join(servergroupAPIPath, identifier, "add_servers"),
		attachment,
	)
}

// RemoveServersFromServerGroup remove servers from an existing server group
func (c *Client) RemoveServersFromServerGroup(ctx context.Context, identifier string, attachment ServerGroupMemberList) (*ServerGroup, error) {
	return apiPost[ServerGroup](
		ctx,
		c,
		path.Join(servergroupAPIPath, identifier, "remove_servers"),
		attachment,
	)
}

// MoveServersToServerGroup moves servers between two existing server groups
func (c *Client) MoveServersToServerGroup(ctx context.Context, from string, to string, servers ServerGroupMemberList) (*ServerGroup, error) {
	opts := struct {
		ServerGroupMemberList
		Destination string `json:"destination"`
	}{servers, to}
	return apiPost[ServerGroup](
		ctx,
		c,
		path.Join(servergroupAPIPath, from, "move_servers"),
		opts,
	)
}
