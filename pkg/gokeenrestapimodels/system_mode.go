package gokeenrestapimodels

// SystemMode represents the router's operating mode from /rci/show/system/mode endpoint.
// Keenetic routers can operate in "router" mode or "extender" mode.
type SystemMode struct {
	// Active is the currently active operating mode (e.g., "router", "extender")
	Active string `json:"active,omitempty"`
	// Selected is the user-selected operating mode
	Selected string `json:"selected,omitempty"`
}
