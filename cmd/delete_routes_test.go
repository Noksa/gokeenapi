package cmd

import (
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
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

	It("should actually remove routes from router state", func() {
		routesBefore, err := gokeenrestapi.Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(routesBefore).NotTo(BeEmpty(), "mock should have default routes")

		config.Cfg.Routes = []config.Route{
			{InterfaceID: "Wireguard0"},
		}

		cmd := newDeleteRoutesCmd()
		_ = cmd.Flags().Set("interface-id", "Wireguard0")
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		routesAfter, err := gokeenrestapi.Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(routesAfter).To(BeEmpty())
	})

	It("should only delete routes for the specified interface", func() {
		// Add a route to ISP so both interfaces have routes
		server.Close()
		gokeenrestapi.CleanupTestConfig()
		server = setupMockRouter(gokeenrestapi.WithRoutes([]gokeenrestapi.MockRoute{
			{Network: "10.0.0.0", Host: "10.0.0.0", Mask: "255.0.0.0", Interface: "Wireguard0"},
			{Network: "172.16.0.0", Host: "172.16.0.0", Mask: "255.255.0.0", Interface: "ISP"},
		}))

		config.Cfg.Routes = []config.Route{
			{InterfaceID: "Wireguard0"},
			{InterfaceID: "ISP"},
		}

		cmd := newDeleteRoutesCmd()
		_ = cmd.Flags().Set("interface-id", "Wireguard0")
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		wgRoutes, err := gokeenrestapi.Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(wgRoutes).To(BeEmpty())

		ispRoutes, err := gokeenrestapi.Ip.GetAllUserRoutesRciIpRoute("ISP")
		Expect(err).NotTo(HaveOccurred())
		Expect(ispRoutes).To(HaveLen(1))
		Expect(ispRoutes[0].Network).To(Equal("172.16.0.0"))
	})

	It("should handle no routes to delete gracefully", func() {
		server.Close()
		gokeenrestapi.CleanupTestConfig()
		server = setupMockRouter(gokeenrestapi.WithRoutes([]gokeenrestapi.MockRoute{}))

		config.Cfg.Routes = []config.Route{
			{InterfaceID: "Wireguard0"},
		}

		cmd := newDeleteRoutesCmd()
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should fail for non-existent interface", func() {
		cmd := newDeleteRoutesCmd()
		_ = cmd.Flags().Set("interface-id", "NonExistent99")
		_ = cmd.Flags().Set("force", "true")

		Expect(cmd.RunE(cmd, []string{})).To(HaveOccurred())
	})
})
