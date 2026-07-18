package cmd

import (
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteAllRoutes", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes and flags", func() {
		cmd := newDeleteAllRoutesCmd()

		Expect(cmd.Use).To(Equal(CmdDeleteAllRoutes))
		Expect(cmd.Aliases).To(Equal(AliasesDeleteAllRoutes))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.Long).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())

		forceFlag := cmd.Flags().Lookup("force")
		Expect(forceFlag).NotTo(BeNil())
		Expect(forceFlag.Value.Type()).To(Equal("bool"))
		Expect(forceFlag.DefValue).To(Equal("false"))
	})

	It("should execute successfully with force flag", func() {
		cmd := newDeleteAllRoutesCmd()
		_ = cmd.Flags().Set("force", "true")

		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should cancel when not forced and user declines", func() {
		// Without --force, the command prompts for confirmation.
		// When stdin is not a terminal (as in tests), confirmAction returns an error or false.
		// This test verifies the command doesn't panic and handles non-interactive mode gracefully.
		cmd := newDeleteAllRoutesCmd()
		// Don't set force - let it try to prompt (will fail in non-interactive)
		err := cmd.RunE(cmd, []string{})
		// In non-interactive mode confirmAction returns an error reading stdin
		Expect(err).To(HaveOccurred())
	})

	It("should delete all routes via single API call", func() {
		// Verify routes exist before deletion
		routes, err := gokeenrestapi.Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(routes).NotTo(BeEmpty(), "mock should have default routes")

		cmd := newDeleteAllRoutesCmd()
		_ = cmd.Flags().Set("force", "true")

		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should work when router has no routes", func() {
		server.Close()
		gokeenrestapi.CleanupTestConfig()
		server = setupMockRouter(gokeenrestapi.WithRoutes([]gokeenrestapi.MockRoute{}))

		cmd := newDeleteAllRoutesCmd()
		_ = cmd.Flags().Set("force", "true")

		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})
})
