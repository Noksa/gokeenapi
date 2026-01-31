package gokeenrestapimodels

// RciIpRoute represents a static route configuration from /rci/ip/route endpoint.
type RciIpRoute struct {
	// Network is the destination network address (mutually exclusive with Host)
	Network string `json:"network"`
	// Host is the destination host address (mutually exclusive with Network)
	Host string `json:"host"`
	// Mask is the subnet mask for network routes (e.g., "255.255.255.0")
	Mask string `json:"mask"`
	// Interface is the target interface ID for this route (e.g., "Wireguard0")
	Interface string `json:"interface"`
	// Auto indicates if the route was automatically created
	Auto bool `json:"auto"`
}

// RciShowIpRoute represents a route entry from the routing table (/rci/show/ip/route).
type RciShowIpRoute struct {
	// Destination is the route destination in CIDR notation (e.g., "10.0.0.0/8")
	Destination string `json:"destination,omitempty"`
	// Interface is the outgoing interface for this route
	Interface string `json:"interface,omitempty"`
}
