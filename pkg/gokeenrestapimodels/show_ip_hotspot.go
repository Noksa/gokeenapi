package gokeenrestapimodels

// RciShowIpHotspot represents the hotspot (known devices) list from /rci/show/ip/hotspot endpoint.
type RciShowIpHotspot struct {
	// Host contains the list of known devices
	Host []Host `json:"host,omitempty"`
	// Prompt is the CLI prompt string
	Prompt string `json:"prompt,omitempty"`
}

// Host represents a single device in the router's known hosts database.
type Host struct {
	// Mac is the device's MAC address
	Mac string `json:"mac,omitempty"`
	// Via indicates how the device is connected (e.g., interface name)
	Via string `json:"via,omitempty"`
	// IP is the device's IP address
	IP string `json:"ip,omitempty"`
	// Hostname is the device's hostname from DHCP or mDNS
	Hostname string `json:"hostname,omitempty"`
	// Name is the user-assigned device name
	Name string `json:"name,omitempty"`
	// Registered indicates if the device is registered in the router
	Registered bool `json:"registered,omitempty"`
	// Link indicates the connection status
	Link string `json:"link,omitempty"`
}
