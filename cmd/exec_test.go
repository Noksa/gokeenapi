package cmd

import (
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Exec", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes", func() {
		cmd := newExecCmd()

		Expect(cmd.Use).To(Equal(CmdExec))
		Expect(cmd.Aliases).To(Equal(AliasesExec))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())
	})

	It("should execute with arguments", func() {
		cmd := newExecCmd()
		Expect(cmd.RunE(cmd, []string{"system", "configuration", "save"})).To(Succeed())
	})

	It("should handle no arguments", func() {
		cmd := newExecCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})
})
