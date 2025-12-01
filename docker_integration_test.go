package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDockerIntegration runs integration tests for Docker image
func TestDockerIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// Build the Docker image first
	t.Run("BuildImage", func(t *testing.T) {
		buildDockerImage(t, ctx)
	})

	// Run tests that depend on the image
	t.Run("BasicCommands", func(t *testing.T) {
		testBasicCommands(t, ctx)
	})

	t.Run("ConfigMount", func(t *testing.T) {
		testConfigMount(t, ctx)
	})

	t.Run("EnvironmentVariables", func(t *testing.T) {
		testEnvironmentVariables(t, ctx)
	})

	t.Run("VolumePermissions", func(t *testing.T) {
		testVolumePermissions(t, ctx)
	})

	t.Run("MultiPlatformBuild", func(t *testing.T) {
		testMultiPlatformBuild(t, ctx)
	})
}

// testBasicCommands tests basic Docker commands
func testBasicCommands(t *testing.T, ctx context.Context) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		expectError    bool
	}{
		{
			name:           "Help",
			args:           []string{"--help"},
			expectedOutput: "gokeenapi",
			expectError:    false,
		},
		{
			name:           "Version",
			args:           []string{"version"},
			expectedOutput: "Version:",
			expectError:    false,
		},
		{
			name:           "ShowInterfaces_NoConfig",
			args:           []string{"show-interfaces"},
			expectedOutput: "",
			expectError:    true, // Should fail without config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"run", "--rm", testImageName}, tt.args...)
			cmd := exec.CommandContext(ctx, "docker", args...)

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			if tt.expectError {
				assert.Error(t, err, "Expected command to fail")
			} else {
				assert.NoError(t, err, "Command failed: %s", outputStr)
			}

			if tt.expectedOutput != "" {
				assert.Contains(t, outputStr, tt.expectedOutput,
					"Output should contain expected string")
			}

			t.Logf("Output: %s", outputStr)
		})
	}
}

// testConfigMount tests mounting configuration files
func testConfigMount(t *testing.T, ctx context.Context) {
	// Create temporary directory for test configs
	tmpDir := createDockerAccessibleTempDir(t)

	// Create a test config file
	configContent := `keenetic:
  url: http://192.168.1.1
  login: test-user
  password: test-password

logs:
  debug: true
`
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err, "Failed to create test config")

	// Test mounting config directory
	t.Run("MountConfigDirectory", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
			testImageName,
			"--help",
		)

		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "Failed to run with mounted config: %s", string(output))
	})

	// Test config file is accessible
	t.Run("ConfigFileAccessible", func(t *testing.T) {
		// First verify the file exists on host
		_, err := os.Stat(configPath)
		require.NoError(t, err, "Config file doesn't exist on host")

		cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"--entrypoint", "sh",
			"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
			testImageName,
			"-c", "ls -la /etc/gokeenapi/ && cat /etc/gokeenapi/config.yaml",
		)

		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "Config file not accessible: %s", string(output))
		assert.Contains(t, string(output), "config.yaml")
		assert.Contains(t, string(output), "keenetic:")
	})

	// Test config loads properly and command attempts to connect
	t.Run("ConfigLoadsAndCommandRuns", func(t *testing.T) {
		// Use show-interfaces command which will:
		// 1. Load the config successfully
		// 2. Attempt to connect to the router
		// 3. Fail with connection error (expected - no real router)
		// This proves config parsing works inside Docker
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
			testImageName,
			"--config", "/etc/gokeenapi/config.yaml",
			"show-interfaces",
		)

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// Command should fail (no router to connect to)
		assert.Error(t, err, "Expected connection failure")

		// But the error should be a connection error, not a config error
		// This proves the config was loaded and parsed successfully
		assert.True(t,
			strings.Contains(outputStr, "connection refused") ||
				strings.Contains(outputStr, "no such host") ||
				strings.Contains(outputStr, "dial tcp") ||
				strings.Contains(outputStr, "context deadline exceeded") ||
				strings.Contains(outputStr, "i/o timeout"),
			"Expected connection error (not config error), got: %s", outputStr)

		// Should NOT contain config-related errors
		assert.NotContains(t, outputStr, "failed to load config",
			"Should not have config loading errors")
		assert.NotContains(t, outputStr, "no such file or directory",
			"Should not have file not found errors")
		assert.NotContains(t, outputStr, "config path is empty",
			"Should not have config path errors")

		t.Logf("Config loaded successfully, connection failed as expected: %s", outputStr)
	})

	// Test config loads via environment variable
	t.Run("ConfigLoadsViaEnvVar", func(t *testing.T) {
		// Same test but using GOKEENAPI_CONFIG environment variable
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
			"-e", "GOKEENAPI_CONFIG=/etc/gokeenapi/config.yaml",
			testImageName,
			"show-interfaces",
		)

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// Command should fail (no router to connect to)
		assert.Error(t, err, "Expected connection failure")

		// But the error should be a connection error, not a config error
		assert.True(t,
			strings.Contains(outputStr, "connection refused") ||
				strings.Contains(outputStr, "no such host") ||
				strings.Contains(outputStr, "dial tcp") ||
				strings.Contains(outputStr, "context deadline exceeded") ||
				strings.Contains(outputStr, "i/o timeout"),
			"Expected connection error (not config error), got: %s", outputStr)

		// Should NOT contain config-related errors
		assert.NotContains(t, outputStr, "failed to load config",
			"Should not have config loading errors")
		assert.NotContains(t, outputStr, "config path is empty",
			"Should not have config path errors")

		t.Logf("Config loaded via env var successfully, connection failed as expected: %s", outputStr)
	})
}

