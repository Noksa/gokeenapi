package gokeenrestapi

import "github.com/noksa/gokeenapi/pkg/config"

// SetupTestConfig configures the global API client for testing with the given server URL.
// This is a test helper that encapsulates the necessary global state mutation.
//
// Note: This function modifies global state (config.Cfg and restyClient) which is necessary
// for the current API client architecture. Tests should call this after creating a mock server.
func SetupTestConfig(serverURL string) {
	config.Cfg = config.GokeenapiConfig{
		Keenetic: config.Keenetic{
			URL:      serverURL,
			Login:    "admin",
			Password: "password",
		},
	}
	// Reset client to use new config
	restyClient = nil
}
