package cmd

import (
	"io"
	"net/http/httptest"
	"os"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

// setupMockRouter creates a mock router server, authenticates, and registers cleanup.
// Use in BeforeEach blocks for standard command tests.
func setupMockRouter(opts ...gokeenrestapi.MockRouterOption) *httptest.Server {
	server := gokeenrestapi.SetupMockRouterForTest(opts...)
	Expect(gokeenrestapi.Common.Auth()).To(Succeed())
	return server
}

// cleanupMockRouter closes the server and cleans up test config.
// Use in AfterEach blocks.
func cleanupMockRouter(server *httptest.Server) {
	if server != nil {
		server.Close()
	}
	gokeenrestapi.CleanupTestConfig()
}

// captureOutput executes a command's RunE and captures its stdout output.
func captureOutput(cmd *cobra.Command, args []string) (string, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.RunE(cmd, args)

	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)

	return string(out), err
}

// writeTempFile creates a temporary file with the given content and returns its path.
func writeTempFile(dir, name, content string) string {
	path := dir + "/" + name
	ExpectWithOffset(1, os.WriteFile(path, []byte(content), 0644)).To(Succeed())
	return path
}
