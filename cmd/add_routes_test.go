package cmd

import (
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddRoutes", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes", func() {
		cmd := newAddRoutesCmd()

		Expect(cmd.Use).To(Equal(CmdAddRoutes))
		Expect(cmd.Aliases).To(Equal(AliasesAddRoutes))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())
	})

	It("should execute with configured routes", func() {
		config.Cfg.Routes = []config.Route{
			{
				InterfaceID: "Wireguard0",
				BatFileList: config.BatFileList{BatFile: []string{}},
				BatURLList:  config.BatURLList{BatURL: []string{}},
			},
		}

		cmd := newAddRoutesCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should handle empty routes config", func() {
		config.Cfg.Routes = []config.Route{}

		cmd := newAddRoutesCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should add routes from bat-file and verify state", func() {
		tmpDir := GinkgoT().TempDir()
		batFile := filepath.Join(tmpDir, "routes.bat")
		Expect(os.WriteFile(batFile, []byte(
			"route add 10.10.0.0 mask 255.255.0.0 0.0.0.0\n"+
				"route add 172.16.5.0 mask 255.255.255.0 0.0.0.0\n",
		), 0644)).To(Succeed())

		config.Cfg.Routes = []config.Route{
			{
				InterfaceID: "Wireguard0",
				BatFileList: config.BatFileList{BatFile: []string{batFile}},
			},
		}

		cmd := newAddRoutesCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		routes, err := gokeenrestapi.Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())

		routeNetworks := make([]string, 0, len(routes))
		for _, r := range routes {
			routeNetworks = append(routeNetworks, r.Network)
		}
		Expect(routeNetworks).To(ContainElement("10.10.0.0"))
		Expect(routeNetworks).To(ContainElement("172.16.5.0"))
	})

	It("should skip duplicate routes from bat-file", func() {
		tmpDir := GinkgoT().TempDir()
		batFile := filepath.Join(tmpDir, "routes.bat")
		// 192.168.1.0/255.255.255.0 already exists in mock default state
		Expect(os.WriteFile(batFile, []byte(
			"route add 192.168.1.0 mask 255.255.255.0 0.0.0.0\n"+
				"route add 10.20.0.0 mask 255.255.0.0 0.0.0.0\n",
		), 0644)).To(Succeed())

		config.Cfg.Routes = []config.Route{
			{
				InterfaceID: "Wireguard0",
				BatFileList: config.BatFileList{BatFile: []string{batFile}},
			},
		}

		routesBefore, err := gokeenrestapi.Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())

		cmd := newAddRoutesCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		routesAfter, err := gokeenrestapi.Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		// Only 10.20.0.0 should be added, 192.168.1.0 already existed
		Expect(routesAfter).To(HaveLen(len(routesBefore) + 1))
	})

	It("should fail for non-existent interface", func() {
		config.Cfg.Routes = []config.Route{
			{
				InterfaceID: "NonExistent99",
				BatFileList: config.BatFileList{BatFile: []string{}},
			},
		}

		cmd := newAddRoutesCmd()
		Expect(cmd.RunE(cmd, []string{})).To(HaveOccurred())
	})

	It("should skip lines with invalid format in bat-file", func() {
		tmpDir := GinkgoT().TempDir()
		batFile := filepath.Join(tmpDir, "routes.bat")
		Expect(os.WriteFile(batFile, []byte(
			"# comment line\n"+
				"garbage data\n"+
				"route add 10.30.0.0 mask 255.255.0.0 0.0.0.0\n",
		), 0644)).To(Succeed())

		config.Cfg.Routes = []config.Route{
			{
				InterfaceID: "Wireguard0",
				BatFileList: config.BatFileList{BatFile: []string{batFile}},
			},
		}

		cmd := newAddRoutesCmd()
		// Returns error because of invalid lines (multierr)
		err := cmd.RunE(cmd, []string{})
		Expect(err).To(HaveOccurred())

		// But valid route should still be added
		routes, err := gokeenrestapi.Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		routeNetworks := make([]string, 0, len(routes))
		for _, r := range routes {
			routeNetworks = append(routeNetworks, r.Network)
		}
		Expect(routeNetworks).To(ContainElement("10.30.0.0"))
	})
})
