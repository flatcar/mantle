package brightbox

import (
	"time"
)

// FirewallRule represents a firewall rule.
// https://api.gb1.brightbox.com/1.0/#firewall_rule
type FirewallRule struct {
	ResourceRef
	ID              string
	Source          string          `json:"source"`
	SourcePort      string          `json:"source_port"`
	Destination     string          `json:"destination"`
	DestinationPort string          `json:"destination_port"`
	Protocol        string          `json:"protocol"`
	IcmpTypeName    string          `json:"icmp_type_name"`
	Description     string          `json:"description"`
	CreatedAt       *time.Time      `json:"created_at"`
	FirewallPolicy  *FirewallPolicy `json:"firewall_policy"`
}

// FirewallRuleOptions is used in conjunction with CreateFirewallRule and
// UpdateFirewallRule to create and update firewall rules.
type FirewallRuleOptions struct {
	ID              string  `json:"-"`
	FirewallPolicy  string  `json:"firewall_policy,omitempty"`
	Protocol        *string `json:"protocol,omitempty"`
	Source          *string `json:"source,omitempty"`
	SourcePort      *string `json:"source_port,omitempty"`
	Destination     *string `json:"destination,omitempty"`
	DestinationPort *string `json:"destination_port,omitempty"`
	IcmpTypeName    *string `json:"icmp_type_name,omitempty"`
	Description     *string `json:"description,omitempty"`
}
