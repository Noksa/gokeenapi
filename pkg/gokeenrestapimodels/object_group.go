package gokeenrestapimodels

// ObjectGroupFqdnResponse represents the response from /rci/show/object-group/fqdn
type ObjectGroupFqdnResponse struct {
	Group []ObjectGroupFqdn `json:"group"`
}

// ObjectGroupFqdn represents a single FQDN object-group
type ObjectGroupFqdn struct {
	GroupName string                 `json:"group-name"`
	Entry     []ObjectGroupFqdnEntry `json:"entry"`
}

// ObjectGroupFqdnEntry represents a single entry in an object-group
type ObjectGroupFqdnEntry struct {
	Fqdn string `json:"fqdn"`
}

// DnsProxyRouteResponse represents the response from /rci/dns-proxy/route
type DnsProxyRouteResponse []DnsProxyRoute

// DnsProxyRoute represents a single dns-proxy route
type DnsProxyRoute struct {
	Group     string `json:"group"`
	Interface string `json:"interface"`
}
