package cmd

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteKnownHosts", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes and flags", func() {
		cmd := newDeleteKnownHostsCmd()

		Expect(cmd.Use).To(Equal(CmdDeleteKnownHosts))
		Expect(cmd.Aliases).To(Equal(AliasesDeleteKnownHosts))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())

		Expect(cmd.Flags().Lookup("name-pattern")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("mac-pattern")).NotTo(BeNil())

		forceFlag := cmd.Flags().Lookup("force")
		Expect(forceFlag).NotTo(BeNil())
		Expect(forceFlag.Value.Type()).To(Equal("bool"))
	})

	It("should fail when no pattern is specified", func() {
		cmd := newDeleteKnownHostsCmd()

		err := cmd.RunE(cmd, []string{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exactly one of --name-pattern or --mac-pattern must be specified"))
	})

	It("should fail when both patterns are specified", func() {
		cmd := newDeleteKnownHostsCmd()
		_ = cmd.Flags().Set("name-pattern", "test")
		_ = cmd.Flags().Set("mac-pattern", "aa:bb")

		err := cmd.RunE(cmd, []string{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exactly one of --name-pattern or --mac-pattern must be specified"))
	})

	It("should fail with invalid regex", func() {
		cmd := newDeleteKnownHostsCmd()
		_ = cmd.Flags().Set("name-pattern", "[invalid")

		Expect(cmd.RunE(cmd, []string{})).To(HaveOccurred())
	})

	It("should execute with name-pattern and force", func() {
		cmd := newDeleteKnownHostsCmd()
		_ = cmd.Flags().Set("name-pattern", "nonexistent")
		_ = cmd.Flags().Set("force", "true")

		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should execute with mac-pattern and force", func() {
		cmd := newDeleteKnownHostsCmd()
		_ = cmd.Flags().Set("mac-pattern", "nonexistent")
		_ = cmd.Flags().Set("force", "true")

		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})
})
