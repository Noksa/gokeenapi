package gokeenrestapi

import (
	"net/http/httptest"
	"testing"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/stretchr/testify/suite"
)

type IpTestSuite struct {
	suite.Suite
	server *httptest.Server
}

func (s *IpTestSuite) SetupTest() {
	// Recreate server for each test to ensure test isolation
	// This prevents state changes in one test from affecting others
	s.server = SetupMockRouterForTest(
		WithRoutes([]MockRoute{
			{
				Network:   "10.0.0.0",
				Host:      "192.168.1.1",
				Mask:      "255.255.255.0",
				Interface: "Wireguard0",
				Auto:      false,
			},
			{
				Network:   "172.16.0.0",
				Host:      "192.168.1.1",
				Mask:      "255.255.0.0",
				Interface: "ISP",
				Auto:      false,
			},
		}),
		WithHotspotDevices([]MockHost{
			{
				Name:       "test-device-1",
				Mac:        "aa:bb:cc:dd:ee:ff",
				IP:         "192.168.1.100",
				Hostname:   "device1",
				Registered: false,
				Link:       "up",
				Via:        "ISP",
			},
			{
				Name:       "test-device-2",
				Mac:        "11:22:33:44:55:66",
				IP:         "192.168.1.101",
				Hostname:   "device2",
				Registered: false,
				Link:       "up",
				Via:        "ISP",
			},
		}),
	)
}

func (s *IpTestSuite) TearDownTest() {
	if s.server != nil {
		s.server.Close()
	}
}

func TestIpTestSuite(t *testing.T) {
	suite.Run(t, new(IpTestSuite))
}

func (s *IpTestSuite) TestGetAllHotspots() {
	hotspot, err := Ip.GetAllHotspots()
	s.NoError(err)
	s.Len(hotspot.Host, 2)

	expectedHosts := map[string]string{
		"test-device-1": "aa:bb:cc:dd:ee:ff",
		"test-device-2": "11:22:33:44:55:66",
	}

	for _, host := range hotspot.Host {
		expectedMac, exists := expectedHosts[host.Name]
		s.True(exists, "Unexpected host: %s", host.Name)
		s.Equal(expectedMac, host.Mac, "Host %s MAC mismatch", host.Name)
	}
}

func (s *IpTestSuite) TestDeleteKnownHosts() {
	// Test with empty slice
	err := Ip.DeleteKnownHosts([]string{})
	s.NoError(err)

	// Test with MAC addresses
	macs := []string{"aa:bb:cc:dd:ee:ff", "11:22:33:44:55:66"}
	err = Ip.DeleteKnownHosts(macs)
	s.NoError(err)
}

func (s *IpTestSuite) TestGetAllUserRoutesRciIpRoute() {
	routes, err := Ip.GetAllUserRoutesRciIpRoute("Wireguard0")
	s.NoError(err)
	s.Len(routes, 1)

	for _, route := range routes {
		if route.Network == "10.0.0.0" {
			s.Equal("255.255.255.0", route.Mask)
			s.Equal("Wireguard0", route.Interface)
		}
	}
}

func (s *IpTestSuite) TestDeleteRoutes() {
	routes := []gokeenrestapimodels.RciIpRoute{
		{
			Network:   "10.10.0.0",
			Mask:      "255.255.255.0",
			Host:      "192.168.1.1",
			Interface: "Wireguard0",
			Auto:      false,
		},
	}

	err := Ip.DeleteRoutes(routes, "Wireguard0")
	s.NoError(err)
}

func (s *IpTestSuite) TestAddDnsRecords() {
	domains := []string{"newdomain.com 5.6.7.8", "another.test 192.168.1.200"}
	err := Ip.AddDnsRecords(domains)
	s.NoError(err)
}

func (s *IpTestSuite) TestDeleteDnsRecords() {
	domains := []string{"example.com", "test.local"}
	err := Ip.DeleteDnsRecords(domains)
	s.NoError(err)
}
