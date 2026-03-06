package gokeenrestapi

import (
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("API", func() {
	var server interface{ Close() }

	BeforeEach(func() {
		server = SetupMockRouterForTest()
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
		CleanupTestConfig()
	})

	It("should ping", func() {
		Expect(Common.Ping()).To(Succeed())
	})

	It("should auth", func() {
		Expect(Common.Auth()).To(Succeed())
	})

	It("should get version", func() {
		version, err := Common.Version()
		Expect(err).NotTo(HaveOccurred())
		Expect(version.Model).To(Equal("KN-1010"))
		Expect(version.Title).To(Equal("4.3.6.3"))
	})

	Context("interfaces", func() {
		It("should get all interfaces", func() {
			interfaces, err := Interface.GetInterfacesViaRciShowInterfaces(false)
			Expect(err).NotTo(HaveOccurred())
			Expect(interfaces).To(HaveLen(2))

			wg, exists := interfaces["Wireguard0"]
			Expect(exists).To(BeTrue())
			Expect(wg.Type).To(Equal(InterfaceTypeWireguard))
			Expect(wg.Address).To(Equal("10.0.0.1/24"))
		})

		It("should filter interfaces by type", func() {
			interfaces, err := Interface.GetInterfacesViaRciShowInterfaces(false, InterfaceTypeWireguard)
			Expect(err).NotTo(HaveOccurred())
			Expect(interfaces).To(HaveLen(1))

			_, exists := interfaces["Wireguard0"]
			Expect(exists).To(BeTrue())
		})

		It("should get single interface", func() {
			iface, err := Interface.GetInterfaceViaRciShowInterfaces("Wireguard0")
			Expect(err).NotTo(HaveOccurred())
			Expect(iface.Id).To(Equal("Wireguard0"))
			Expect(iface.Type).To(Equal(InterfaceTypeWireguard))
		})

		It("should get SC interfaces", func() {
			interfaces, err := Interface.GetInterfacesViaRciShowScInterfaces()
			Expect(err).NotTo(HaveOccurred())
			Expect(interfaces).To(HaveLen(1))

			wg, exists := interfaces["Wireguard0"]
			Expect(exists).To(BeTrue())
			Expect(wg.Description).To(Equal("Test WireGuard interface"))
		})

		It("should up interface", func() {
			Expect(Interface.UpInterface("Wireguard0")).To(Succeed())
		})

		It("should set global IP in interface", func() {
			Expect(Interface.SetGlobalIpInInterface("Wireguard0", true)).To(Succeed())
			Expect(Interface.SetGlobalIpInInterface("Wireguard0", false)).To(Succeed())
		})
	})

	It("should execute post parse", func() {
		parseRequests := []gokeenrestapimodels.ParseRequest{
			{Parse: "interface Wireguard0 up"},
			{Parse: "system configuration save"},
		}

		responses, err := Common.ExecutePostParse(parseRequests...)
		Expect(err).NotTo(HaveOccurred())
		Expect(responses).To(HaveLen(2))

		for i, response := range responses {
			Expect(response.Parse.Status).NotTo(BeEmpty(), "Response %d has no status", i)
			Expect(response.Parse.Status[0].Status).To(Equal(StatusOK), "Response %d status", i)
		}
	})

	It("should show running config", func() {
		config, err := Common.ShowRunningConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(config.Message).NotTo(BeEmpty())
		Expect(config.Message).To(ContainElement(ContainSubstring("system mode router")))
	})

	It("should show static running config", func() {
		if server != nil {
			server.Close()
		}
		s := SetupMockRouterForTest(WithStaticRunningConfig([]string{
			"test running config line 1",
			"test running config line 2",
		}))
		DeferCleanup(s.Close)

		config, err := Common.ShowRunningConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(config.Message).To(HaveLen(2))
		Expect(config.Message[0]).To(Equal("test running config line 1"))
	})

	It("should get all hotspots", func() {
		hotspot, err := Ip.GetAllHotspots()
		Expect(err).NotTo(HaveOccurred())
		Expect(hotspot.Host).To(HaveLen(2))

		expectedHosts := map[string]string{
			"test-device-1": "aa:bb:cc:dd:ee:ff",
			"test-device-2": "11:22:33:44:55:66",
		}

		for _, host := range hotspot.Host {
			expectedMac, exists := expectedHosts[host.Name]
			Expect(exists).To(BeTrue(), "Unexpected host: %s", host.Name)
			Expect(host.Mac).To(Equal(expectedMac), "Host %s MAC mismatch", host.Name)
		}
	})
})
