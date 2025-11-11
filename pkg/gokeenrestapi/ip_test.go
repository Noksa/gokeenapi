package gokeenrestapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/stretchr/testify/suite"
)

type IpTestSuite struct {
	suite.Suite
	server *httptest.Server
}

func (s *IpTestSuite) SetupSuite() {
	s.server = s.setupMockServerForIP()
	SetupTestConfig(s.server.URL)
}

func (s *IpTestSuite) TearDownSuite() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *IpTestSuite) setupMockServerForIP() *httptest.Server {
	mux := http.NewServeMux()

	// Hotspot endpoint
	mux.HandleFunc("/rci/show/ip/hotspot", func(w http.ResponseWriter, r *http.Request) {
		hotspot := gokeenrestapimodels.RciShowIpHotspot{
			Host: []gokeenrestapimodels.Host{
				{
					Name:     "test-device-1",
					Mac:      "aa:bb:cc:dd:ee:ff",
					IP:       "192.168.1.100",
					Hostname: "device1",
				},
				{
					Name:     "test-device-2",
					Mac:      "11:22:33:44:55:66",
					IP:       "192.168.1.101",
					Hostname: "device2",
				},
			},
		}
		encodeJSON(w, hotspot)
	})

	// Route endpoints
	mux.HandleFunc("/rci/ip/route", func(w http.ResponseWriter, r *http.Request) {
		routes := []gokeenrestapimodels.RciIpRoute{
			{
				Network:   "10.0.0.0",
				Mask:      "255.255.255.0",
				Host:      "192.168.1.1",
				Interface: "Wireguard0",
				Auto:      false,
			},
			{
				Network:   "172.16.0.0",
				Mask:      "255.255.0.0",
				Host:      "192.168.1.1",
				Interface: "ISP",
				Auto:      false,
			},
		}
		encodeJSON(w, routes)
	})

	// DNS records endpoint
	mux.HandleFunc("/rci/show/ip/name-server", func(w http.ResponseWriter, r *http.Request) {
		dnsRecords := map[string]any{
			"static": map[string]string{
				"example.com": "1.2.3.4",
				"test.local":  "192.168.1.50",
			},
		}
		encodeJSON(w, dnsRecords)
	})

	// Parse endpoint for IP operations
	mux.HandleFunc("/rci/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var requests []gokeenrestapimodels.ParseRequest
		if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		responses := make([]gokeenrestapimodels.ParseResponse, len(requests))
		for i, req := range requests {
			status := StatusOK
			message := "Command executed successfully"

			// Simulate error for invalid commands
			if req.Parse == "invalid command" {
				status = StatusError
				message = "Invalid command"
			}

			responses[i] = gokeenrestapimodels.ParseResponse{
				Parse: gokeenrestapimodels.Parse{
					Status: []gokeenrestapimodels.Status{
						{
							Status:  status,
							Code:    "0",
							Message: message,
						},
					},
				},
			}
		}
		encodeJSON(w, responses)
	})

	return httptest.NewServer(mux)
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
