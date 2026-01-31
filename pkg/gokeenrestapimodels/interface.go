package gokeenrestapimodels

// RciShowInterface represents network interface information from /rci/show/interface endpoint.
type RciShowInterface struct {
	// Id is the unique interface identifier (e.g., "Wireguard0", "ISP")
	Id string `json:"id"`
	// Type indicates the interface type (e.g., "Wireguard", "PPPoE", "Bridge")
	Type string `json:"type"`
	// Description is the user-friendly interface name
	Description string `json:"description"`
	// Address is the IP address assigned to the interface
	Address string `json:"address"`
	// Connected indicates connection status (e.g., "yes", "no")
	Connected string `json:"connected"`
	// Link indicates physical link status (e.g., "up", "down")
	Link string `json:"link"`
	// State indicates administrative state (e.g., "up", "down")
	State string `json:"state"`
	// DefaultGw indicates if this interface is the default gateway
	DefaultGw bool `json:"defaultgw,omitempty"`
}

// Import represents an interface import configuration.
type Import struct {
	// Import is the import type identifier
	Import string `json:"import"`
	// Name is the import name
	Name string `json:"name"`
	// Filename is the associated configuration file
	Filename string `json:"filename"`
}

// CreatedInterface represents the response when a new interface is created.
type CreatedInterface struct {
	// Created contains the ID of the newly created interface
	Created string `json:"created"`
	// Status contains execution status messages
	Status []struct {
		Status  string `json:"status"`
		Code    string `json:"code"`
		Ident   string `json:"ident"`
		Message string `json:"message"`
	} `json:"status"`
}
