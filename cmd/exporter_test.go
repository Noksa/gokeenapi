package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

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
})
