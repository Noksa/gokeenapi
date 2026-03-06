package gokeenrestapi

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MockRouter Routes", func() {
	It("should add a route", func() {
		server := NewMockRouterServer()
		defer server.Close()

		resp, err := http.Get(server.URL + "/rci/ip/route")
		Expect(err).NotTo(HaveOccurred())
		var initialRoutes []gokeenrestapimodels.RciIpRoute
		Expect(json.NewDecoder(resp.Body).Decode(&initialRoutes)).To(Succeed())
		_ = resp.Body.Close()
		initialCount := len(initialRoutes)

		requests := []gokeenrestapimodels.ParseRequest{
			{Parse: "ip route 10.0.0.0 255.255.255.0 Wireguard0"},
		}
		body, _ := json.Marshal(requests)
		resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		Expect(err).NotTo(HaveOccurred())
		var responses []gokeenrestapimodels.ParseResponse
		Expect(json.NewDecoder(resp.Body).Decode(&responses)).To(Succeed())
		_ = resp.Body.Close()

		Expect(responses).To(HaveLen(1))
		Expect(responses[0].Parse.Status[0].Status).To(Equal("ok"))

		resp, err = http.Get(server.URL + "/rci/ip/route")
		Expect(err).NotTo(HaveOccurred())
		var routes []gokeenrestapimodels.RciIpRoute
		Expect(json.NewDecoder(resp.Body).Decode(&routes)).To(Succeed())
		_ = resp.Body.Close()

		Expect(routes).To(HaveLen(initialCount + 1))
		found := false
		for _, route := range routes {
			if route.Network == "10.0.0.0" && route.Mask == "255.255.255.0" && route.Interface == "Wireguard0" {
				found = true
			}
		}
		Expect(found).To(BeTrue(), "Added route should be present")
	})

	It("should add a route with auto flag", func() {
		server := NewMockRouterServer()
		defer server.Close()

		requests := []gokeenrestapimodels.ParseRequest{
			{Parse: "ip route 172.16.0.0 255.255.0.0 Wireguard0 auto"},
		}
		body, _ := json.Marshal(requests)
		resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		Expect(err).NotTo(HaveOccurred())
		var responses []gokeenrestapimodels.ParseResponse
		Expect(json.NewDecoder(resp.Body).Decode(&responses)).To(Succeed())
		_ = resp.Body.Close()

		Expect(responses[0].Parse.Status[0].Status).To(Equal("ok"))

		resp, err = http.Get(server.URL + "/rci/ip/route")
		Expect(err).NotTo(HaveOccurred())
		var routes []gokeenrestapimodels.RciIpRoute
		Expect(json.NewDecoder(resp.Body).Decode(&routes)).To(Succeed())
		_ = resp.Body.Close()

		found := false
		for _, route := range routes {
			if route.Network == "172.16.0.0" && route.Mask == "255.255.0.0" && route.Interface == "Wireguard0" {
				found = true
				Expect(route.Auto).To(BeTrue())
			}
		}
		Expect(found).To(BeTrue())
	})

	It("should reject route to non-existent interface", func() {
		server := NewMockRouterServer()
		defer server.Close()

		requests := []gokeenrestapimodels.ParseRequest{
			{Parse: "ip route 10.0.0.0 255.255.255.0 NonExistentInterface"},
		}
		body, _ := json.Marshal(requests)
		resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		Expect(err).NotTo(HaveOccurred())
		var responses []gokeenrestapimodels.ParseResponse
		Expect(json.NewDecoder(resp.Body).Decode(&responses)).To(Succeed())
		_ = resp.Body.Close()

		Expect(responses[0].Parse.Status[0].Status).To(Equal("error"))
		Expect(responses[0].Parse.Status[0].Message).To(ContainSubstring("does not exist"))
	})

	It("should reject malformed route commands", func() {
		server := NewMockRouterServer()
		defer server.Close()

		for _, cmd := range []string{
			"ip route 10.0.0.0 255.255.255.0",
			"ip route 10.0.0.0",
			"ip route",
		} {
			requests := []gokeenrestapimodels.ParseRequest{{Parse: cmd}}
			body, _ := json.Marshal(requests)
			resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			Expect(err).NotTo(HaveOccurred())
			var responses []gokeenrestapimodels.ParseResponse
			Expect(json.NewDecoder(resp.Body).Decode(&responses)).To(Succeed())
			_ = resp.Body.Close()

			Expect(responses[0].Parse.Status[0].Status).To(Equal("error"), "command: %s", cmd)
		}
	})

	It("should delete a route", func() {
		server := NewMockRouterServer()
		defer server.Close()

		// Add
		addReq := []gokeenrestapimodels.ParseRequest{{Parse: "ip route 10.0.0.0 255.255.255.0 Wireguard0"}}
		body, _ := json.Marshal(addReq)
		resp, _ := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		_ = resp.Body.Close()

		resp, _ = http.Get(server.URL + "/rci/ip/route")
		var routesAfterAdd []gokeenrestapimodels.RciIpRoute
		_ = json.NewDecoder(resp.Body).Decode(&routesAfterAdd)
		_ = resp.Body.Close()
		countAfterAdd := len(routesAfterAdd)

		// Delete
		delReq := []gokeenrestapimodels.ParseRequest{{Parse: "no ip route 10.0.0.0 255.255.255.0 Wireguard0"}}
		body, _ = json.Marshal(delReq)
		resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		Expect(err).NotTo(HaveOccurred())
		var responses []gokeenrestapimodels.ParseResponse
		Expect(json.NewDecoder(resp.Body).Decode(&responses)).To(Succeed())
		_ = resp.Body.Close()

		Expect(responses[0].Parse.Status[0].Status).To(Equal("ok"))

		resp, _ = http.Get(server.URL + "/rci/ip/route")
		var routesAfterDelete []gokeenrestapimodels.RciIpRoute
		_ = json.NewDecoder(resp.Body).Decode(&routesAfterDelete)
		_ = resp.Body.Close()

		Expect(routesAfterDelete).To(HaveLen(countAfterAdd - 1))
		for _, route := range routesAfterDelete {
			if route.Network == "10.0.0.0" && route.Mask == "255.255.255.0" && route.Interface == "Wireguard0" {
				Fail("Deleted route should not be present")
			}
		}
	})

	It("should succeed when deleting non-existent route", func() {
		server := NewMockRouterServer()
		defer server.Close()

		requests := []gokeenrestapimodels.ParseRequest{
			{Parse: "no ip route 99.99.99.0 255.255.255.0 Wireguard0"},
		}
		body, _ := json.Marshal(requests)
		resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		Expect(err).NotTo(HaveOccurred())
		var responses []gokeenrestapimodels.ParseResponse
		Expect(json.NewDecoder(resp.Body).Decode(&responses)).To(Succeed())
		_ = resp.Body.Close()

		Expect(responses[0].Parse.Status[0].Status).To(Equal("ok"))
	})

	It("should reject malformed delete route commands", func() {
		server := NewMockRouterServer()
		defer server.Close()

		for _, cmd := range []string{
			"no ip route 10.0.0.0 255.255.255.0",
			"no ip route 10.0.0.0",
			"no ip route",
		} {
			requests := []gokeenrestapimodels.ParseRequest{{Parse: cmd}}
			body, _ := json.Marshal(requests)
			resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			Expect(err).NotTo(HaveOccurred())
			var responses []gokeenrestapimodels.ParseResponse
			Expect(json.NewDecoder(resp.Body).Decode(&responses)).To(Succeed())
			_ = resp.Body.Close()

			Expect(responses[0].Parse.Status[0].Status).To(Equal("error"), "command: %s", cmd)
		}
	})

	It("should round-trip add and delete a route", func() {
		server := NewMockRouterServer()
		defer server.Close()

		resp, _ := http.Get(server.URL + "/rci/ip/route")
		var initialRoutes []gokeenrestapimodels.RciIpRoute
		_ = json.NewDecoder(resp.Body).Decode(&initialRoutes)
		_ = resp.Body.Close()
		initialCount := len(initialRoutes)

		addReq := []gokeenrestapimodels.ParseRequest{{Parse: "ip route 192.168.100.0 255.255.255.0 ISP auto"}}
		body, _ := json.Marshal(addReq)
		resp, _ = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		_ = resp.Body.Close()

		resp, _ = http.Get(server.URL + "/rci/ip/route")
		var afterAdd []gokeenrestapimodels.RciIpRoute
		_ = json.NewDecoder(resp.Body).Decode(&afterAdd)
		_ = resp.Body.Close()
		Expect(afterAdd).To(HaveLen(initialCount + 1))

		delReq := []gokeenrestapimodels.ParseRequest{{Parse: "no ip route 192.168.100.0 255.255.255.0 ISP"}}
		body, _ = json.Marshal(delReq)
		resp, _ = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		_ = resp.Body.Close()

		resp, _ = http.Get(server.URL + "/rci/ip/route")
		var afterDelete []gokeenrestapimodels.RciIpRoute
		_ = json.NewDecoder(resp.Body).Decode(&afterDelete)
		_ = resp.Body.Close()
		Expect(afterDelete).To(HaveLen(initialCount))
	})
})
