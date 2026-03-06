package cmd

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConfigureAwg", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes and flags", func() {
		cmd := newUpdateAwgCmd()

		Expect(cmd.Use).To(Equal(CmdUpdateAwg))
		Expect(cmd.Aliases).To(Equal(AliasesUpdateAwg))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("conf-file")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("interface-id")).NotTo(BeNil())
	})

	It("should fail when conf-file is missing", func() {
		cmd := newUpdateAwgCmd()
		_ = cmd.Flags().Set("interface-id", "Wireguard0")

		err := cmd.RunE(cmd, []string{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("--conf-file flag is required"))
	})

	It("should fail when interface-id is missing", func() {
		cmd := newUpdateAwgCmd()
		_ = cmd.Flags().Set("conf-file", "/tmp/test.conf")

		err := cmd.RunE(cmd, []string{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("--interface-id flag is required"))
	})
})
