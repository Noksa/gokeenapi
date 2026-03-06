package cmd

import (
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteRoutes", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes and flags", func() {
		cmd := newDeleteRoutesCmd()

		Expect(cmd.Use).To(Equal(CmdDeleteRoutes))
		Expect(cmd.Aliases).To(Equal(AliasesDeleteRoutes))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("interface-id")).NotTo(BeNil())

		forceFlag := cmd.Flags().Lookup("force")
		Expect(forceFlag).NotTo(BeNil())
		Expect(forceFlag.Value.Type()).To(Equal("bool"))
	})

	It("should execute with interface-id and force", func() {
		config.Cfg.Routes = []config.Route{
			{InterfaceID: "Wireguard0"},
		}

		cmd := newDeleteRoutesCmd()
		_ = cmd.Flags().Set("interface-id", "Wireguard0")
		_ = cmd.Flags().Set("force", "true")

		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should execute with force flag only", func() {
		config.Cfg.Routes = []config.Route{
			{InterfaceID: "Wireguard0"},
		}

		cmd := newDeleteRoutesCmd()
		_ = cmd.Flags().Set("force", "true")

		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})
})
