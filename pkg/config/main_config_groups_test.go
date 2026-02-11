package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadConfig_GroupsExpansion tests basic groups expansion from YAML file
func TestLoadConfig_GroupsExpansion(t *testing.T) {
	tmpDir := t.TempDir()

	// Create domain files that groups reference
	domainsDir := filepath.Join(tmpDir, "domains")
	err := os.MkdirAll(domainsDir, 0755)
	require.NoError(t, err)

	// Create youtube.yaml
	youtubeContent := `domain-url:
  - https://example.com/youtube.txt`
	err = os.WriteFile(filepath.Join(domainsDir, "youtube.yaml"), []byte(youtubeContent), 0644)
	require.NoError(t, err)

	// Create telegram.yaml
	telegramContent := `domain-url:
  - https://example.com/telegram.txt`
	err = os.WriteFile(filepath.Join(domainsDir, "telegram.yaml"), []byte(telegramContent), 0644)
	require.NoError(t, err)

	// Create a groups list YAML file
	groupsListContent := `groups:
  - name: youtube
    domain-url:
      - domains/youtube.yaml
    interfaceId: Wireguard0
  - name: telegram
    domain-url:
      - domains/telegram.yaml
    interfaceId: Wireguard0`

	groupsListPath := filepath.Join(tmpDir, "common_groups.yaml")
	err = os.WriteFile(groupsListPath, []byte(groupsListContent), 0644)
	require.NoError(t, err)

	// Create main config that references the groups list
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
dns:
  routes:
    groups:
      - common_groups.yaml`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Verify that the YAML file was expanded
	require.Len(t, Cfg.DNS.Routes.Groups, 2)
	assert.Equal(t, "youtube", Cfg.DNS.Routes.Groups[0].Name)
	assert.Equal(t, "Wireguard0", Cfg.DNS.Routes.Groups[0].InterfaceID)
	assert.False(t, Cfg.DNS.Routes.Groups[0].isFileReference)

	assert.Equal(t, "telegram", Cfg.DNS.Routes.Groups[1].Name)
	assert.Equal(t, "Wireguard0", Cfg.DNS.Routes.Groups[1].InterfaceID)
	assert.False(t, Cfg.DNS.Routes.Groups[1].isFileReference)
}

// TestLoadConfig_GroupsExpansionMixed tests mixing imported and local groups
func TestLoadConfig_GroupsExpansionMixed(t *testing.T) {
	tmpDir := t.TempDir()

	// Create domain files
	domainsDir := filepath.Join(tmpDir, "domains")
	err := os.MkdirAll(domainsDir, 0755)
	require.NoError(t, err)

	youtubeContent := `domain-url:
  - https://example.com/youtube.txt`
	err = os.WriteFile(filepath.Join(domainsDir, "youtube.yaml"), []byte(youtubeContent), 0644)
	require.NoError(t, err)

	localContent := `local1.com
local2.com`
	err = os.WriteFile(filepath.Join(domainsDir, "local.txt"), []byte(localContent), 0644)
	require.NoError(t, err)

	// Create groups list
	groupsListContent := `groups:
  - name: youtube
    domain-url:
      - domains/youtube.yaml
    interfaceId: Wireguard0`

	groupsListPath := filepath.Join(tmpDir, "common.yaml")
	err = os.WriteFile(groupsListPath, []byte(groupsListContent), 0644)
	require.NoError(t, err)

	// Create config with mixed groups
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
dns:
  routes:
    groups:
      - common.yaml
      - name: local-only
        domain-file:
          - domains/local.txt
        interfaceId: GigabitEthernet0`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Verify: 1 from imported + 1 local = 2 total
	require.Len(t, Cfg.DNS.Routes.Groups, 2)
	assert.Equal(t, "youtube", Cfg.DNS.Routes.Groups[0].Name)
	assert.Equal(t, "local-only", Cfg.DNS.Routes.Groups[1].Name)
	assert.Equal(t, "GigabitEthernet0", Cfg.DNS.Routes.Groups[1].InterfaceID)
}

// TestLoadConfig_GroupsNoImport tests that old format (no imports) still works
func TestLoadConfig_GroupsNoImport(t *testing.T) {
	tmpDir := t.TempDir()

	// Create domain files
	domainsDir := filepath.Join(tmpDir, "domains")
	err := os.MkdirAll(domainsDir, 0755)
	require.NoError(t, err)

	youtubeContent := `domain-url:
  - https://example.com/youtube.txt`
	err = os.WriteFile(filepath.Join(domainsDir, "youtube.yaml"), []byte(youtubeContent), 0644)
	require.NoError(t, err)

	workContent := `work1.com
work2.com`
	err = os.WriteFile(filepath.Join(domainsDir, "work.txt"), []byte(workContent), 0644)
	require.NoError(t, err)

	// Create config with old format (no imports)
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
dns:
  routes:
    groups:
      - name: youtube
        domain-url:
          - domains/youtube.yaml
        interfaceId: Wireguard0
      - name: work
        domain-file:
          - domains/work.txt
        interfaceId: Wireguard0`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Verify old format still works
	require.Len(t, Cfg.DNS.Routes.Groups, 2)
	assert.Equal(t, "youtube", Cfg.DNS.Routes.Groups[0].Name)
	assert.Equal(t, "Wireguard0", Cfg.DNS.Routes.Groups[0].InterfaceID)
	assert.False(t, Cfg.DNS.Routes.Groups[0].isFileReference)

	assert.Equal(t, "work", Cfg.DNS.Routes.Groups[1].Name)
	assert.Equal(t, "Wireguard0", Cfg.DNS.Routes.Groups[1].InterfaceID)
	assert.False(t, Cfg.DNS.Routes.Groups[1].isFileReference)
}

// TestLoadConfig_GroupsNameWithYamlExtension tests that groups with .yaml in name but with fields are not treated as imports
func TestLoadConfig_GroupsNameWithYamlExtension(t *testing.T) {
	tmpDir := t.TempDir()

	// Create domain files
	domainsDir := filepath.Join(tmpDir, "domains")
	err := os.MkdirAll(domainsDir, 0755)
	require.NoError(t, err)

	testContent := `test1.com
test2.com`
	err = os.WriteFile(filepath.Join(domainsDir, "test.txt"), []byte(testContent), 0644)
	require.NoError(t, err)

	// Create config with a group that has .yaml in name but is a real group (has domain-file)
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
dns:
  routes:
    groups:
      - name: my-group.yaml
        domain-file:
          - domains/test.txt
        interfaceId: Wireguard0`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Should be treated as a regular group, not a file reference
	require.Len(t, Cfg.DNS.Routes.Groups, 1)
	assert.Equal(t, "my-group.yaml", Cfg.DNS.Routes.Groups[0].Name)
	assert.Len(t, Cfg.DNS.Routes.Groups[0].DomainFile, 1)
	assert.False(t, Cfg.DNS.Routes.Groups[0].isFileReference)
}

