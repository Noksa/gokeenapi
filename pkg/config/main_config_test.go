package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_ValidFile(t *testing.T) {
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file: ["routes.bat"]
dns:
  records:
    - domain: "test.local"
      ip: ["192.168.1.100"]
logs:
  debug: true`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	assert.Equal(t, "http://192.168.1.1", Cfg.Keenetic.URL)
	assert.Equal(t, "admin", Cfg.Keenetic.Login)
	assert.Equal(t, "password", Cfg.Keenetic.Password)
	assert.Len(t, Cfg.Routes, 1)
	assert.Equal(t, "Wireguard0", Cfg.Routes[0].InterfaceID)
	assert.Len(t, Cfg.DNS.Records, 1)
	assert.Equal(t, "test.local", Cfg.DNS.Records[0].Domain)
	assert.True(t, Cfg.Logs.Debug)
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	err := LoadConfig("/nonexistent/config.yaml")
	assert.Error(t, err)
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	invalidContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file: ["routes.bat"
dns:`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.Error(t, err)
}

func TestLoadConfig_EmptyPath(t *testing.T) {
	// Clear environment variable
	_ = os.Unsetenv("GOKEENAPI_CONFIG")

	err := LoadConfig("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config path is empty")
}

func TestLoadConfig_FromEnvironment(t *testing.T) {
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "env_config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment variable
	_ = os.Setenv("GOKEENAPI_CONFIG", configPath)
	defer func() { _ = os.Unsetenv("GOKEENAPI_CONFIG") }()

	err = LoadConfig("")
	assert.NoError(t, err)
	assert.Equal(t, "http://192.168.1.1", Cfg.Keenetic.URL)
}

func TestLoadConfig_EnvironmentOverrides(t *testing.T) {
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set environment overrides
	_ = os.Setenv("GOKEENAPI_KEENETIC_LOGIN", "env_admin")
	_ = os.Setenv("GOKEENAPI_KEENETIC_PASSWORD", "env_password")
	defer func() {
		_ = os.Unsetenv("GOKEENAPI_KEENETIC_LOGIN")
		_ = os.Unsetenv("GOKEENAPI_KEENETIC_PASSWORD")
	}()

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	assert.Equal(t, "http://192.168.1.1", Cfg.Keenetic.URL)
	assert.Equal(t, "env_admin", Cfg.Keenetic.Login)
	assert.Equal(t, "env_password", Cfg.Keenetic.Password)
}

func TestLoadConfig_DockerEnvironment(t *testing.T) {
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Set Docker environment
	_ = os.Setenv("GOKEENAPI_INSIDE_DOCKER", "true")
	defer func() { _ = os.Unsetenv("GOKEENAPI_INSIDE_DOCKER") }()

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	assert.Equal(t, "/etc/gokeenapi", Cfg.DataDir)
}

func TestLoadConfig_BatFileListExpansion(t *testing.T) {
	// Create a bat-file list YAML
	batListContent := `bat-file:
  - /path/to/discord.bat
  - /path/to/youtube.bat
  - /path/to/instagram.bat`

	tmpDir := t.TempDir()
	batListPath := filepath.Join(tmpDir, "all-of-them.yaml")
	err := os.WriteFile(batListPath, []byte(batListContent), 0644)
	require.NoError(t, err)

	// Create main config that references the bat-file list
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "all-of-them.yaml"
      - "/path/to/extra.bat"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Verify that the YAML file was expanded
	assert.Len(t, Cfg.Routes, 1)
	assert.Equal(t, "Wireguard0", Cfg.Routes[0].InterfaceID)
	// Should have 3 files from YAML + 1 extra .bat file = 4 total
	assert.Len(t, Cfg.Routes[0].BatFile, 4)
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/to/discord.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/to/youtube.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/to/instagram.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/to/extra.bat")
}

func TestLoadConfig_BatFileListWithAbsolutePath(t *testing.T) {
	// Create a bat-file list YAML
	batListContent := `bat-file:
  - /absolute/path/to/file1.bat
  - /absolute/path/to/file2.bat`

	tmpDir := t.TempDir()
	batListPath := filepath.Join(tmpDir, "subdir", "batlist.yaml")
	err := os.MkdirAll(filepath.Dir(batListPath), 0755)
	require.NoError(t, err)
	err = os.WriteFile(batListPath, []byte(batListContent), 0644)
	require.NoError(t, err)

	// Create main config with absolute path to bat-file list
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "` + batListPath + `"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	assert.Len(t, Cfg.Routes[0].BatFile, 2)
	assert.Contains(t, Cfg.Routes[0].BatFile, "/absolute/path/to/file1.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/absolute/path/to/file2.bat")
}

