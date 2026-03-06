package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Docker Scheduler Integration", func() {
	var ctx context.Context
	var cancel context.CancelFunc

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
		DeferCleanup(cancel)
		ensureDockerImage(ctx)
	})

	Context("scheduler with mounted configs", func() {
		var tmpDir string

		BeforeEach(func() {
			tmpDir = createDockerAccessibleTempDir(GinkgoT())

			routerConfig := `keenetic:
  url: http://192.168.1.1
  login: test-user
  password: test-password

logs:
  debug: true
`
			err := os.WriteFile(filepath.Join(tmpDir, "router.yaml"), []byte(routerConfig), 0644)
			Expect(err).NotTo(HaveOccurred())

			schedulerConfig := `tasks:
  - name: "Test task"
    commands:
      - show-interfaces
    configs:
      - /etc/gokeenapi/router.yaml
    interval: "1h"
`
			err = os.WriteFile(filepath.Join(tmpDir, "scheduler.yaml"), []byte(schedulerConfig), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should show scheduler help with mounted config", func() {
			cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
				"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
				testImageName,
				"scheduler", "--config", "/etc/gokeenapi/scheduler.yaml", "--help",
			)
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Scheduler help failed: %s", string(output))
			Expect(string(output)).To(ContainSubstring("scheduler"))
		})
	})

	Context("scheduler config validation", func() {
		var tmpDir string

		BeforeEach(func() {
			tmpDir = createDockerAccessibleTempDir(GinkgoT())

			routerConfig := `keenetic:
  url: http://192.168.1.1
  login: test
  password: test
`
			err := os.WriteFile(filepath.Join(tmpDir, "router.yaml"), []byte(routerConfig), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable("scheduler config variants",
			func(config string, expectError bool) {
				err := os.WriteFile(filepath.Join(tmpDir, "scheduler.yaml"), []byte(config), 0644)
				Expect(err).NotTo(HaveOccurred())

				cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
					"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
					testImageName,
					"scheduler", "--config", "/etc/gokeenapi/scheduler.yaml", "--help",
				)
				output, err := cmd.CombinedOutput()
				outputStr := string(output)

				if expectError {
					Expect(err).To(HaveOccurred(), "Expected error for invalid config")
				} else {
					Expect(err).NotTo(HaveOccurred(), "Unexpected error: %s", outputStr)
				}
			},
			Entry("valid interval task", `tasks:
  - name: "Valid interval task"
    commands:
      - show-interfaces
    configs:
      - /etc/gokeenapi/router.yaml
    interval: "1h"
`, false),
			Entry("valid time task", `tasks:
  - name: "Valid time task"
    commands:
      - show-interfaces
    configs:
      - /etc/gokeenapi/router.yaml
    times:
      - "06:00"
      - "18:00"
`, false),
			Entry("empty tasks list", "tasks: []\n", false),
		)
	})

	Context("scheduler with bat files", func() {
		var tmpDir string

		BeforeEach(func() {
			tmpDir = createDockerAccessibleTempDir(GinkgoT())

			batDir := filepath.Join(tmpDir, "batfiles")
			err := os.MkdirAll(batDir, 0755)
			Expect(err).NotTo(HaveOccurred())

			batContent := `route add 1.1.1.0 mask 255.255.255.0 0.0.0.0
route add 8.8.8.0 mask 255.255.255.0 0.0.0.0
`
			err = os.WriteFile(filepath.Join(batDir, "test.bat"), []byte(batContent), 0644)
			Expect(err).NotTo(HaveOccurred())

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
			err = os.WriteFile(filepath.Join(tmpDir, "router.yaml"), []byte(routerConfig), 0644)
			Expect(err).NotTo(HaveOccurred())

			schedulerConfig := `tasks:
  - name: "Update routes with bat files"
    commands:
      - add-routes
    configs:
      - /etc/gokeenapi/router.yaml
    interval: "1h"
`
			err = os.WriteFile(filepath.Join(tmpDir, "scheduler.yaml"), []byte(schedulerConfig), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should make bat files accessible inside container", func() {
			cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
				"--entrypoint", "sh",
				"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
				testImageName,
				"-c", "cat /etc/gokeenapi/batfiles/test.bat",
			)
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Failed to read bat file: %s", string(output))
			Expect(string(output)).To(ContainSubstring("route add"))
		})

		It("should load scheduler config with bat file references", func() {
			cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
				"-v", fmt.Sprintf("%s:/etc/gokeenapi:ro", tmpDir),
				testImageName,
				"scheduler", "--config", "/etc/gokeenapi/scheduler.yaml", "--help",
			)
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), "Scheduler failed to load config: %s", string(output))
		})
	})
})

var _ = Describe("Docker Entrypoint", func() {
	var ctx context.Context
	var cancel context.CancelFunc

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Minute)
		DeferCleanup(cancel)
		ensureDockerImage(ctx)
	})

	It("should show help by default", func() {
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm", testImageName)
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "Default command failed: %s", string(output))
		Expect(string(output)).To(ContainSubstring("gokeenapi"))
	})

	It("should run version command", func() {
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm", testImageName, "version")
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "Version command failed: %s", string(output))
		Expect(string(output)).To(ContainSubstring("Version:"))
	})

	It("should have gokeenapi binary in PATH", func() {
		cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
			"--entrypoint", "sh",
			testImageName,
			"-c", "which gokeenapi",
		)
		output, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "Shell command failed: %s", string(output))
		Expect(string(output)).To(ContainSubstring("gokeenapi"))
	})
})

var _ = Describe("Docker Image Size", func() {
	It("should report a non-empty image size", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		ensureDockerImage(ctx)

		cmd := exec.CommandContext(ctx, "docker", "images", testImageName,
			"--format", "{{.Size}}")
		output, err := cmd.Output()
		Expect(err).NotTo(HaveOccurred(), "Failed to get image size")
		Expect(string(output)).NotTo(BeEmpty())
	})
})
