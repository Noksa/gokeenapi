package exporter

import (
	"context"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Exporter process", func() {
	It("exits successfully on SIGTERM after starting the real CLI server", func() {
		dir := GinkgoT().TempDir()
		binary := filepath.Join(dir, "gokeenapi")
		buildCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		output, err := exec.CommandContext(buildCtx, "go", "build", "-o", binary, "../..").CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(output))
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		address := listener.Addr().String()
		Expect(listener.Close()).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "router.yaml"), []byte("keenetic:\n  url: http://127.0.0.1:1\n  login: monitor\n  password: secret\n"), 0o600)).To(Succeed())
		path := filepath.Join(dir, "exporter.yaml")
		Expect(os.WriteFile(path, []byte("listen: "+address+"\nrouters:\n  - {name: home, config: router.yaml}\n"), 0o600)).To(Succeed())
		process := exec.Command(binary, "exporter", "--config", path)
		Expect(process.Start()).To(Succeed())
		DeferCleanup(func() {
			if process.ProcessState == nil {
				_ = process.Process.Kill()
				_ = process.Wait()
			}
		})
		Eventually(func() error {
			conn, err := net.DialTimeout("tcp", address, 20*time.Millisecond)
			if err == nil {
				return conn.Close()
			}
			return err
		}).WithTimeout(5 * time.Second).WithPolling(20 * time.Millisecond).Should(Succeed())
		Expect(process.Process.Signal(syscall.SIGTERM)).To(Succeed())
		done := make(chan error, 1)
		go func() { done <- process.Wait() }()
		Eventually(done).WithTimeout(5 * time.Second).Should(Receive(Succeed()))
	})
})

var _ = Describe("Existing interactive command process", func() {
	It("remains interruptible while waiting for deletion confirmation", func() {
		server := gokeenrestapi.SetupMockRouterForTest()
		defer server.Close()
		defer gokeenrestapi.CleanupTestConfig()
		dir := GinkgoT().TempDir()
		binary := filepath.Join(dir, "gokeenapi")
		buildCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		output, err := exec.CommandContext(buildCtx, "go", "build", "-o", binary, "../..").CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), string(output))
		path := filepath.Join(dir, "router.yaml")
		Expect(os.WriteFile(path, []byte("dataDir: "+dir+"\nkeenetic:\n  url: "+server.URL+"\n  login: monitor\n  password: secret\n"), 0o600)).To(Succeed())
		input, writer, err := os.Pipe()
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(input.Close)
		DeferCleanup(writer.Close)
		logPath := filepath.Join(dir, "process.log")
		log, err := os.Create(logPath)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(log.Close)
		process := exec.Command(binary, "delete-all-dns-routing", "--config", path)
		process.Stdin = input
		process.Stdout = log
		process.Stderr = log
		Expect(process.Start()).To(Succeed())
		DeferCleanup(func() {
			if process.ProcessState == nil {
				_ = process.Process.Kill()
				_ = process.Wait()
			}
		})
		Eventually(func() string { b, _ := os.ReadFile(logPath); return string(b) }).WithTimeout(5 * time.Second).WithPolling(20 * time.Millisecond).Should(ContainSubstring("(y/N):"))
		Expect(process.Process.Signal(syscall.SIGTERM)).To(Succeed())
		done := make(chan error, 1)
		go func() { done <- process.Wait() }()
		var exitErr error
		Eventually(done).WithTimeout(time.Second).Should(Receive(&exitErr))
		Expect(exitErr).To(HaveOccurred())
	})
})
