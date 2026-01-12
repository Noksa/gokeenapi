package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetURLCacheTTL_Default(t *testing.T) {
	// Reset config
	Cfg = GokeenapiConfig{}

	ttl := GetURLCacheTTL()
	assert.Equal(t, time.Minute, ttl, "Default TTL should be 1 minute")
}

func TestGetURLCacheTTL_Configured(t *testing.T) {
	// Reset config
	Cfg = GokeenapiConfig{
		Cache: Cache{
			URLTTL: 5 * time.Minute,
		},
	}

	ttl := GetURLCacheTTL()
	assert.Equal(t, 5*time.Minute, ttl, "Should return configured TTL")
}

func TestGetURLCacheTTL_Zero(t *testing.T) {
	// Reset config
	Cfg = GokeenapiConfig{
		Cache: Cache{
			URLTTL: 0,
		},
	}

	ttl := GetURLCacheTTL()
	assert.Equal(t, time.Minute, ttl, "Zero TTL should default to 1 minute")
}

func TestGetURLCacheTTL_Negative(t *testing.T) {
	// Reset config
	Cfg = GokeenapiConfig{
		Cache: Cache{
			URLTTL: -5 * time.Minute,
		},
	}

	ttl := GetURLCacheTTL()
	assert.Equal(t, time.Minute, ttl, "Negative TTL should default to 1 minute")
}

func TestLoadConfig_WithCacheTTL(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
keenetic:
  url: http://192.168.1.1
  login: admin
  password: pass

cache:
  urlTtl: 10m
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config
	err = LoadConfig(configPath)
	require.NoError(t, err)

	// Verify cache TTL is loaded correctly
	assert.Equal(t, 10*time.Minute, Cfg.Cache.URLTTL)
	assert.Equal(t, 10*time.Minute, GetURLCacheTTL())
}

func TestLoadConfig_WithoutCacheTTL(t *testing.T) {
	// Reset config
	Cfg = GokeenapiConfig{}

	// Create temporary config file without cache section
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
keenetic:
  url: http://192.168.1.1
  login: admin
  password: pass
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config
	err = LoadConfig(configPath)
	require.NoError(t, err)

	// Verify default TTL is used
	assert.Equal(t, time.Duration(0), Cfg.Cache.URLTTL)
	assert.Equal(t, time.Minute, GetURLCacheTTL())
}
