package brightbox

// Interface represent a server's network interface(s)
// https://api.gb1.brightbox.com/1.0/#interface
type Interface struct {
	ResourceRef
	ID          string
	MacAddress  string `json:"mac_address"`
	IPv4Address string `json:"ipv4_address"`
	IPv6Address string `json:"ipv6_address"`
	Server      *Server
}
