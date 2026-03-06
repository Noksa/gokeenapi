package gokeenrestapi

import (
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ip", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = SetupMockRouterForTest(
			WithRoutes([]MockRoute{
				{Network: "10.0.0.0", Host: "192.168.1.1", Mask: "255.255.255.0", Interface: "Wireguard0", Auto: false},
				{Network: "172.16.0.0", Host: "192.168.1.1", Mask: "255.255.0.0", Interface: "ISP", Auto: false},
			}),
			WithHotspotDevices([]MockHost{
				{Name: "test-device-1", Mac: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.100", Hostname: "device1", Link: "up", Via: "ISP"},
				{Name: "test-device-2", Mac: "11:22:33:44:55:66", IP: "192.168.1.101", Hostname: "device2", Link: "up", Via: "ISP"},
			}),
		)
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
	})

	It("should get all hotspots", func() {
		hotspot, err := Ip.GetAllHotspots()
		Expect(err).NotTo(HaveOccurred())
		Expect(hotspot.Host).To(HaveLen(2))
	})

	It("should delete known hosts", func() {
		Expect(Ip.DeleteKnownHosts([]string{})).To(Succeed())
		Expect(Ip.DeleteKnownHosts([]string{"aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66"})).To(Succeed())
	})

	It("should get user routes for interface", func() {
		routes, err := Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
		Expect(err).NotTo(HaveOccurred())
		Expect(routes).To(HaveLen(1))
		Expect(routes[0].Mask).To(Equal("255.255.255.0"))
		Expect(routes[0].Interface).To(Equal("Wireguard0"))
	})

	It("should delete routes", func() {
		routes := []gokeenrestapimodels.RciIpRoute{
			{Network: "10.10.0.0", Mask: "255.255.255.0", Host: "192.168.1.1", Interface: "Wireguard0", Auto: false},
		}
		Expect(Ip.DeleteRoutes(routes, "Wireguard0")).To(Succeed())
	})

	It("should add DNS records", func() {
		Expect(Ip.AddDnsRecords([]string{"newdomain.com 5.6.7.8", "another.test 192.168.1.200"})).To(Succeed())
	})

	It("should delete DNS records", func() {
		Expect(Ip.DeleteDnsRecords([]string{"example.com", "test.local"})).To(Succeed())
	})
})
