package brightbox

import (
	"context"
	"path"
	"time"
)

// FirewallPolicy represents a firewall policy.
// https://api.gb1.brightbox.com/1.0/#firewall_policy
type FirewallPolicy struct {
	ResourceRef
	ID          string
	Name        string
	Default     bool
	Description string
	CreatedAt   *time.Time `json:"created_at"`
	Account     *Account
	ServerGroup *ServerGroup   `json:"server_group"`
	Rules       []FirewallRule `json:"rules"`
}

// FirewallPolicyOptions is used in conjunction with CreateFirewallPolicy and
// UpdateFirewallPolicy to create and update firewall policies.
type FirewallPolicyOptions struct {
	ID          string  `json:"-"`
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	*FirewallPolicyAttachment
}

// FirewallPolicyAttachment is used in conjunction with FirewallPolicyOptions,
// ApplyFirewallPolicy and RemoveFirewallPolicy to specify the group that
// the firewall policy should apply to. The ServerGroup parameter should
// be a server group identifier.
type FirewallPolicyAttachment struct {
	ServerGroup string `json:"server_group"`
}

// ApplyFirewallPolicy issues a request to apply the given firewall policy to
// the given server group.
func (c *Client) ApplyFirewallPolicy(ctx context.Context, identifier string, attachment FirewallPolicyAttachment) (*FirewallPolicy, error) {
	return apiPost[FirewallPolicy](
		ctx,
		c,
		path.Join(firewallpolicyAPIPath, identifier, "apply_to"),
		attachment,
	)

}

// RemoveFirewallPolicy issues a request to remove the given firewall policy from
// the given server group.
func (c *Client) RemoveFirewallPolicy(ctx context.Context, identifier string, serverGroup FirewallPolicyAttachment) (*FirewallPolicy, error) {
	return apiPost[FirewallPolicy](
		ctx,
		c,
		path.Join(firewallpolicyAPIPath, identifier, "remove"),
		serverGroup,
	)
}
