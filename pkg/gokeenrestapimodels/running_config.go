package gokeenrestapimodels

// RunningConfig represents the router's current running configuration from /rci/show/running-config.
type RunningConfig struct {
	// Message contains the configuration as a list of CLI commands
	Message []string `json:"message"`
}
