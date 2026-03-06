package gokeenrestapi

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MockRouter Interfaces", func() {
	Describe("interface state", func() {
		It("should bring interface down and up", func() {
			server := NewMockRouterServer()
			defer server.Close()

			// Verify initial state
			resp, err := http.Get(server.URL + "/rci/show/interface/Wireguard0")
			Expect(err).NotTo(HaveOccurred())
			var iface gokeenrestapimodels.RciShowInterface
			Expect(json.NewDecoder(resp.Body).Decode(&iface)).To(Succeed())
			_ = resp.Body.Close()
			Expect(iface.Connected).To(Equal(StateConnected))
			Expect(iface.Link).To(Equal(StateUp))
			Expect(iface.State).To(Equal(StateUp))

			// Bring down
			downReq := []gokeenrestapimodels.ParseRequest{{Parse: "interface Wireguard0 down"}}
			body, _ := json.Marshal(downReq)
			resp, _ = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			var downResp []gokeenrestapimodels.ParseResponse
			_ = json.NewDecoder(resp.Body).Decode(&downResp)
			_ = resp.Body.Close()
			Expect(downResp[0].Parse.Status[0].Status).To(Equal(StatusOK))

			resp, _ = http.Get(server.URL + "/rci/show/interface/Wireguard0")
			var downIface gokeenrestapimodels.RciShowInterface
			_ = json.NewDecoder(resp.Body).Decode(&downIface)
			_ = resp.Body.Close()
			Expect(downIface.Connected).To(Equal(StateDisconnected))
			Expect(downIface.Link).To(Equal(StateDown))
			Expect(downIface.State).To(Equal(StateDown))

			// Bring up
			upReq := []gokeenrestapimodels.ParseRequest{{Parse: "interface Wireguard0 up"}}
			body, _ = json.Marshal(upReq)
			resp, _ = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			var upResp []gokeenrestapimodels.ParseResponse
			_ = json.NewDecoder(resp.Body).Decode(&upResp)
			_ = resp.Body.Close()
			Expect(upResp[0].Parse.Status[0].Status).To(Equal(StatusOK))

			resp, _ = http.Get(server.URL + "/rci/show/interface/Wireguard0")
			var upIface gokeenrestapimodels.RciShowInterface
			_ = json.NewDecoder(resp.Body).Decode(&upIface)
			_ = resp.Body.Close()
			Expect(upIface.Connected).To(Equal(StateConnected))
			Expect(upIface.Link).To(Equal(StateUp))
			Expect(upIface.State).To(Equal(StateUp))
		})

		It("should reject state change for non-existent interface", func() {
			server := NewMockRouterServer()
			defer server.Close()

			requests := []gokeenrestapimodels.ParseRequest{{Parse: "interface NonExistent up"}}
			body, _ := json.Marshal(requests)
			resp, _ := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			var responses []gokeenrestapimodels.ParseResponse
			_ = json.NewDecoder(resp.Body).Decode(&responses)
			_ = resp.Body.Close()

			Expect(responses[0].Parse.Status[0].Status).To(Equal(StatusError))
			Expect(responses[0].Parse.Status[0].Message).To(ContainSubstring("does not exist"))
		})
	})

	Describe("interface creation", func() {
		It("should create a new interface", func() {
			server := NewMockRouterServer()
			defer server.Close()

			// Verify doesn't exist
			resp, _ := http.Get(server.URL + "/rci/show/interface/Wireguard1")
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			_ = resp.Body.Close()

			// Create
			createReq := []gokeenrestapimodels.ParseRequest{
				{Parse: "interface Wireguard1 create type Wireguard description TestInterface address 10.1.1.1/24"},
			}
			body, _ := json.Marshal(createReq)
			resp, _ = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			var createResp []gokeenrestapimodels.ParseResponse
			_ = json.NewDecoder(resp.Body).Decode(&createResp)
			_ = resp.Body.Close()
			Expect(createResp[0].Parse.Status[0].Status).To(Equal(StatusOK))

			// Verify in regular listing
			resp, _ = http.Get(server.URL + "/rci/show/interface/Wireguard1")
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			var newIface gokeenrestapimodels.RciShowInterface
			_ = json.NewDecoder(resp.Body).Decode(&newIface)
			_ = resp.Body.Close()
			Expect(newIface.Id).To(Equal("Wireguard1"))
			Expect(newIface.Type).To(Equal(InterfaceTypeWireguard))
			Expect(newIface.Description).To(Equal("TestInterface"))
			Expect(newIface.Address).To(Equal("10.1.1.1/24"))
			Expect(newIface.Connected).To(Equal(StateDisconnected))
			Expect(newIface.Link).To(Equal(StateDown))
			Expect(newIface.State).To(Equal(StateDown))

			// Verify in SC listing
			resp, _ = http.Get(server.URL + "/rci/show/sc/interface/Wireguard1")
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			var newScIface gokeenrestapimodels.RciShowScInterface
			_ = json.NewDecoder(resp.Body).Decode(&newScIface)
			_ = resp.Body.Close()
			Expect(newScIface.Description).To(Equal("TestInterface"))
			Expect(newScIface.IP.Address.Address).To(Equal("10.1.1.1/24"))
		})

		It("should reject creating an existing interface", func() {
			server := NewMockRouterServer()
			defer server.Close()

			requests := []gokeenrestapimodels.ParseRequest{
				{Parse: "interface Wireguard0 create type Wireguard"},
			}
			body, _ := json.Marshal(requests)
			resp, _ := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			var responses []gokeenrestapimodels.ParseResponse
			_ = json.NewDecoder(resp.Body).Decode(&responses)
			_ = resp.Body.Close()

			Expect(responses[0].Parse.Status[0].Status).To(Equal(StatusError))
			Expect(responses[0].Parse.Status[0].Message).To(ContainSubstring("already exists"))
		})
	})

	Describe("AWG config", func() {
		It("should update AWG parameters", func() {
			server := NewMockRouterServer()
			defer server.Close()

			resp, _ := http.Get(server.URL + "/rci/show/sc/interface/Wireguard0")
			var initial gokeenrestapimodels.RciShowScInterface
			_ = json.NewDecoder(resp.Body).Decode(&initial)
			_ = resp.Body.Close()
			Expect(initial.Wireguard.Asc.Jc).To(Equal("3"))
			Expect(initial.Wireguard.Asc.Jmin).To(Equal("50"))

			updateReq := []gokeenrestapimodels.ParseRequest{
				{Parse: "interface Wireguard0 wireguard asc jc 5 jmin 100 jmax 2000 s1 99 s2 10 h1 10 h2 20 h3 30 h4 40"},
			}
			body, _ := json.Marshal(updateReq)
			resp, _ = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			var updateResp []gokeenrestapimodels.ParseResponse
			_ = json.NewDecoder(resp.Body).Decode(&updateResp)
			_ = resp.Body.Close()
			Expect(updateResp[0].Parse.Status[0].Status).To(Equal(StatusOK))

			resp, _ = http.Get(server.URL + "/rci/show/sc/interface/Wireguard0")
			var updated gokeenrestapimodels.RciShowScInterface
			_ = json.NewDecoder(resp.Body).Decode(&updated)
			_ = resp.Body.Close()
			Expect(updated.Wireguard.Asc.Jc).To(Equal("5"))
			Expect(updated.Wireguard.Asc.Jmin).To(Equal("100"))
			Expect(updated.Wireguard.Asc.Jmax).To(Equal("2000"))
			Expect(updated.Wireguard.Asc.S1).To(Equal("99"))
			Expect(updated.Wireguard.Asc.S2).To(Equal("10"))
			Expect(updated.Wireguard.Asc.H1).To(Equal("10"))
			Expect(updated.Wireguard.Asc.H2).To(Equal("20"))
			Expect(updated.Wireguard.Asc.H3).To(Equal("30"))
			Expect(updated.Wireguard.Asc.H4).To(Equal("40"))
		})

		It("should reject AWG config for non-existent interface", func() {
			server := NewMockRouterServer()
			defer server.Close()

			requests := []gokeenrestapimodels.ParseRequest{
				{Parse: "interface NonExistent wireguard asc jc 5"},
			}
			body, _ := json.Marshal(requests)
			resp, _ := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			var responses []gokeenrestapimodels.ParseResponse
			_ = json.NewDecoder(resp.Body).Decode(&responses)
			_ = resp.Body.Close()

			Expect(responses[0].Parse.Status[0].Status).To(Equal(StatusError))
			Expect(responses[0].Parse.Status[0].Message).To(ContainSubstring("does not exist"))
		})
	})
})
