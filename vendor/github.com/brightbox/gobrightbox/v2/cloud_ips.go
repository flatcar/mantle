package brightbox

import (
	"context"
	"path"

	"github.com/brightbox/gobrightbox/v2/enums/cloudipstatus"
	"github.com/brightbox/gobrightbox/v2/enums/mode"
	"github.com/brightbox/gobrightbox/v2/enums/transportprotocol"
)

//go:generate ./generate_enum cloudipstatus mapped unmapped
//go:generate ./generate_enum mode nat route
//go:generate ./generate_enum transportprotocol tcp udp

// CloudIP represents a Cloud IP
// https://api.gb1.brightbox.com/1.0/#cloud_ip
type CloudIP struct {
	ResourceRef
	ID              string
	Name            string
	PublicIP        string             `json:"public_ip"`
	PublicIPv4      string             `json:"public_ipv4"`
	PublicIPv6      string             `json:"public_ipv6"`
	Status          cloudipstatus.Enum `json:"status"`
	ReverseDNS      string             `json:"reverse_dns"`
	Fqdn            string
	Mode            mode.Enum
	Account         *Account
	Interface       *Interface
	Server          *Server
	ServerGroup     *ServerGroup     `json:"server_group"`
	PortTranslators []PortTranslator `json:"port_translators"`
	LoadBalancer    *LoadBalancer    `json:"load_balancer"`
	DatabaseServer  *DatabaseServer  `json:"database_server"`
}

// PortTranslator represents a port translator on a Cloud IP
type PortTranslator struct {
	Incoming uint16                 `json:"incoming"`
	Outgoing uint16                 `json:"outgoing"`
	Protocol transportprotocol.Enum `json:"protocol,omitempty"`
}

// CloudIPOptions is used in conjunction with CreateCloudIP and UpdateCloudIP to
// create and update cloud IPs.
type CloudIPOptions struct {
	ID              string           `json:"-"`
	ReverseDNS      *string          `json:"reverse_dns,omitempty"`
	Mode            mode.Enum        `json:"mode,omitempty"`
	Name            *string          `json:"name,omitempty"`
	PortTranslators []PortTranslator `json:"port_translators,omitempty"`
}

// CloudIPAttachment is used in conjunction with MapCloudIP to specify
// the destination the CloudIP should be mapped to
type CloudIPAttachment struct {
	Destination string `json:"destination"`
}

// MapCloudIP issues a request to map the cloud ip to the destination. The
// destination can be an identifier of any resource capable of receiving a Cloud
// IP, such as a server interface, a load balancer, or a cloud sql instace.
func (c *Client) MapCloudIP(ctx context.Context, identifier string, attachment CloudIPAttachment) (*CloudIP, error) {
	return apiPost[CloudIP](
		ctx,
		c,
		path.Join(cloudipAPIPath, identifier, "map"),
		attachment,
	)
}

// UnMapCloudIP issues a request to unmap the cloud ip.
func (c *Client) UnMapCloudIP(ctx context.Context, identifier string) (*CloudIP, error) {
	return apiPost[CloudIP](
		ctx,
		c,
		path.Join(cloudipAPIPath, identifier, "unmap"),
		nil,
	)
}
