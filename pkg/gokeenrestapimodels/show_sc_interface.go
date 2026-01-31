package gokeenrestapimodels

// Address wraps an IP address string.
type Address struct {
	// Address is the IP address value
	Address string `json:"address"`
}

// IP contains IP address configuration for an interface.
type IP struct {
	// Address contains the interface IP address
	Address Address `json:"address"`
}

// Asc contains AmneziaWG (AWG) specific parameters for WireGuard interfaces.
// These parameters provide additional obfuscation for WireGuard traffic.
type Asc struct {
	// Jc is the junk packet count parameter
	Jc string `json:"jc"`
	// Jmin is the minimum junk packet size
	Jmin string `json:"jmin"`
	// Jmax is the maximum junk packet size
	Jmax string `json:"jmax"`
	// S1 is the first obfuscation seed
	S1 string `json:"s1"`
	// S2 is the second obfuscation seed
	S2 string `json:"s2"`
	// H1 is the first header modification parameter
	H1 string `json:"h1"`
	// H2 is the second header modification parameter
	H2 string `json:"h2"`
	// H3 is the third header modification parameter
	H3 string `json:"h3"`
	// H4 is the fourth header modification parameter
	H4 string `json:"h4"`
}

// Endpoint contains the WireGuard peer endpoint address.
type Endpoint struct {
	// Address is the peer endpoint in "host:port" format
	Address string `json:"address"`
}

// KeepaliveInterval contains the WireGuard persistent keepalive setting.
type KeepaliveInterval struct {
	// Interval is the keepalive interval in seconds
	Interval int `json:"interval"`
}

// AllowIps represents an allowed IP range for a WireGuard peer.
type AllowIps struct {
	// Address is the network address
	Address string `json:"address"`
	// Mask is the subnet mask
	Mask string `json:"mask"`
}

// Peer represents a WireGuard peer configuration.
type Peer struct {
	// Key is the peer's public key
	Key string `json:"key"`
	// Comment is an optional description for the peer
	Comment string `json:"comment,omitempty"`
	// Endpoint is the peer's endpoint address
	Endpoint Endpoint `json:"endpoint"`
	// KeepaliveInterval is the persistent keepalive setting
	KeepaliveInterval KeepaliveInterval `json:"keepalive-interval"`
	// PresharedKey is the optional pre-shared key for additional security
	PresharedKey string `json:"preshared-key"`
	// AllowIps lists the allowed IP ranges for this peer
	AllowIps []AllowIps `json:"allow-ips"`
}

// Wireguard contains WireGuard-specific interface configuration.
type Wireguard struct {
	// Asc contains AmneziaWG obfuscation parameters
	Asc Asc `json:"asc"`
	// Peer contains the list of WireGuard peers
	Peer []Peer `json:"peer"`
}

// RciShowScInterface represents system configuration for an interface from /rci/show/sc/interface endpoint.
type RciShowScInterface struct {
	// Description is the interface description
	Description string `json:"description,omitempty"`
	// IP contains IP address configuration
	IP IP `json:"ip"`
	// Wireguard contains WireGuard-specific configuration (if applicable)
	Wireguard Wireguard `json:"wireguard"`
}
