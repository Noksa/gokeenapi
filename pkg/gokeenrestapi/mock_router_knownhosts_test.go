package gokeenrestapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MockRouter KnownHosts", func() {
	It("should delete a known host by MAC", func() {
		server := NewMockRouterServer()
		defer server.Close()

		resp, _ := http.Get(server.URL + "/rci/show/ip/hotspot")
		var initialHotspot gokeenrestapimodels.RciShowIpHotspot
		_ = json.NewDecoder(resp.Body).Decode(&initialHotspot)
		_ = resp.Body.Close()
		initialCount := len(initialHotspot.Host)
		Expect(initialCount).To(BeNumerically(">", 0))

		deviceToDelete := initialHotspot.Host[0]

		deleteReq := []gokeenrestapimodels.ParseRequest{
			{Parse: fmt.Sprintf("no known host \"%s\"", deviceToDelete.Mac)},
		}
		body, _ := json.Marshal(deleteReq)
		resp, _ = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		var deleteResp []gokeenrestapimodels.ParseResponse
		_ = json.NewDecoder(resp.Body).Decode(&deleteResp)
		_ = resp.Body.Close()
		Expect(deleteResp[0].Parse.Status[0].Status).To(Equal(StatusOK))

		resp, _ = http.Get(server.URL + "/rci/show/ip/hotspot")
		var updatedHotspot gokeenrestapimodels.RciShowIpHotspot
		_ = json.NewDecoder(resp.Body).Decode(&updatedHotspot)
		_ = resp.Body.Close()
		Expect(updatedHotspot.Host).To(HaveLen(initialCount - 1))

		for _, host := range updatedHotspot.Host {
			Expect(strings.EqualFold(host.Mac, deviceToDelete.Mac)).To(BeFalse())
		}
	})

	It("should delete a known host without quotes", func() {
		server := NewMockRouterServer()
		defer server.Close()

		resp, _ := http.Get(server.URL + "/rci/show/ip/hotspot")
		var initialHotspot gokeenrestapimodels.RciShowIpHotspot
		_ = json.NewDecoder(resp.Body).Decode(&initialHotspot)
		_ = resp.Body.Close()
		initialCount := len(initialHotspot.Host)

		deviceToDelete := initialHotspot.Host[0]

		deleteReq := []gokeenrestapimodels.ParseRequest{
			{Parse: fmt.Sprintf("no known host %s", deviceToDelete.Mac)},
		}
		body, _ := json.Marshal(deleteReq)
		resp, _ = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		var deleteResp []gokeenrestapimodels.ParseResponse
		_ = json.NewDecoder(resp.Body).Decode(&deleteResp)
		_ = resp.Body.Close()
		Expect(deleteResp[0].Parse.Status[0].Status).To(Equal(StatusOK))

		resp, _ = http.Get(server.URL + "/rci/show/ip/hotspot")
		var updatedHotspot gokeenrestapimodels.RciShowIpHotspot
		_ = json.NewDecoder(resp.Body).Decode(&updatedHotspot)
		_ = resp.Body.Close()
		Expect(updatedHotspot.Host).To(HaveLen(initialCount - 1))
	})

	It("should match MAC case-insensitively", func() {
		server := NewMockRouterServer()
		defer server.Close()

		resp, _ := http.Get(server.URL + "/rci/show/ip/hotspot")
		var initialHotspot gokeenrestapimodels.RciShowIpHotspot
		_ = json.NewDecoder(resp.Body).Decode(&initialHotspot)
		_ = resp.Body.Close()
		initialCount := len(initialHotspot.Host)

		upperMac := strings.ToUpper(initialHotspot.Host[0].Mac)

		deleteReq := []gokeenrestapimodels.ParseRequest{
			{Parse: fmt.Sprintf("no known host \"%s\"", upperMac)},
		}
		body, _ := json.Marshal(deleteReq)
		resp, _ = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		var deleteResp []gokeenrestapimodels.ParseResponse
		_ = json.NewDecoder(resp.Body).Decode(&deleteResp)
		_ = resp.Body.Close()
		Expect(deleteResp[0].Parse.Status[0].Status).To(Equal(StatusOK))

		resp, _ = http.Get(server.URL + "/rci/show/ip/hotspot")
		var updatedHotspot gokeenrestapimodels.RciShowIpHotspot
		_ = json.NewDecoder(resp.Body).Decode(&updatedHotspot)
		_ = resp.Body.Close()
		Expect(updatedHotspot.Host).To(HaveLen(initialCount - 1))
	})

	It("should succeed when deleting non-existent host", func() {
		server := NewMockRouterServer()
		defer server.Close()

		deleteReq := []gokeenrestapimodels.ParseRequest{
			{Parse: "no known host \"ff:ff:ff:ff:ff:ff\""},
		}
		body, _ := json.Marshal(deleteReq)
		resp, _ := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		var deleteResp []gokeenrestapimodels.ParseResponse
		_ = json.NewDecoder(resp.Body).Decode(&deleteResp)
		_ = resp.Body.Close()

		Expect(deleteResp[0].Parse.Status[0].Status).To(Equal(StatusOK))
	})

	It("should reject delete without MAC address", func() {
		server := NewMockRouterServer()
		defer server.Close()

		deleteReq := []gokeenrestapimodels.ParseRequest{
			{Parse: "no known host"},
		}
		body, _ := json.Marshal(deleteReq)
		resp, _ := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		var deleteResp []gokeenrestapimodels.ParseResponse
		_ = json.NewDecoder(resp.Body).Decode(&deleteResp)
		_ = resp.Body.Close()

		Expect(deleteResp[0].Parse.Status[0].Status).To(Equal(StatusError))
	})

	It("should delete multiple hosts in one request", func() {
		server := NewMockRouterServer()
		defer server.Close()

		resp, _ := http.Get(server.URL + "/rci/show/ip/hotspot")
		var initialHotspot gokeenrestapimodels.RciShowIpHotspot
		_ = json.NewDecoder(resp.Body).Decode(&initialHotspot)
		_ = resp.Body.Close()
		initialCount := len(initialHotspot.Host)
		Expect(initialCount).To(BeNumerically(">=", 2))

		deleteReq := []gokeenrestapimodels.ParseRequest{
			{Parse: fmt.Sprintf("no known host \"%s\"", initialHotspot.Host[0].Mac)},
			{Parse: fmt.Sprintf("no known host \"%s\"", initialHotspot.Host[1].Mac)},
		}
		body, _ := json.Marshal(deleteReq)
		resp, _ = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
		var deleteResp []gokeenrestapimodels.ParseResponse
		_ = json.NewDecoder(resp.Body).Decode(&deleteResp)
		_ = resp.Body.Close()

		Expect(deleteResp).To(HaveLen(2))
		Expect(deleteResp[0].Parse.Status[0].Status).To(Equal(StatusOK))
		Expect(deleteResp[1].Parse.Status[0].Status).To(Equal(StatusOK))

		resp, _ = http.Get(server.URL + "/rci/show/ip/hotspot")
		var updatedHotspot gokeenrestapimodels.RciShowIpHotspot
		_ = json.NewDecoder(resp.Body).Decode(&updatedHotspot)
		_ = resp.Body.Close()
		Expect(updatedHotspot.Host).To(HaveLen(initialCount - 2))
	})
})