// TestLoadConfig_GroupsWithDomainFilePathResolution tests that domain-file paths
// in imported groups are resolved correctly relative to the groups file, not the config file
func TestLoadConfig_GroupsWithDomainFilePathResolution(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure:
	// tmpDir/
	//   config.yaml
	//   common_groups.yaml
	//   domains/
	//     googleplay.txt
	//     work.txt

	domainsDir := filepath.Join(tmpDir, "domains")
	err := os.MkdirAll(domainsDir, 0755)
	require.NoError(t, err)

	// Create actual domain files
	googleplayContent := `play.google.com
android.clients.google.com`
	err = os.WriteFile(filepath.Join(domainsDir, "googleplay.txt"), []byte(googleplayContent), 0644)
	require.NoError(t, err)

	workContent := `work1.com
work2.com`
	err = os.WriteFile(filepath.Join(domainsDir, "work.txt"), []byte(workContent), 0644)
	require.NoError(t, err)

	// Create youtube.yaml for domain-url expansion
	youtubeContent := `domain-url:
  - https://example.com/youtube.txt`
	err = os.WriteFile(filepath.Join(domainsDir, "youtube.yaml"), []byte(youtubeContent), 0644)
	require.NoError(t, err)

	// Create groups list with domain-file paths relative to the groups file
	// These paths should be resolved relative to common_groups.yaml location (tmpDir)
	groupsListContent := `groups:
  - name: googleplay
    domain-file:
      - domains/googleplay.txt
    interfaceId: Wireguard0
  - name: work
    domain-file:
      - domains/work.txt
    interfaceId: Wireguard0
  - name: youtube
    domain-url:
      - domains/youtube.yaml
    interfaceId: Wireguard0`

	groupsListPath := filepath.Join(tmpDir, "common_groups.yaml")
	err = os.WriteFile(groupsListPath, []byte(groupsListContent), 0644)
	require.NoError(t, err)

	// Create main config that imports the groups
	configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
dns:
  routes:
    groups:
      - common_groups.yaml`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load config - this should resolve paths correctly
	err = LoadConfig(configPath)
	assert.NoError(t, err)

	// Verify groups were loaded
	require.Len(t, Cfg.DNS.Routes.Groups, 3)

	// Verify googleplay group
	assert.Equal(t, "googleplay", Cfg.DNS.Routes.Groups[0].Name)
	assert.Equal(t, "Wireguard0", Cfg.DNS.Routes.Groups[0].InterfaceID)
	require.Len(t, Cfg.DNS.Routes.Groups[0].DomainFile, 1)

	// The path should be absolute and point to the correct file
	googleplayPath := Cfg.DNS.Routes.Groups[0].DomainFile[0]
	assert.True(t, filepath.IsAbs(googleplayPath), "googleplay path should be absolute")

	// Verify the file actually exists at the resolved path
	_, err = os.Stat(googleplayPath)
	assert.NoError(t, err, "googleplay.txt should exist at resolved path: %s", googleplayPath)

	// Verify work group
	assert.Equal(t, "work", Cfg.DNS.Routes.Groups[1].Name)
	require.Len(t, Cfg.DNS.Routes.Groups[1].DomainFile, 1)

	workPath := Cfg.DNS.Routes.Groups[1].DomainFile[0]
	assert.True(t, filepath.IsAbs(workPath), "work path should be absolute")

	_, err = os.Stat(workPath)
	assert.NoError(t, err, "work.txt should exist at resolved path: %s", workPath)

	// Verify youtube group (domain-url should be expanded)
	assert.Equal(t, "youtube", Cfg.DNS.Routes.Groups[2].Name)
	require.Len(t, Cfg.DNS.Routes.Groups[2].DomainURL, 1)
	assert.Equal(t, "https://example.com/youtube.txt", Cfg.DNS.Routes.Groups[2].DomainURL[0])
}
