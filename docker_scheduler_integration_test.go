package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDockerSchedulerIntegration tests scheduler functionality in Docker
func TestDockerSchedulerIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Ensure image is built
	ensureDockerImage(t, ctx)

	t.Run("SchedulerWithMountedConfigs", func(t *testing.T) {
		testSchedulerWithMountedConfigs(t, ctx)
	})

	t.Run("SchedulerConfigValidation", func(t *testing.T) {
		testSchedulerConfigValidation(t, ctx)
	})

	t.Run("SchedulerWithBatFiles", func(t *testing.T) {
		testSchedulerWithBatFiles(t, ctx)
	})
}

// testSchedulerWithMountedConfigs tests scheduler with mounted configuration files
func testSchedulerWithMountedConfigs(t *testing.T, ctx context.Context) {
	tmpDir := createDockerAccessibleTempDir(t)

	// Create router config
	routerConfig := `keenetic:
  url: http://192.168.1.1
  login: test-user
  password: test-password

logs:
  debug: true
`
	routerConfigPath := filepath.Join(tmpDir, "router.yaml")
	err := os.WriteFile(routerConfigPath, []byte(routerConfig), 0644)
	require.NoError(t, err)

	// Create scheduler config
	schedulerConfig := `tasks:
  - name: "Test task"
    commands:
      - show-interfaces
    configs:
      - /etc/gokeenapi/router.yaml
    interval: "1h"
`
	schedulerConfigPath := filepath.Join(tmpDir, "scheduler.yaml")
	err = os.WriteFile(schedulerConfigPath, []byte(schedulerConfig), 0644)
	require.NoError(t, err)

	// Test scheduler dry-run (validate config)
	t.Run("ValidateSchedulerConfig", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
			testImageName,
			"scheduler", "--config", "/etc/gokeenapi/scheduler.yaml", "--help",
		)

		output, err := cmd.CombinedOutput()
		// Help should work even with config file present
		assert.NoError(t, err, "Scheduler help failed: %s", string(output))
		assert.Contains(t, string(output), "scheduler")
	})
}

// testSchedulerConfigValidation tests scheduler configuration validation
func testSchedulerConfigValidation(t *testing.T, ctx context.Context) {
	tmpDir := createDockerAccessibleTempDir(t)

	tests := []struct {
		name        string
		config      string
		expectError bool
		errorMsg    string
	}{
		{
			name: "ValidIntervalTask",
			config: `tasks:
  - name: "Valid interval task"
    commands:
      - show-interfaces
    configs:
      - /etc/gokeenapi/router.yaml
    interval: "1h"
`,
			expectError: false,
		},
		{
			name: "ValidTimeTask",
			config: `tasks:
  - name: "Valid time task"
    commands:
      - show-interfaces
    configs:
      - /etc/gokeenapi/router.yaml
    times:
      - "06:00"
      - "18:00"
`,
			expectError: false,
		},
		{
			name: "InvalidEmptyTasks",
			config: `tasks: []
`,
			expectError: false, // Empty tasks is valid, just does nothing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create scheduler config
			schedulerConfigPath := filepath.Join(tmpDir, "scheduler.yaml")
			err := os.WriteFile(schedulerConfigPath, []byte(tt.config), 0644)
			require.NoError(t, err)

			// Create dummy router config
			routerConfig := `keenetic:
  url: http://192.168.1.1
  login: test
  password: test
`
			routerConfigPath := filepath.Join(tmpDir, "router.yaml")
			err = os.WriteFile(routerConfigPath, []byte(routerConfig), 0644)
			require.NoError(t, err)

			// Try to validate by showing help (config gets loaded)
			cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
				"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
				testImageName,
				"scheduler", "--config", "/etc/gokeenapi/scheduler.yaml", "--help",
			)

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			if tt.expectError {
				assert.Error(t, err, "Expected error for invalid config")
				if tt.errorMsg != "" {
					assert.Contains(t, outputStr, tt.errorMsg)
				}
			} else {
				assert.NoError(t, err, "Unexpected error: %s", outputStr)
			}
		})
	}
}

// testSchedulerWithBatFiles tests scheduler with bat files
func testSchedulerWithBatFiles(t *testing.T, ctx context.Context) {
	tmpDir := createDockerAccessibleTempDir(t)

	// Create bat file directory
	batDir := filepath.Join(tmpDir, "batfiles")
	err := os.MkdirAll(batDir, 0755)
	require.NoError(t, err)

	// Create a test bat file
	batContent := `route add 1.1.1.0 mask 255.255.255.0 0.0.0.0
route add 8.8.8.0 mask 255.255.255.0 0.0.0.0
`
	batFilePath := filepath.Join(batDir, "test.bat")
	err = os.WriteFile(batFilePath, []byte(batContent), 0644)
	require.NoError(t, err)

	// Create router config with bat file reference
	routerConfig := `keenetic:
  url: http://192.168.1.1
  login: test-user
  password: test-password

routes:
  - interfaceId: Wireguard0
    bat-file:
      - /etc/gokeenapi/batfiles/test.bat

logs:
  debug: true
`
	routerConfigPath := filepath.Join(tmpDir, "router.yaml")
	err = os.WriteFile(routerConfigPath, []byte(routerConfig), 0644)
	require.NoError(t, err)

	// Create scheduler config
	schedulerConfig := `tasks:
  - name: "Update routes with bat files"
    commands:
      - add-routes
    configs:
      - /etc/gokeenapi/router.yaml
    interval: "1h"
`
	schedulerConfigPath := filepath.Join(tmpDir, "scheduler.yaml")
	err = os.WriteFile(schedulerConfigPath, []byte(schedulerConfig), 0644)
	require.NoError(t, err)

	// Test that files are accessible
	t.Run("BatFilesAccessible", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"--entrypoint", "sh",
			"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
			testImageName,
			"-c", "cat /etc/gokeenapi/batfiles/test.bat",
		)

		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "Failed to read bat file: %s", string(output))
		assert.Contains(t, string(output), "route add")
	})

	// Test scheduler can load config with bat files
	t.Run("SchedulerLoadsBatFileConfig", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
			testImageName,
			"scheduler", "--config", "/etc/gokeenapi/scheduler.yaml", "--help",
		)

		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "Scheduler failed to load config: %s", string(output))
	})
}

// TestDockerEntrypoint tests Docker entrypoint behavior
func TestDockerEntrypoint(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ensureDockerImage(t, ctx)

	t.Run("DefaultCommand", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm", testImageName)
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "Default command failed: %s", string(output))
		assert.Contains(t, string(output), "gokeenapi", "Should show help by default")
	})

	t.Run("CustomCommand", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm", testImageName, "version")
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "Version command failed: %s", string(output))
		assert.Contains(t, string(output), "Version:", "Should show version")
	})

	t.Run("ShellAccess", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"--entrypoint", "sh",
			testImageName,
			"-c", "which gokeenapi")
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "Shell command failed: %s", string(output))
		assert.Contains(t, string(output), "gokeenapi", "Binary should be in PATH")
	})
}

// TestDockerImageSize tests that the image size is reasonable
func TestDockerImageSize(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ensureDockerImage(t, ctx)

	cmd := exec.CommandContext(ctx, "docker", "images", testImageName,
		"--format", "{{.Size}}")
	output, err := cmd.Output()
	require.NoError(t, err, "Failed to get image size")

	size := string(output)
	t.Logf("Image size: %s", size)

	// Image should be reasonably small (Alpine-based)
	// This is informational, not a hard requirement
	assert.NotEmpty(t, size, "Should have image size")
}
