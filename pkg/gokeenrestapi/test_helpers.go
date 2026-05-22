package gokeenrestapi

import (
	"os"
	"sync"

	"github.com/noksa/gokeenapi/pkg/config"
)

// SetupTestConfig configures the global API client for testing with the given server URL.
// This is a test helper that encapsulates the necessary global state mutation.
//
// Note: This function modifies global state (config.Cfg and restyClient) which is necessary
// for the current API client architecture. Tests should call this after creating a mock server.
func SetupTestConfig(serverURL string) {
	// Create temporary directory for test cache
	tmpDir, err := os.MkdirTemp("", "gokeenapi-test-*")
	if err != nil {
		panic("failed to create temp dir for tests: " + err.Error())
	}

	config.Cfg = config.GokeenapiConfig{
		Keenetic: config.Keenetic{
			URL:      serverURL,
			Login:    "admin",
			Password: "password",
		},
		DataDir: tmpDir,
	}
	// Reset client and its Once guard so the next call re-initializes with the new config.
	restyClient = nil
	restyClientOnce = sync.Once{}
	cachedCookieMu.Lock()
	cachedCookie = ""
	cachedCookieMu.Unlock()
}

// CleanupTestConfig removes the temporary test cache directory
func CleanupTestConfig() {
	if config.Cfg.DataDir != "" {
		_ = os.RemoveAll(config.Cfg.DataDir)
	}
}
