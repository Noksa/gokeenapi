package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testImageName = "gokeenapi-test:local"
	testTimeout   = 5 * time.Minute
)

// createDockerAccessibleTempDir creates a temp directory that Docker can access
// For Colima on macOS, this needs to be under $HOME
// For Docker Desktop, this would work with default file sharing
// For Linux, any temp dir works
func createDockerAccessibleTempDir(t *testing.T) string {
	// Try to create in $HOME/tmp for Colima compatibility
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to standard temp dir
		return t.TempDir()
	}

	tmpBase := filepath.Join(homeDir, ".tmp", "gokeenapi-docker-test")
	err = os.MkdirAll(tmpBase, 0755)
	if err != nil {
		// Fallback to standard temp dir
		return t.TempDir()
	}

	// Create unique subdirectory
	tmpDir, err := os.MkdirTemp(tmpBase, "test-*")
	if err != nil {
		// Fallback to standard temp dir
		return t.TempDir()
	}

	// Register cleanup
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return tmpDir
}

// buildDockerImage builds the Docker image for testing
func buildDockerImage(t *testing.T, ctx context.Context) {
	t.Log("Building Docker image...")

	version := "test-" + time.Now().Format("20060102-150405")
	buildDate := time.Now().Format(time.RFC3339)

	cmd := exec.CommandContext(ctx, "docker", "build",
		"-t", testImageName,
		"--build-arg", fmt.Sprintf("GOKEENAPI_VERSION=%s", version),
		"--build-arg", fmt.Sprintf("GOKEENAPI_BUILDDATE=%s", buildDate),
		"-f", "Dockerfile",
		".",
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Docker build failed: %s", string(output))
	t.Log("Docker image built successfully")
}

// ensureDockerImage ensures the test image exists
func ensureDockerImage(t *testing.T, ctx context.Context) {
	// Check if image exists
	cmd := exec.CommandContext(ctx, "docker", "images", "-q", testImageName)
	output, err := cmd.Output()

	if err != nil || len(output) == 0 {
		t.Log("Test image not found, building...")
		buildDockerImage(t, ctx)
	}
}
