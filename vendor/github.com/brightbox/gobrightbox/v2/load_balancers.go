package brightbox

import (
	"context"
	"path"
	"time"

	"github.com/brightbox/gobrightbox/v2/enums/balancingpolicy"
	"github.com/brightbox/gobrightbox/v2/enums/healthchecktype"
	"github.com/brightbox/gobrightbox/v2/enums/listenerprotocol"
	"github.com/brightbox/gobrightbox/v2/enums/loadbalancerstatus"
	"github.com/brightbox/gobrightbox/v2/enums/proxyprotocol"
)

//go:generate ./generate_enum loadbalancerstatus creating active deleting deleted failing failed
//go:generate ./generate_enum proxyprotocol v1 v2 v2-ssl v2-ssl-cn
//go:generate ./generate_enum balancingpolicy least-connections round-robin source-address
//go:generate ./generate_enum healthchecktype tcp http
//go:generate ./generate_enum listenerprotocol tcp http https

// LoadBalancer represents a Load Balancer
// https://api.gb1.brightbox.com/1.0/#load_balancer
type LoadBalancer struct {
	ResourceRef
	ID                string
	Name              string
	Status            loadbalancerstatus.Enum
	Locked            bool
	HTTPSRedirect     bool   `json:"https_redirect"`
	SslMinimumVersion string `json:"ssl_minimum_version"`
	BufferSize        uint   `json:"buffer_size"`
	Policy            balancingpolicy.Enum
	Listeners         []LoadBalancerListener
	Healthcheck       LoadBalancerHealthcheck
	Certificate       *LoadBalancerAcmeCertificate
	Acme              *LoadBalancerAcme
	CreatedAt         *time.Time `json:"created_at"`
	DeletedAt         *time.Time `json:"deleted_at"`
	Account           *Account
	Nodes             []Server
	CloudIPs          []CloudIP `json:"cloud_ips"`
}

// LoadBalancerAcme represents an ACME object on a LoadBalancer
type LoadBalancerAcme struct {
	Certificate *LoadBalancerAcmeCertificate `json:"certificate"`
	Domains     []LoadBalancerAcmeDomain     `json:"domains"`
}

// LoadBalancerAcmeCertificate represents an ACME issued certificate on
// a LoadBalancer
type LoadBalancerAcmeCertificate struct {
	Fingerprint string    `json:"fingerprint"`
	ExpiresAt   time.Time `json:"expires_at"`
	IssuedAt    time.Time `json:"issued_at"`
}

// LoadBalancerAcmeDomain represents a domain for which ACME support
// has been requested
type LoadBalancerAcmeDomain struct {
	Identifier  string `json:"identifier"`
	Status      string `json:"status"`
	LastMessage string `json:"last_message"`
}

// LoadBalancerHealthcheck represents a health check on a LoadBalancer
type LoadBalancerHealthcheck struct {
	Type          healthchecktype.Enum `json:"type"`
	Port          uint16               `json:"port"`
	Request       string               `json:"request,omitempty"`
	Interval      uint                 `json:"interval,omitempty"`
	Timeout       uint                 `json:"timeout,omitempty"`
	ThresholdUp   uint                 `json:"threshold_up,omitempty"`
	ThresholdDown uint                 `json:"threshold_down,omitempty"`
}

// LoadBalancerListener represents a listener on a LoadBalancer
type LoadBalancerListener struct {
	Protocol      listenerprotocol.Enum `json:"protocol,omitempty"`
	In            uint16                `json:"in,omitempty"`
	Out           uint16                `json:"out,omitempty"`
	Timeout       uint                  `json:"timeout,omitempty"`
	ProxyProtocol proxyprotocol.Enum    `json:"proxy_protocol,omitempty"`
}

// LoadBalancerOptions is used in conjunction with CreateLoadBalancer and
// UpdateLoadBalancer to create and update load balancers
type LoadBalancerOptions struct {
	ID                    string                   `json:"-"`
	Name                  *string                  `json:"name,omitempty"`
	Nodes                 []LoadBalancerNode       `json:"nodes,omitempty"`
	Policy                balancingpolicy.Enum     `json:"policy,omitempty"`
	Listeners             []LoadBalancerListener   `json:"listeners,omitempty"`
	Healthcheck           *LoadBalancerHealthcheck `json:"healthcheck,omitempty"`
	Domains               *[]string                `json:"domains,omitempty"`
	CertificatePem        *string                  `json:"certificate_pem,omitempty"`
	CertificatePrivateKey *string                  `json:"certificate_private_key,omitempty"`
	SslMinimumVersion     *string                  `json:"ssl_minimum_version,omitempty"`
	HTTPSRedirect         *bool                    `json:"https_redirect,omitempty"`
}

// LoadBalancerNode is used in conjunction with LoadBalancerOptions,
// AddNodesToLoadBalancer, RemoveNodesFromLoadBalancer to specify a list of
// servers to use as load balancer nodes. The Node parameter should be a server
// identifier.
type LoadBalancerNode struct {
	Node string `json:"node,omitempty"`
}

// AddNodesToLoadBalancer adds nodes to an existing load balancer.
func (c *Client) AddNodesToLoadBalancer(ctx context.Context, identifier string, nodes []LoadBalancerNode) (*LoadBalancer, error) {
	return apiPost[LoadBalancer](
		ctx,
		c,
		path.Join(loadbalancerAPIPath, identifier, "add_nodes"),
		nodes,
	)

}

// RemoveNodesFromLoadBalancer removes nodes from an existing load balancer.
func (c *Client) RemoveNodesFromLoadBalancer(ctx context.Context, identifier string, nodes []LoadBalancerNode) (*LoadBalancer, error) {
	return apiPost[LoadBalancer](
		ctx,
		c,
		path.Join(loadbalancerAPIPath, identifier, "remove_nodes"),
		nodes,
	)
}

// AddListenersToLoadBalancer adds listeners to an existing load balancer.
func (c *Client) AddListenersToLoadBalancer(ctx context.Context, identifier string, listeners []LoadBalancerListener) (*LoadBalancer, error) {
	return apiPost[LoadBalancer](
		ctx,
		c,
		path.Join(loadbalancerAPIPath, identifier, "add_listeners"),
		listeners,
	)

}

// RemoveListenersFromLoadBalancer removes listeners from an existing load balancer.
func (c *Client) RemoveListenersFromLoadBalancer(ctx context.Context, identifier string, listeners []LoadBalancerListener) (*LoadBalancer, error) {
	return apiPost[LoadBalancer](
		ctx,
		c,
		path.Join(loadbalancerAPIPath, identifier, "remove_listeners"),
		listeners,
	)
}
