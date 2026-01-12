package gokeenrestapimodels

// ObjectGroupFqdnResponse represents the response from /rci/object-group/fqdn
// Maps group names to their configurations
type ObjectGroupFqdnResponse map[string]ObjectGroupFqdn

// ObjectGroupFqdn represents a single FQDN object-group
type ObjectGroupFqdn struct {
	Include []ObjectGroupFqdnEntry `json:"include"`
}

// ObjectGroupFqdnEntry represents a single entry in an object-group
type ObjectGroupFqdnEntry struct {
	Address string `json:"address"`
}

// DnsProxyRouteResponse represents the response from /rci/dns-proxy/route
type DnsProxyRouteResponse []DnsProxyRoute

// DnsProxyRoute represents a single dns-proxy route
type DnsProxyRoute struct {
	Group     string `json:"group"`
	Interface string `json:"interface"`
}
