package cmd

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddAwg", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes and flags", func() {
		cmd := newAddAwgCmd()

		Expect(cmd.Use).To(Equal(CmdAddAwg))
		Expect(cmd.Aliases).To(Equal(AliasesAddAwg))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("conf-file")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("name")).NotTo(BeNil())
	})

	It("should fail when conf-file is missing", func() {
		cmd := newAddAwgCmd()

		err := cmd.RunE(cmd, []string{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("conf-file flag is required"))
	})

	It("should fail with invalid path", func() {
		cmd := newAddAwgCmd()
		_ = cmd.Flags().Set("conf-file", "/nonexistent/path.conf")

		Expect(cmd.RunE(cmd, []string{})).To(HaveOccurred())
	})
})
