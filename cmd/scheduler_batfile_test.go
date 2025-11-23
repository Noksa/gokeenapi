package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchedulerWithExpandedBatFiles tests that scheduler works with expanded bat-file lists
func TestSchedulerWithExpandedBatFiles(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := t.TempDir()

	// Create a bat-file list YAML
	batFileListPath := filepath.Join(tmpDir, "common-routes.yaml")
	batFileListContent := `bat-file:
  - /path/to/route1.bat
  - /path/to/route2.bat
  - /path/to/route3.bat
`
	err := os.WriteFile(batFileListPath, []byte(batFileListContent), 0644)
	require.NoError(t, err)

	// Create a router config that references the bat-file list
	routerConfigPath := filepath.Join(tmpDir, "router.yaml")
	routerConfigContent := `keenetic:
  url: http://192.168.1.1
  login: admin
  password: admin

routes:
  - interfaceId: Wireguard0
    bat-file:
      - common-routes.yaml
      - /path/to/extra-route.bat
`
	err = os.WriteFile(routerConfigPath, []byte(routerConfigContent), 0644)
	require.NoError(t, err)

	// Create a scheduler config that uses the router config
	schedulerConfigPath := filepath.Join(tmpDir, "scheduler.yaml")
	schedulerConfigContent := `tasks:
  - name: "Test task with expanded bat-files"
    commands:
      - add-routes
    configs:
      - ` + routerConfigPath + `
    interval: "1h"
`
	err = os.WriteFile(schedulerConfigPath, []byte(schedulerConfigContent), 0644)
	require.NoError(t, err)

	// Load scheduler config
	schedulerCfg, err := config.LoadSchedulerConfig(schedulerConfigPath)
	require.NoError(t, err)
	require.Len(t, schedulerCfg.Tasks, 1)

	// Verify scheduler config loaded correctly
	task := schedulerCfg.Tasks[0]
	assert.Equal(t, "Test task with expanded bat-files", task.Name)
	assert.Equal(t, []string{"add-routes"}, task.Commands)
	assert.Equal(t, []string{routerConfigPath}, task.Configs)
	assert.Equal(t, "1h", task.Interval)

	// Now load the router config to verify bat-file expansion works
	err = config.LoadConfig(routerConfigPath)
	require.NoError(t, err)

	// Verify bat-files were expanded
	require.Len(t, config.Cfg.Routes, 1)
	route := config.Cfg.Routes[0]
	assert.Equal(t, "Wireguard0", route.InterfaceID)

	// Should have 4 bat files: 3 from common-routes.yaml + 1 extra
	expectedBatFiles := []string{
		"/path/to/route1.bat",
		"/path/to/route2.bat",
		"/path/to/route3.bat",
		"/path/to/extra-route.bat",
	}
	assert.Equal(t, expectedBatFiles, route.BatFile)
}

// TestSchedulerValidationWithBatFileExpansion tests that scheduler validation works with bat-file expansion
func TestSchedulerValidationWithBatFileExpansion(t *testing.T) {
	tmpDir := t.TempDir()

	// Create bat-file list
	batFileListPath := filepath.Join(tmpDir, "routes.yaml")
	batFileListContent := `bat-file:
  - /path/to/route1.bat
`
	err := os.WriteFile(batFileListPath, []byte(batFileListContent), 0644)
	require.NoError(t, err)

	// Create router config
	routerConfigPath := filepath.Join(tmpDir, "router.yaml")
	routerConfigContent := `keenetic:
  url: http://192.168.1.1
  login: admin
  password: admin

routes:
  - interfaceId: Wireguard0
    bat-file:
      - routes.yaml
`
	err = os.WriteFile(routerConfigPath, []byte(routerConfigContent), 0644)
	require.NoError(t, err)

	// Create scheduler config
	schedulerConfigPath := filepath.Join(tmpDir, "scheduler.yaml")
	schedulerConfigContent := `tasks:
  - name: "Test validation"
    commands:
      - add-routes
    configs:
      - ` + routerConfigPath + `
    interval: "3h"
`
	err = os.WriteFile(schedulerConfigPath, []byte(schedulerConfigContent), 0644)
	require.NoError(t, err)

	// Load and validate scheduler config
	schedulerCfg, err := config.LoadSchedulerConfig(schedulerConfigPath)
	require.NoError(t, err)

	// Validate task
	err = validateTask(schedulerCfg.Tasks[0])
	assert.NoError(t, err)

	// Verify config file exists (scheduler validation)
	_, err = os.Stat(routerConfigPath)
	assert.NoError(t, err)
}

// TestSchedulerWithNestedBatFileExpansion tests nested YAML references
func TestSchedulerWithNestedBatFileExpansion(t *testing.T) {
	tmpDir := t.TempDir()

	// Create first bat-file list
	list1Path := filepath.Join(tmpDir, "list1.yaml")
	list1Content := `bat-file:
  - /path/to/route1.bat
  - /path/to/route2.bat
`
	err := os.WriteFile(list1Path, []byte(list1Content), 0644)
	require.NoError(t, err)

	// Create second bat-file list
	list2Path := filepath.Join(tmpDir, "list2.yaml")
	list2Content := `bat-file:
  - /path/to/route3.bat
  - /path/to/route4.bat
`
	err = os.WriteFile(list2Path, []byte(list2Content), 0644)
	require.NoError(t, err)

	// Create router config that references both lists
	routerConfigPath := filepath.Join(tmpDir, "router.yaml")
	routerConfigContent := `keenetic:
  url: http://192.168.1.1
  login: admin
  password: admin

routes:
  - interfaceId: Wireguard0
    bat-file:
      - list1.yaml
      - list2.yaml
      - /path/to/direct.bat
`
	err = os.WriteFile(routerConfigPath, []byte(routerConfigContent), 0644)
	require.NoError(t, err)

	// Load router config
	err = config.LoadConfig(routerConfigPath)
	require.NoError(t, err)

	// Verify all bat-files were expanded correctly
	require.Len(t, config.Cfg.Routes, 1)
	route := config.Cfg.Routes[0]

	expectedBatFiles := []string{
		"/path/to/route1.bat",
		"/path/to/route2.bat",
		"/path/to/route3.bat",
		"/path/to/route4.bat",
		"/path/to/direct.bat",
	}
	assert.Equal(t, expectedBatFiles, route.BatFile)
}