func TestLoadConfig_BatFileListNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main config that references non-existent bat-file list
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "nonexistent.yaml"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read bat-list")
}

func TestLoadConfig_MixedBatFilesAndYAML(t *testing.T) {
	// Create a bat-file list YAML
	batListContent := `bat-file:
  - /path/from/yaml1.bat
  - /path/from/yaml2.bat`

	tmpDir := t.TempDir()
	batListPath := filepath.Join(tmpDir, "list.yaml")
	err := os.WriteFile(batListPath, []byte(batListContent), 0644)
	require.NoError(t, err)

	// Create main config with mix of .bat and .yaml files
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "/direct/path/file1.bat"
      - "list.yaml"
      - "/direct/path/file2.bat"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Should have 2 direct .bat files + 2 from YAML = 4 total
	assert.Len(t, Cfg.Routes[0].BatFile, 4)
	assert.Equal(t, "/direct/path/file1.bat", Cfg.Routes[0].BatFile[0])
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/from/yaml1.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/from/yaml2.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/direct/path/file2.bat")
}

func TestLoadConfig_BatFileListWithYmlExtension(t *testing.T) {
	// Test that .yml extension works (not just .yaml)
	batListContent := `bat-file:
  - /path/to/file1.bat
  - /path/to/file2.bat`

	tmpDir := t.TempDir()
	batListPath := filepath.Join(tmpDir, "list.yml")
	err := os.WriteFile(batListPath, []byte(batListContent), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "list.yml"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	assert.Len(t, Cfg.Routes[0].BatFile, 2)
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/to/file1.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/to/file2.bat")
}

func TestLoadConfig_BatFileListEmpty(t *testing.T) {
	// Test empty bat-file list in YAML
	batListContent := `bat-file: []`

	tmpDir := t.TempDir()
	batListPath := filepath.Join(tmpDir, "empty.yaml")
	err := os.WriteFile(batListPath, []byte(batListContent), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "empty.yaml"
      - "/some/file.bat"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Should only have the direct .bat file
	assert.Len(t, Cfg.Routes[0].BatFile, 1)
	assert.Equal(t, "/some/file.bat", Cfg.Routes[0].BatFile[0])
}

func TestLoadConfig_BatFileListInvalidStructure(t *testing.T) {
	// Test YAML with wrong structure (missing bat-file key)
	batListContent := `wrong-key:
  - /some/file.bat`

	tmpDir := t.TempDir()
	batListPath := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(batListPath, []byte(batListContent), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "invalid.yaml"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	// Should succeed but with empty list (YAML unmarshals to empty BatFileList)
	assert.NoError(t, err)
	assert.Len(t, Cfg.Routes[0].BatFile, 0)
}

func TestLoadConfig_BatFileListNestedYAML(t *testing.T) {
	// Test that nested YAML files are NOT recursively expanded
	// (only one level of expansion is supported)
	nestedYAML := `bat-file:
  - /nested/file1.bat
  - /nested/file2.bat`

	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested.yaml")
	err := os.WriteFile(nestedPath, []byte(nestedYAML), 0644)
	require.NoError(t, err)

	// Main YAML references nested YAML
	mainYAML := `bat-file:
  - /main/file1.bat
  - nested.yaml
  - /main/file2.bat`

	mainPath := filepath.Join(tmpDir, "main.yaml")
	err = os.WriteFile(mainPath, []byte(mainYAML), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "main.yaml"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Should have 3 items: /main/file1.bat, nested.yaml (resolved to absolute path), /main/file2.bat
	// Nested YAML is NOT recursively expanded, but relative paths are resolved
	assert.Len(t, Cfg.Routes[0].BatFile, 3)
	assert.Equal(t, "/main/file1.bat", Cfg.Routes[0].BatFile[0])
	assert.Equal(t, nestedPath, Cfg.Routes[0].BatFile[1]) // Now resolved to absolute path
	assert.Equal(t, "/main/file2.bat", Cfg.Routes[0].BatFile[2])
}

func TestLoadConfig_MultipleRoutes(t *testing.T) {
	// Test that bat-file expansion works for multiple route entries
	batList1 := `bat-file:
  - /route1/file1.bat
  - /route1/file2.bat`

	batList2 := `bat-file:
  - /route2/file1.bat`

	tmpDir := t.TempDir()
	batListPath1 := filepath.Join(tmpDir, "list1.yaml")
	err := os.WriteFile(batListPath1, []byte(batList1), 0644)
	require.NoError(t, err)

	batListPath2 := filepath.Join(tmpDir, "list2.yaml")
	err = os.WriteFile(batListPath2, []byte(batList2), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "list1.yaml"
  - interfaceId: "Wireguard1"
    bat-file:
      - "list2.yaml"
      - "/direct/file.bat"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Check first route
	assert.Len(t, Cfg.Routes, 2)
	assert.Len(t, Cfg.Routes[0].BatFile, 2)
	assert.Contains(t, Cfg.Routes[0].BatFile, "/route1/file1.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/route1/file2.bat")

	// Check second route
	assert.Len(t, Cfg.Routes[1].BatFile, 2)
	assert.Contains(t, Cfg.Routes[1].BatFile, "/route2/file1.bat")
	assert.Contains(t, Cfg.Routes[1].BatFile, "/direct/file.bat")
}

func TestLoadConfig_BatURLListExpansion(t *testing.T) {
	// Create a bat-url list YAML
	batURLListContent := `bat-url:
  - https://example.com/discord.bat
  - https://example.com/youtube.bat
  - https://example.com/instagram.bat`

	tmpDir := t.TempDir()
	batURLListPath := filepath.Join(tmpDir, "all-urls.yaml")
	err := os.WriteFile(batURLListPath, []byte(batURLListContent), 0644)
	require.NoError(t, err)

	// Create main config that references the bat-url list
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "all-urls.yaml"
      - "https://example.com/extra.bat"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Verify that the YAML file was expanded
	assert.Len(t, Cfg.Routes, 1)
	assert.Equal(t, "Wireguard0", Cfg.Routes[0].InterfaceID)
	// Should have 3 URLs from YAML + 1 extra URL = 4 total
	assert.Len(t, Cfg.Routes[0].BatURL, 4)
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/discord.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/youtube.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/instagram.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/extra.bat")
}

func TestLoadConfig_BatURLListWithAbsolutePath(t *testing.T) {
	// Create a bat-url list YAML
	batURLListContent := `bat-url:
  - https://example.com/file1.bat
  - https://example.com/file2.bat`

	tmpDir := t.TempDir()
	batURLListPath := filepath.Join(tmpDir, "subdir", "urllist.yaml")
	err := os.MkdirAll(filepath.Dir(batURLListPath), 0755)
	require.NoError(t, err)
	err = os.WriteFile(batURLListPath, []byte(batURLListContent), 0644)
	require.NoError(t, err)

	// Create main config with absolute path to bat-url list
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "` + batURLListPath + `"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	assert.Len(t, Cfg.Routes[0].BatURL, 2)
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/file1.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/file2.bat")
}

func TestLoadConfig_BatURLListNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main config that references non-existent bat-url list
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "nonexistent.yaml"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read bat-list")
}

func TestLoadConfig_MixedBatURLsAndYAML(t *testing.T) {
	// Create a bat-url list YAML
	batURLListContent := `bat-url:
  - https://example.com/yaml1.bat
  - https://example.com/yaml2.bat`

	tmpDir := t.TempDir()
	batURLListPath := filepath.Join(tmpDir, "urllist.yaml")
	err := os.WriteFile(batURLListPath, []byte(batURLListContent), 0644)
	require.NoError(t, err)

	// Create main config with mix of URLs and .yaml files
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "https://direct.com/file1.bat"
      - "urllist.yaml"
      - "https://direct.com/file2.bat"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Should have 2 direct URLs + 2 from YAML = 4 total
	assert.Len(t, Cfg.Routes[0].BatURL, 4)
	assert.Equal(t, "https://direct.com/file1.bat", Cfg.Routes[0].BatURL[0])
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/yaml1.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/yaml2.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://direct.com/file2.bat")
}

func TestLoadConfig_BatURLListWithYmlExtension(t *testing.T) {
	// Test that .yml extension works (not just .yaml)
	batURLListContent := `bat-url:
  - https://example.com/file1.bat
  - https://example.com/file2.bat`

	tmpDir := t.TempDir()
	batURLListPath := filepath.Join(tmpDir, "urllist.yml")
	err := os.WriteFile(batURLListPath, []byte(batURLListContent), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "urllist.yml"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	assert.Len(t, Cfg.Routes[0].BatURL, 2)
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/file1.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/file2.bat")
}

func TestLoadConfig_BatURLListEmpty(t *testing.T) {
	// Test empty bat-url list in YAML
	batURLListContent := `bat-url: []`

	tmpDir := t.TempDir()
	batURLListPath := filepath.Join(tmpDir, "empty.yaml")
	err := os.WriteFile(batURLListPath, []byte(batURLListContent), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "empty.yaml"
      - "https://example.com/file.bat"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Should only have the direct URL
	assert.Len(t, Cfg.Routes[0].BatURL, 1)
	assert.Equal(t, "https://example.com/file.bat", Cfg.Routes[0].BatURL[0])
}

func TestLoadConfig_BatURLListInvalidStructure(t *testing.T) {
	// Test YAML with wrong structure (missing bat-url key)
	batURLListContent := `wrong-key:
  - https://example.com/file.bat`

	tmpDir := t.TempDir()
	batURLListPath := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(batURLListPath, []byte(batURLListContent), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "invalid.yaml"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	// Should succeed but with empty list (YAML unmarshals to empty BatURLList)
	assert.NoError(t, err)
	assert.Len(t, Cfg.Routes[0].BatURL, 0)
}

func TestLoadConfig_MultipleRoutesWithBatURL(t *testing.T) {
	// Test that bat-url expansion works for multiple route entries
	batURLList1 := `bat-url:
  - https://route1.com/file1.bat
  - https://route1.com/file2.bat`

	batURLList2 := `bat-url:
  - https://route2.com/file1.bat`

	tmpDir := t.TempDir()
	batURLListPath1 := filepath.Join(tmpDir, "urllist1.yaml")
	err := os.WriteFile(batURLListPath1, []byte(batURLList1), 0644)
	require.NoError(t, err)

	batURLListPath2 := filepath.Join(tmpDir, "urllist2.yaml")
	err = os.WriteFile(batURLListPath2, []byte(batURLList2), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "urllist1.yaml"
  - interfaceId: "Wireguard1"
    bat-url:
      - "urllist2.yaml"
      - "https://direct.com/file.bat"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Check first route
	assert.Len(t, Cfg.Routes, 2)
	assert.Len(t, Cfg.Routes[0].BatURL, 2)
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://route1.com/file1.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://route1.com/file2.bat")

	// Check second route
	assert.Len(t, Cfg.Routes[1].BatURL, 2)
	assert.Contains(t, Cfg.Routes[1].BatURL, "https://route2.com/file1.bat")
	assert.Contains(t, Cfg.Routes[1].BatURL, "https://direct.com/file.bat")
}

func TestLoadConfig_BothBatFileAndBatURL(t *testing.T) {
	// Test that both bat-file and bat-url can be used together
	batFileListContent := `bat-file:
  - /path/to/file1.bat
  - /path/to/file2.bat`

	batURLListContent := `bat-url:
  - https://example.com/file1.bat
  - https://example.com/file2.bat`

	tmpDir := t.TempDir()
	batFileListPath := filepath.Join(tmpDir, "filelist.yaml")
	err := os.WriteFile(batFileListPath, []byte(batFileListContent), 0644)
	require.NoError(t, err)

	batURLListPath := filepath.Join(tmpDir, "urllist.yaml")
	err = os.WriteFile(batURLListPath, []byte(batURLListContent), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "filelist.yaml"
    bat-url:
      - "urllist.yaml"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Check that both bat-file and bat-url were expanded
	assert.Len(t, Cfg.Routes[0].BatFile, 2)
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/to/file1.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/to/file2.bat")

	assert.Len(t, Cfg.Routes[0].BatURL, 2)
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/file1.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/file2.bat")
}

func TestLoadConfig_CombinedBatFileAndBatURLInSameYAML(t *testing.T) {
	// Test that a single YAML file can contain both bat-file and bat-url lists
	// This verifies the optimization where we read each YAML file only once
	combinedListContent := `bat-file:
  - /path/to/file1.bat
  - /path/to/file2.bat
bat-url:
  - https://example.com/url1.bat
  - https://example.com/url2.bat`

	tmpDir := t.TempDir()
	combinedListPath := filepath.Join(tmpDir, "combined.yaml")
	err := os.WriteFile(combinedListPath, []byte(combinedListContent), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "combined.yaml"
      - "/extra/file.bat"
    bat-url:
      - "combined.yaml"
      - "https://extra.com/url.bat"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Verify bat-file expansion
	assert.Len(t, Cfg.Routes[0].BatFile, 3) // 2 from YAML + 1 extra
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/to/file1.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/to/file2.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/extra/file.bat")

	// Verify bat-url expansion
	assert.Len(t, Cfg.Routes[0].BatURL, 3) // 2 from YAML + 1 extra
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/url1.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/url2.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://extra.com/url.bat")
}

func TestLoadConfig_YAMLInBatFileOnlyExpandsBatFile(t *testing.T) {
	// Test that when a YAML file is referenced only in bat-file,
	// we expand only its bat-file content, not bat-url content
	combinedListContent := `bat-file:
  - /path/to/file1.bat
  - /path/to/file2.bat
bat-url:
  - https://example.com/url1.bat
  - https://example.com/url2.bat`

	tmpDir := t.TempDir()
	combinedListPath := filepath.Join(tmpDir, "combined.yaml")
	err := os.WriteFile(combinedListPath, []byte(combinedListContent), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "combined.yaml"
      - "/extra/file.bat"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Verify bat-file expansion - should have files from YAML
	assert.Len(t, Cfg.Routes[0].BatFile, 3) // 2 from YAML + 1 extra
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/to/file1.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/path/to/file2.bat")
	assert.Contains(t, Cfg.Routes[0].BatFile, "/extra/file.bat")

	// Verify bat-url is empty - should NOT expand bat-url from the YAML
	assert.Len(t, Cfg.Routes[0].BatURL, 0)
}

func TestLoadConfig_YAMLInBatURLOnlyExpandsBatURL(t *testing.T) {
	// Test that when a YAML file is referenced only in bat-url,
	// we expand only its bat-url content, not bat-file content
	combinedListContent := `bat-file:
  - /path/to/file1.bat
  - /path/to/file2.bat
bat-url:
  - https://example.com/url1.bat
  - https://example.com/url2.bat`

	tmpDir := t.TempDir()
	combinedListPath := filepath.Join(tmpDir, "combined.yaml")
	err := os.WriteFile(combinedListPath, []byte(combinedListContent), 0644)
	require.NoError(t, err)

	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "combined.yaml"
      - "https://extra.com/url.bat"`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Verify bat-file is empty - should NOT expand bat-file from the YAML
	assert.Len(t, Cfg.Routes[0].BatFile, 0)

	// Verify bat-url expansion - should have URLs from YAML
	assert.Len(t, Cfg.Routes[0].BatURL, 3) // 2 from YAML + 1 extra
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/url1.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://example.com/url2.bat")
	assert.Contains(t, Cfg.Routes[0].BatURL, "https://extra.com/url.bat")
}

func TestLoadConfig_RouteWithoutBatFileOrBatURL(t *testing.T) {
	// Test that routes without bat-file or bat-url fields work correctly
	// and don't trigger any expansion logic
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
  - interfaceId: "Wireguard1"
dns:
  records:
    - domain: "test.local"
      ip: ["192.168.1.100"]`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Verify routes are loaded
	assert.Len(t, Cfg.Routes, 2)
	assert.Equal(t, "Wireguard0", Cfg.Routes[0].InterfaceID)
	assert.Equal(t, "Wireguard1", Cfg.Routes[1].InterfaceID)

	// Verify bat-file and bat-url are empty (not expanded)
	assert.Len(t, Cfg.Routes[0].BatFile, 0)
	assert.Len(t, Cfg.Routes[0].BatURL, 0)
	assert.Len(t, Cfg.Routes[1].BatFile, 0)
	assert.Len(t, Cfg.Routes[1].BatURL, 0)

	// Verify DNS records are loaded correctly
	assert.Len(t, Cfg.DNS.Records, 1)
	assert.Equal(t, "test.local", Cfg.DNS.Records[0].Domain)
}