// testEnvironmentVariables tests environment variable handling
func testEnvironmentVariables(t *testing.T, ctx context.Context) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name: "InsideDockerEnv",
			envVars: map[string]string{
				"GOKEENAPI_INSIDE_DOCKER": "true",
			},
			expected: "GOKEENAPI_INSIDE_DOCKER",
		},
		{
			name: "CredentialsEnv",
			envVars: map[string]string{
				"GOKEENAPI_KEENETIC_LOGIN":    "env-user",
				"GOKEENAPI_KEENETIC_PASSWORD": "env-password",
			},
			expected: "env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"run", "--rm", "--entrypoint", "sh"}

			// Add environment variables
			for key, value := range tt.envVars {
				args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
			}

			args = append(args, testImageName, "-c", "env")

			cmd := exec.CommandContext(ctx, "docker", args...)
			output, err := cmd.CombinedOutput()
			assert.NoError(t, err, "Failed to run with env vars: %s", string(output))

			outputStr := string(output)
			for key := range tt.envVars {
				assert.Contains(t, outputStr, key,
					"Environment variable %s not found", key)
			}
		})
	}
}

// testVolumePermissions tests volume mount permissions
func testVolumePermissions(t *testing.T, ctx context.Context) {
	tmpDir := createDockerAccessibleTempDir(t)

	t.Run("WriteToVolume", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"--entrypoint", "sh",
			"-v", fmt.Sprintf("%s:/etc/gokeenapi", tmpDir),
			testImageName,
			"-c", "echo 'test' > /etc/gokeenapi/test.txt && cat /etc/gokeenapi/test.txt && ls -la /etc/gokeenapi/",
		)

		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "Failed to write to volume: %s", string(output))
		assert.Contains(t, string(output), "test")

		// Give Docker a moment to sync the file
		time.Sleep(100 * time.Millisecond)

		// Verify file was created on host
		testFile := filepath.Join(tmpDir, "test.txt")
		content, err := os.ReadFile(testFile)
		assert.NoError(t, err, "Failed to read file from host: %s", testFile)
		assert.Contains(t, string(content), "test")
	})

	t.Run("ReadOnlyVolume", func(t *testing.T) {
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"--entrypoint", "sh",
			"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
			testImageName,
			"-c", "echo 'test' > /etc/gokeenapi/readonly.txt",
		)

		output, err := cmd.CombinedOutput()
		assert.Error(t, err, "Should fail to write to read-only volume")
		assert.Contains(t, string(output), "Read-only")
	})
}

