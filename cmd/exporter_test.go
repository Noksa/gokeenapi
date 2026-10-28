package cmd

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func executeExporter(args []string) (string, error) {
	var output bytes.Buffer
	root := NewRootCmd()
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs(args)
	err := root.Execute()
	return output.String(), err
}

var _ = Describe("Exporter command", func() {
	It("provides help without router config or authentication", func() {
		output, err := executeExporter([]string{"exporter", "--help"})
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("cached"))
	})
	It("loads an exporter config instead of interpreting it as a router config", func() {
		path := filepath.Join(GinkgoT().TempDir(), "exporter.yaml")
		Expect(os.WriteFile(path, []byte("routers: []\n"), 0o600)).To(Succeed())
		_, err := executeExporter([]string{"exporter", "--config", path})
		Expect(err).To(MatchError(ContainSubstring("at least one router")))
	})
	It("uses GOKEENAPI_CONFIG and respects cancellation", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "router.yaml"), []byte("keenetic:\n  url: http://127.0.0.1:1\n  login: monitor\n  password: secret\n"), 0o600)).To(Succeed())
		path := filepath.Join(dir, "exporter.yaml")
		Expect(os.WriteFile(path, []byte("routers:\n  - {name: home, config: router.yaml}\n"), 0o600)).To(Succeed())
		GinkgoT().Setenv("GOKEENAPI_CONFIG", path)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		root := NewRootCmd()
		root.SetArgs([]string{"exporter"})
		Expect(root.ExecuteContext(ctx)).To(Succeed())
	})
	It("overrides only the listener with --listen without changing the config file", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "router.yaml"), []byte("keenetic:\n  url: http://127.0.0.1:1\n  login: monitor\n  password: secret\n"), 0o600)).To(Succeed())
		path := filepath.Join(dir, "exporter.yaml")
		original := []byte("listen: 127.0.0.1:1\nrouters:\n  - {name: home, config: router.yaml}\n")
		Expect(os.WriteFile(path, original, 0o600)).To(Succeed())
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		address := listener.Addr().String()
		Expect(listener.Close()).To(Succeed())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		root := NewRootCmd()
		root.SetArgs([]string{"exporter", "--config", path, "--listen", address})
		done := make(chan error, 1)
		go func() { done <- root.ExecuteContext(ctx) }()
		client := &http.Client{Timeout: 100 * time.Millisecond}
		Eventually(func() error {
			response, err := client.Get("http://" + address + "/metrics")
			if err != nil {
				return err
			}
			return response.Body.Close()
		}).WithTimeout(3 * time.Second).WithPolling(20 * time.Millisecond).Should(Succeed())
		cancel()
		Eventually(done).WithTimeout(time.Second).Should(Receive(Succeed()))
		contents, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(contents).To(Equal(original))
	})
	DescribeTable("rejects invalid --listen overrides before starting", func(address string) {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "router.yaml"), []byte("keenetic:\n  url: http://127.0.0.1:1\n  login: monitor\n  password: secret\n"), 0o600)).To(Succeed())
		path := filepath.Join(dir, "exporter.yaml")
		Expect(os.WriteFile(path, []byte("routers:\n  - {name: home, config: router.yaml}\n"), 0o600)).To(Succeed())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		root := NewRootCmd()
		root.SetArgs([]string{"exporter", "--config", path, "--listen", address})
		Expect(root.ExecuteContext(ctx)).To(MatchError(ContainSubstring("listen")))
	}, Entry("empty", ""), Entry("missing port", "localhost"), Entry("invalid port", "127.0.0.1:invalid"))
})
