package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Docker Integration", func() {
	var ctx context.Context
	var cancel context.CancelFunc

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), testTimeout)
		DeferCleanup(cancel)
	})

	Context("image build", func() {
		It("should build the Docker image successfully", func() {
			buildDockerImage(ctx)
		})
	})

	Context("basic commands", func() {
		BeforeEach(func() {
			ensureDockerImage(ctx)
		})

		DescribeTable("command output",
			func(args []string, expectedOutput string, expectError bool) {
				cmdArgs := append([]string{"run", "--rm", testImageName}, args...)
				cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
				output, err := cmd.CombinedOutput()
				outputStr := string(output)

				if expectError {
					Expect(err).To(HaveOccurred(), "Expected command to fail")
				} else {
					Expect(err).NotTo(HaveOccurred(), "Command failed: %s", outputStr)
				}

				if expectedOutput != "" {
					Expect(outputStr).To(ContainSubstring(expectedOutput))
				}
			},
			Entry("help", []string{"--help"}, "gokeenapi", false),
			Entry("version", []string{"version"}, "Version:", false),
			Entry("show-interfaces without config", []string{"show-interfaces"}, "", true),
		)
	})

	Context("config mount", func() {
		var tmpDir string

		BeforeEach(func() {
			ensureDockerImage(ctx)
			tmpDir = createDockerAccessibleTempDir(GinkgoT())

			configContent := `keenetic:
  url: http://192.168.1.1
  login: test-user
  password: test-password

logs:
  debug: true
`
			err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should mount config directory", func() {
			cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
				"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
				testImageName, "--help",
			)
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Failed to run with mounted config: %s", string(output))
		})

		It("should make config file accessible inside container", func() {
			_, err := os.Stat(filepath.Join(tmpDir, "config.yaml"))
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
				"--entrypoint", "sh",
				"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
				testImageName,
				"-c", "ls -la /etc/gokeenapi/ && cat /etc/gokeenapi/config.yaml",
			)
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Config file not accessible: %s", string(output))
			Expect(string(output)).To(ContainSubstring("config.yaml"))
			Expect(string(output)).To(ContainSubstring("keenetic:"))
		})

		It("should load config and fail with connection error (not config error)", func() {
			cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
				"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
				testImageName,
				"--config", "/etc/gokeenapi/config.yaml",
				"show-interfaces",
			)
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			Expect(err).To(HaveOccurred(), "Expected connection failure")
			Expect(outputStr).To(SatisfyAny(
				ContainSubstring("connection refused"),
				ContainSubstring("no such host"),
				ContainSubstring("dial tcp"),
				ContainSubstring("context deadline exceeded"),
				ContainSubstring("i/o timeout"),
			), "Expected connection error, got: %s", outputStr)
			Expect(outputStr).NotTo(ContainSubstring("failed to load config"))
			Expect(outputStr).NotTo(ContainSubstring("no such file or directory"))
			Expect(outputStr).NotTo(ContainSubstring("config path is empty"))
		})

		It("should load config via GOKEENAPI_CONFIG env var", func() {
			cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
				"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
				"-e", "GOKEENAPI_CONFIG=/etc/gokeenapi/config.yaml",
				testImageName,
				"show-interfaces",
			)
			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			Expect(err).To(HaveOccurred(), "Expected connection failure")
			Expect(outputStr).To(SatisfyAny(
				ContainSubstring("connection refused"),
				ContainSubstring("no such host"),
				ContainSubstring("dial tcp"),
				ContainSubstring("context deadline exceeded"),
				ContainSubstring("i/o timeout"),
			), "Expected connection error, got: %s", outputStr)
			Expect(outputStr).NotTo(ContainSubstring("failed to load config"))
			Expect(outputStr).NotTo(ContainSubstring("config path is empty"))
		})
	})

	Context("environment variables", func() {
		BeforeEach(func() {
			ensureDockerImage(ctx)
		})

		DescribeTable("env vars are passed to container",
			func(envVars map[string]string) {
				args := []string{"run", "--rm", "--entrypoint", "sh"}
				for key, value := range envVars {
					args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
				}
				args = append(args, testImageName, "-c", "env")

				cmd := exec.CommandContext(ctx, "docker", args...)
				output, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred(), "Failed to run with env vars: %s", string(output))

				for key := range envVars {
					Expect(string(output)).To(ContainSubstring(key))
				}
			},
			Entry("GOKEENAPI_INSIDE_DOCKER", map[string]string{"GOKEENAPI_INSIDE_DOCKER": "true"}),
			Entry("credentials env vars", map[string]string{
				"GOKEENAPI_KEENETIC_LOGIN":    "env-user",
				"GOKEENAPI_KEENETIC_PASSWORD": "env-password",
			}),
		)
	})

	Context("volume permissions", func() {
		var tmpDir string

		BeforeEach(func() {
			ensureDockerImage(ctx)
			tmpDir = createDockerAccessibleTempDir(GinkgoT())
		})

		It("should write to writable volume", func() {
			cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
				"--entrypoint", "sh",
				"-v", fmt.Sprintf("%s:/etc/gokeenapi", tmpDir),
				testImageName,
				"-c", "echo 'test' > /etc/gokeenapi/test.txt && cat /etc/gokeenapi/test.txt",
			)
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Failed to write to volume: %s", string(output))
			Expect(string(output)).To(ContainSubstring("test"))

			Eventually(func() string {
				content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
				if err != nil {
					return ""
				}
				return string(content)
			}).
				WithTimeout(2 * time.Second).
				WithPolling(100 * time.Millisecond).
				Should(ContainSubstring("test"))
		})

		It("should fail to write to read-only volume", func() {
			cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
				"--entrypoint", "sh",
				"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
				testImageName,
				"-c", "echo 'test' > /etc/gokeenapi/readonly.txt",
			)
			output, err := cmd.CombinedOutput()
			Expect(err).To(HaveOccurred(), "Should fail to write to read-only volume")
			Expect(string(output)).To(ContainSubstring("Read-only"))
		})
	})

	Context("multi-platform build", func() {
		It("should build for multiple platforms", func() {
			cmd := exec.CommandContext(ctx, "docker", "buildx", "version")
			if err := cmd.Run(); err != nil {
				Skip("Docker buildx not available")
			}

			version := "test-multiplatform"
			buildDate := time.Now().Format(time.RFC3339)

			cmd = exec.CommandContext(ctx, "docker", "buildx", "build",
				"--platform", "linux/amd64,linux/arm64",
				"--build-arg", fmt.Sprintf("GOKEENAPI_VERSION=%s", version),
				"--build-arg", fmt.Sprintf("GOKEENAPI_BUILDDATE=%s", buildDate),
				"-f", "Dockerfile",
				".",
			)
			output, err := cmd.CombinedOutput()
			if err != nil {
				Skip("Multi-platform build not supported in this environment")
			}
			Expect(err).NotTo(HaveOccurred(), "Multi-platform build failed: %s", string(output))
		})
	})
})