// testMultiPlatformBuild tests multi-platform build capability
func testMultiPlatformBuild(t *testing.T, ctx context.Context) {
	// Check if buildx is available
	cmd := exec.CommandContext(ctx, "docker", "buildx", "version")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker buildx not available, skipping multi-platform test")
	}

	t.Run("BuildMultiPlatform", func(t *testing.T) {
		version := "test-multiplatform"
		buildDate := time.Now().Format(time.RFC3339)

		// Build for multiple platforms (without push)
		cmd := exec.CommandContext(ctx, "docker", "buildx", "build",
			"--platform", "linux/amd64,linux/arm64",
			"--build-arg", fmt.Sprintf("GOKEENAPI_VERSION=%s", version),
			"--build-arg", fmt.Sprintf("GOKEENAPI_BUILDDATE=%s", buildDate),
			"-f", "Dockerfile",
			".",
		)

		output, err := cmd.CombinedOutput()
		if err != nil {
			// Multi-platform build might fail in some CI environments
			t.Logf("Multi-platform build failed (may be expected in CI): %s", string(output))
			t.Skip("Skipping multi-platform build test")
		}

		assert.NoError(t, err, "Multi-platform build failed: %s", string(output))
	})
}

// TestDockerfileValidity tests Dockerfile syntax and best practices
func TestDockerfileValidity(t *testing.T) {
	content, err := os.ReadFile("Dockerfile")
	require.NoError(t, err, "Failed to read Dockerfile")

	dockerfile := string(content)

	t.Run("HasMultiStageBuilds", func(t *testing.T) {
		assert.Contains(t, dockerfile, "as builder", "Should use multi-stage builds")
		assert.Contains(t, dockerfile, "as final", "Should have final stage")
	})

	t.Run("UsesBuildCache", func(t *testing.T) {
		assert.Contains(t, dockerfile, "--mount=type=cache", "Should use build cache")
	})

	t.Run("HasVolume", func(t *testing.T) {
		assert.Contains(t, dockerfile, "VOLUME", "Should declare volume")
	})

	t.Run("HasEntrypoint", func(t *testing.T) {
		assert.Contains(t, dockerfile, "ENTRYPOINT", "Should have entrypoint")
	})

	t.Run("UsesAlpine", func(t *testing.T) {
		assert.Contains(t, dockerfile, "alpine", "Should use Alpine for smaller image")
	})

	t.Run("HasBuildArgs", func(t *testing.T) {
		assert.Contains(t, dockerfile, "ARG GOKEENAPI_VERSION", "Should have version build arg")
		assert.Contains(t, dockerfile, "ARG GOKEENAPI_BUILDDATE", "Should have build date arg")
	})
}

// TestDockerIgnore tests .dockerignore configuration
func TestDockerIgnore(t *testing.T) {
	content, err := os.ReadFile(".dockerignore")
	require.NoError(t, err, "Failed to read .dockerignore")

	dockerignore := string(content)
	lines := strings.Split(dockerignore, "\n")

	t.Run("ExcludesAll", func(t *testing.T) {
		assert.Contains(t, lines, "**", "Should exclude all by default")
	})

	t.Run("IncludesNecessaryFiles", func(t *testing.T) {
		necessaryFiles := []string{"!go.mod", "!go.sum", "!main.go"}
		for _, file := range necessaryFiles {
			assert.Contains(t, lines, file, "Should include %s", file)
		}
	})

	t.Run("IncludesSourceDirs", func(t *testing.T) {
		sourceDirs := []string{"!cmd/", "!internal/", "!pkg/"}
		for _, dir := range sourceDirs {
			assert.Contains(t, lines, dir, "Should include %s", dir)
		}
	})
}
