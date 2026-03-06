package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/gomega"
)

const (
	testImageName = "gokeenapi-test:local"
	testTimeout   = 5 * time.Minute
)

type cleanupRegistrar interface {
	Cleanup(func())
}

// createDockerAccessibleTempDir creates a temp directory that Docker can access.
// For Colima on macOS, this needs to be under $HOME.
func createDockerAccessibleTempDir(t cleanupRegistrar) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fallbackTempDir(t)
	}

	tmpBase := filepath.Join(homeDir, ".tmp", "gokeenapi-docker-test")
	if err = os.MkdirAll(tmpBase, 0755); err != nil {
		return fallbackTempDir(t)
	}

	tmpDir, err := os.MkdirTemp(tmpBase, "test-*")
	if err != nil {
		return fallbackTempDir(t)
	}

	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	return tmpDir
}

func fallbackTempDir(t cleanupRegistrar) string {
	tmpDir, err := os.MkdirTemp("", "gokeenapi-docker-test-*")
	Expect(err).NotTo(HaveOccurred())
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	return tmpDir
}

// buildDockerImage builds the Docker image for testing
func buildDockerImage(ctx context.Context) {
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
	Expect(err).NotTo(HaveOccurred(), "Docker build failed: %s", string(output))
}

// ensureDockerImage ensures the test image exists
func ensureDockerImage(ctx context.Context) {
	cmd := exec.CommandContext(ctx, "docker", "images", "-q", testImageName)
	output, err := cmd.Output()

	if err != nil || len(output) == 0 {
		buildDockerImage(ctx)
	}
}