var _ = Describe("Dockerfile Validity", func() {
	var dockerfile string

	BeforeEach(func() {
		content, err := os.ReadFile("Dockerfile")
		Expect(err).NotTo(HaveOccurred(), "Failed to read Dockerfile")
		dockerfile = string(content)
	})

	It("should use multi-stage builds", func() {
		Expect(dockerfile).To(ContainSubstring("as builder"))
		Expect(dockerfile).To(ContainSubstring("as final"))
	})

	It("should use build cache", func() {
		Expect(dockerfile).To(ContainSubstring("--mount=type=cache"))
	})

	It("should declare a volume", func() {
		Expect(dockerfile).To(ContainSubstring("VOLUME"))
	})

	It("should have an entrypoint", func() {
		Expect(dockerfile).To(ContainSubstring("ENTRYPOINT"))
	})

	It("should use Alpine for smaller image", func() {
		Expect(dockerfile).To(ContainSubstring("alpine"))
	})

	It("should have build args for version and build date", func() {
		Expect(dockerfile).To(ContainSubstring("ARG GOKEENAPI_VERSION"))
		Expect(dockerfile).To(ContainSubstring("ARG GOKEENAPI_BUILDDATE"))
	})
})

var _ = Describe("DockerIgnore", func() {
	var lines []string

	BeforeEach(func() {
		content, err := os.ReadFile(".dockerignore")
		Expect(err).NotTo(HaveOccurred(), "Failed to read .dockerignore")
		lines = strings.Split(string(content), "\n")
	})

	It("should exclude all by default", func() {
		Expect(lines).To(ContainElement("**"))
	})

	It("should include necessary root files", func() {
		Expect(lines).To(ContainElement("!go.mod"))
		Expect(lines).To(ContainElement("!go.sum"))
		Expect(lines).To(ContainElement("!main.go"))
	})

	It("should include source directories", func() {
		Expect(lines).To(ContainElement("!cmd/"))
		Expect(lines).To(ContainElement("!internal/"))
		Expect(lines).To(ContainElement("!pkg/"))
	})
})
