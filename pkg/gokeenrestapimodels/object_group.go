package gokeenrestapimodels

// ObjectGroupFqdnResponse represents the response from /rci/object-group/fqdn endpoint.
// Maps group names to their FQDN configurations for DNS-routing.
type ObjectGroupFqdnResponse map[string]ObjectGroupFqdn

// ObjectGroupFqdn represents a single FQDN object-group used for DNS-routing.
// Contains a list of domain names that should be routed through a specific interface.
type ObjectGroupFqdn struct {
	// Include contains the list of domain entries in this group
	Include []ObjectGroupFqdnEntry `json:"include"`
}

// ObjectGroupFqdnEntry represents a single domain entry in an FQDN object-group.
type ObjectGroupFqdnEntry struct {
	// Address is the domain name (e.g., "example.com", "*.google.com")
	Address string `json:"address"`
}

// DnsProxyRouteResponse represents the response from /rci/dns-proxy/route endpoint.
// Contains the list of DNS-proxy routing rules.
type DnsProxyRouteResponse []DnsProxyRoute

// DnsProxyRoute represents a single DNS-proxy routing rule.
// Links an FQDN object-group to a target interface for policy-based routing.
type DnsProxyRoute struct {
	// Group is the name of the FQDN object-group
	Group string `json:"group"`
	// Interface is the target interface ID for routing (e.g., "Wireguard0")
	Interface string `json:"interface"`
}
