package gokeenrestapi

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MockRouter", func() {
	Describe("basic endpoints", func() {
		It("should register and respond to all endpoints", func() {
			server := NewMockRouterServer()
			defer server.Close()

			tests := []struct {
				name           string
				method         string
				path           string
				expectedStatus int
			}{
				{"Auth GET", "GET", "/auth", http.StatusUnauthorized},
				{"Version", "GET", "/rci/show/version", http.StatusOK},
				{"Interfaces", "GET", "/rci/show/interface", http.StatusOK},
				{"Single Interface", "GET", "/rci/show/interface/Wireguard0", http.StatusOK},
				{"SC Interfaces", "GET", "/rci/show/sc/interface", http.StatusOK},
				{"Routes", "GET", "/rci/ip/route", http.StatusOK},
				{"DNS Records", "GET", "/rci/show/ip/name-server", http.StatusOK},
				{"Hotspot", "GET", "/rci/show/ip/hotspot", http.StatusOK},
				{"Running Config", "GET", "/rci/show/running-config", http.StatusOK},
				{"System Mode", "GET", "/rci/show/system/mode", http.StatusOK},
			}

			for _, tt := range tests {
				req, err := http.NewRequest(tt.method, server.URL+tt.path, nil)
				Expect(err).NotTo(HaveOccurred(), tt.name)

				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred(), tt.name)
				_ = resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(tt.expectedStatus), tt.name)
			}
		})
	})

	Describe("state management", func() {
		It("should return current state via GetState", func() {
			mock := NewMockRouter()
			state := mock.GetState()

			Expect(state.Interfaces).NotTo(BeEmpty())
			Expect(state.Routes).NotTo(BeEmpty())
			Expect(state.DNSRecords).NotTo(BeEmpty())
			Expect(state.HotspotDevices).NotTo(BeEmpty())
			Expect(state.SystemMode.Active).To(Equal("router"))
		})

		It("should restore initial state via ResetState", func() {
			mock := NewMockRouter()
			initialState := mock.GetState()
			initialRouteCount := len(initialState.Routes)

			mock.mu.Lock()
			mock.routes = append(mock.routes, MockRoute{
				Network: "10.0.0.0", Mask: "255.255.255.0", Interface: "Wireguard0",
			})
			mock.mu.Unlock()

			Expect(mock.GetState().Routes).To(HaveLen(initialRouteCount + 1))

			mock.ResetState()
			Expect(mock.GetState().Routes).To(HaveLen(initialRouteCount))
		})
	})

	Describe("custom options", func() {
		It("should apply WithInterfaces option", func() {
			customInterfaces := []MockInterface{
				{ID: "CustomWG", Type: InterfaceTypeWireguard, Address: "192.168.100.1/24", Connected: StateConnected},
			}

			server := NewMockRouterServer(WithInterfaces(customInterfaces))
			defer server.Close()

			resp, err := http.Get(server.URL + "/rci/show/interface")
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			var interfaces map[string]gokeenrestapimodels.RciShowInterface
			Expect(json.NewDecoder(resp.Body).Decode(&interfaces)).To(Succeed())

			customIface, exists := interfaces["CustomWG"]
			Expect(exists).To(BeTrue())
			Expect(customIface.Address).To(Equal("192.168.100.1/24"))
		})

		It("should return 404 for non-existent interface", func() {
			server := NewMockRouterServer()
			defer server.Close()

			resp, err := http.Get(server.URL + "/rci/show/interface/NonExistent")
			Expect(err).NotTo(HaveOccurred())
			_ = resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Describe("parse endpoint", func() {
		It("should return matching length responses", func() {
			server := NewMockRouterServer()
			defer server.Close()

			for _, count := range []int{0, 1, 3} {
				requests := make([]gokeenrestapimodels.ParseRequest, count)
				for i := range requests {
					requests[i] = gokeenrestapimodels.ParseRequest{Parse: "unknown command"}
				}

				body, err := json.Marshal(requests)
				Expect(err).NotTo(HaveOccurred())

				resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
				Expect(err).NotTo(HaveOccurred())

				var responses []gokeenrestapimodels.ParseResponse
				Expect(json.NewDecoder(resp.Body).Decode(&responses)).To(Succeed())
				_ = resp.Body.Close()

				Expect(responses).To(HaveLen(count))
			}
		})

		It("should handle invalid commands", func() {
			server := NewMockRouterServer()
			defer server.Close()

			tests := []struct {
				command       string
				expectSuccess bool
			}{
				{"", true},
				{"unknown command", false},
				{"ip unknown", false},
				{"no ip", false},
				{"no unknown", false},
			}

			for _, tt := range tests {
				requests := []gokeenrestapimodels.ParseRequest{{Parse: tt.command}}
				body, _ := json.Marshal(requests)

				resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
				Expect(err).NotTo(HaveOccurred())

				var responses []gokeenrestapimodels.ParseResponse
				Expect(json.NewDecoder(resp.Body).Decode(&responses)).To(Succeed())
				_ = resp.Body.Close()

				Expect(responses).To(HaveLen(1))
				Expect(responses[0].Parse.Status).To(HaveLen(1))
				if tt.expectSuccess {
					Expect(responses[0].Parse.Status[0].Status).To(Equal("ok"), "command: %s", tt.command)
				} else {
					Expect(responses[0].Parse.Status[0].Status).To(Equal("error"), "command: %s", tt.command)
				}
				Expect(responses[0].Parse.Status[0].Message).NotTo(BeEmpty())
			}
		})

		It("should reject non-POST methods", func() {
			server := NewMockRouterServer()
			defer server.Close()

			for _, method := range []string{"GET", "PUT", "DELETE", "PATCH"} {
				req, err := http.NewRequest(method, server.URL+"/rci/", nil)
				Expect(err).NotTo(HaveOccurred())

				resp, err := http.DefaultClient.Do(req)
				Expect(err).NotTo(HaveOccurred())
				_ = resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed), method)
			}
		})

		It("should reject malformed JSON", func() {
			server := NewMockRouterServer()
			defer server.Close()

			resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader([]byte("not json")))
			Expect(err).NotTo(HaveOccurred())
			_ = resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})
})
