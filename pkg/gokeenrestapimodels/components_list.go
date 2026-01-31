package gokeenrestapimodels

// RciComponentsList represents the response from /rci/components/list endpoint.
// Contains information about installed and available firmware components.
type RciComponentsList struct {
	// Sandbox is the firmware sandbox identifier
	Sandbox string `json:"sandbox"`
	// Component maps component names to their details
	Component map[string]Component `json:"component"`
}

// Component represents a single firmware component available on the Keenetic router.
type Component struct {
	// Group is the component category (e.g., "base", "network")
	Group string `json:"group,omitempty"`
	// Installed indicates if the component is currently installed
	Installed string `json:"installed,omitempty"`
	// Libndm is the libndm version requirement
	Libndm string `json:"libndm,omitempty"`
	// Version is the component version
	Version string `json:"version,omitempty"`
}
