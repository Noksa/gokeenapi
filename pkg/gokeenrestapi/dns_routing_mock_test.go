package gokeenrestapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// postParse is a helper to send parse requests to the mock server.
func postParse(serverURL string, requests []gokeenrestapimodels.ParseRequest) []gokeenrestapimodels.ParseResponse {
	body, err := json.Marshal(requests)
	Expect(err).NotTo(HaveOccurred())
	resp, err := http.Post(serverURL+"/rci/", "application/json", bytes.NewReader(body))
	Expect(err).NotTo(HaveOccurred())
	defer func() { _ = resp.Body.Close() }()
	var responses []gokeenrestapimodels.ParseResponse
	Expect(json.NewDecoder(resp.Body).Decode(&responses)).To(Succeed())
	return responses
}

var _ = Describe("MockRouter DNS Routing", func() {
	Describe("object-group operations", func() {
		It("should create an object-group", func() {
			server := NewMockRouterServer()
			defer server.Close()

			responses := postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "object-group fqdn social-media"},
			})
			Expect(responses).To(HaveLen(1))
			Expect(responses[0].Parse.Status[0].Status).To(Equal("ok"))
			Expect(responses[0].Parse.Status[0].Message).To(ContainSubstring("social-media"))
		})

		It("should add domain to object-group", func() {
			server := NewMockRouterServer()
			defer server.Close()

			postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "object-group fqdn social-media"},
			})

			responses := postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "object-group fqdn social-media include facebook.com"},
			})
			Expect(responses[0].Parse.Status[0].Status).To(Equal("ok"))
			Expect(responses[0].Parse.Status[0].Message).To(ContainSubstring("facebook.com"))
		})

		It("should delete an object-group", func() {
			server := NewMockRouterServer()
			defer server.Close()

			postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "object-group fqdn social-media"},
			})

			responses := postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "no object-group fqdn social-media"},
			})
			Expect(responses[0].Parse.Status[0].Status).To(Equal("ok"))
		})

		It("should remove domain from object-group", func() {
			server := NewMockRouterServer()
			defer server.Close()

			postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "object-group fqdn test-group"},
				{Parse: "object-group fqdn test-group include example.com"},
				{Parse: "object-group fqdn test-group include test.com"},
			})

			responses := postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "no object-group fqdn test-group include example.com"},
			})
			Expect(responses[0].Parse.Status[0].Status).To(Equal("ok"))
			Expect(responses[0].Parse.Status[0].Message).To(ContainSubstring("example.com"))
			Expect(responses[0].Parse.Status[0].Message).To(ContainSubstring("removed"))
		})

		It("should reject adding domain to non-existent group", func() {
			server := NewMockRouterServer()
			defer server.Close()

			responses := postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "object-group fqdn nonexistent include facebook.com"},
			})
			Expect(responses[0].Parse.Status[0].Status).To(Equal("error"))
			Expect(responses[0].Parse.Status[0].Message).To(ContainSubstring("does not exist"))
		})
	})

	Describe("dns-proxy routes", func() {
		It("should create a dns-proxy route", func() {
			server := NewMockRouterServer()
			defer server.Close()

			responses := postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "object-group fqdn social-media"},
				{Parse: "dns-proxy route object-group social-media Wireguard0 auto"},
			})
			Expect(responses).To(HaveLen(2))
			Expect(responses[0].Parse.Status[0].Status).To(Equal("ok"))
			Expect(responses[1].Parse.Status[0].Status).To(Equal("ok"))
			Expect(responses[1].Parse.Status[0].Message).To(ContainSubstring("social-media"))
			Expect(responses[1].Parse.Status[0].Message).To(ContainSubstring("Wireguard0"))
		})

		It("should delete a dns-proxy route", func() {
			server := NewMockRouterServer()
			defer server.Close()

			postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "object-group fqdn social-media"},
				{Parse: "dns-proxy route object-group social-media Wireguard0 auto"},
			})

			responses := postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "no dns-proxy route object-group social-media Wireguard0"},
			})
			Expect(responses[0].Parse.Status[0].Status).To(Equal("ok"))
		})

		It("should reject route to non-existent interface", func() {
			server := NewMockRouterServer()
			defer server.Close()

			responses := postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "object-group fqdn social-media"},
				{Parse: "dns-proxy route object-group social-media NonExistentInterface auto"},
			})
			Expect(responses[0].Parse.Status[0].Status).To(Equal("ok"))
			Expect(responses[1].Parse.Status[0].Status).To(Equal("error"))
			Expect(responses[1].Parse.Status[0].Message).To(ContainSubstring("does not exist"))
		})
	})

	Describe("full workflow", func() {
		It("should complete create-add-route workflow and reflect in running config", func() {
			server := NewMockRouterServer()
			defer server.Close()

			responses := postParse(server.URL, []gokeenrestapimodels.ParseRequest{
				{Parse: "object-group fqdn social-media"},
				{Parse: "object-group fqdn social-media include facebook.com"},
				{Parse: "object-group fqdn social-media include instagram.com"},
				{Parse: "object-group fqdn social-media include twitter.com"},
				{Parse: "dns-proxy route object-group social-media Wireguard0 auto"},
			})
			Expect(responses).To(HaveLen(5))
			for i, r := range responses {
				Expect(r.Parse.Status[0].Status).To(Equal("ok"), "Response %d should be successful", i)
			}

			resp, err := http.Get(server.URL + "/rci/show/running-config")
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()
			var runningConfig gokeenrestapimodels.RunningConfig
			Expect(json.NewDecoder(resp.Body).Decode(&runningConfig)).To(Succeed())

			configStr := strings.Join(runningConfig.Message, "\n")
			Expect(configStr).To(ContainSubstring("object-group fqdn social-media"))
			Expect(configStr).To(ContainSubstring("include facebook.com"))
			Expect(configStr).To(ContainSubstring("include instagram.com"))
			Expect(configStr).To(ContainSubstring("include twitter.com"))
			Expect(configStr).To(ContainSubstring("dns-proxy route object-group social-media Wireguard0 auto"))
		})
	})

	Describe("WithDnsRoutingGroups option", func() {
		It("should pre-configure groups and routes", func() {
			groups := []MockDnsRoutingGroup{
				{Name: "test-group", Domains: []string{"example.com", "test.com"}},
			}
			routes := []MockDnsProxyRoute{
				{GroupName: "test-group", InterfaceID: "Wireguard0", Mode: "auto"},
			}

			server := NewMockRouterServer(WithDnsRoutingGroups(groups, routes))
			defer server.Close()

			resp, err := http.Get(server.URL + "/rci/show/running-config")
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()
			var runningConfig gokeenrestapimodels.RunningConfig
			Expect(json.NewDecoder(resp.Body).Decode(&runningConfig)).To(Succeed())

			configStr := strings.Join(runningConfig.Message, "\n")
			Expect(configStr).To(ContainSubstring("object-group fqdn test-group"))
			Expect(configStr).To(ContainSubstring("include example.com"))
			Expect(configStr).To(ContainSubstring("include test.com"))
			Expect(configStr).To(ContainSubstring("dns-proxy route object-group test-group Wireguard0 auto"))
		})
	})

	Describe("API client integration", func() {
		It("should be idempotent when adding same groups twice", func() {
			server := NewMockRouterServer(WithVersion("5.0.1"))
			defer server.Close()
			SetupTestConfig(server.URL)
			defer CleanupTestConfig()
			Expect(Common.Auth()).To(Succeed())

			tmpDir := GinkgoT().TempDir()
			domainFile := filepath.Join(tmpDir, "domains.txt")
			Expect(os.WriteFile(domainFile, []byte("example.com\ntest.com\n"), 0644)).To(Succeed())

			groups := []config.DnsRoutingGroup{
				{Name: "test-group", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"},
			}

			Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())

			existingGroups, err := DnsRouting.GetExistingDnsRoutingGroups()
			Expect(err).NotTo(HaveOccurred())
			Expect(existingGroups).To(HaveKey("test-group"))
			Expect(existingGroups["test-group"]).To(ConsistOf("example.com", "test.com"))

			Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())

			existingGroups, err = DnsRouting.GetExistingDnsRoutingGroups()
			Expect(err).NotTo(HaveOccurred())
			Expect(existingGroups["test-group"]).To(ConsistOf("example.com", "test.com"))
		})

		It("should add new domains to existing group", func() {
			server := NewMockRouterServer(WithVersion("5.0.1"))
			defer server.Close()
			SetupTestConfig(server.URL)
			defer CleanupTestConfig()
			Expect(Common.Auth()).To(Succeed())

			tmpDir := GinkgoT().TempDir()
			initialFile := filepath.Join(tmpDir, "initial.txt")
			Expect(os.WriteFile(initialFile, []byte("example.com\n"), 0644)).To(Succeed())

			Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{
				{Name: "test-group", DomainFile: []string{initialFile}, InterfaceID: "Wireguard0"},
			})).To(Succeed())

			existingGroups, err := DnsRouting.GetExistingDnsRoutingGroups()
			Expect(err).NotTo(HaveOccurred())
			Expect(existingGroups["test-group"]).To(ConsistOf("example.com"))

			updatedFile := filepath.Join(tmpDir, "updated.txt")
			Expect(os.WriteFile(updatedFile, []byte("example.com\ntest.com\n"), 0644)).To(Succeed())

			Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{
				{Name: "test-group", DomainFile: []string{updatedFile}, InterfaceID: "Wireguard0"},
			})).To(Succeed())

			existingGroups, err = DnsRouting.GetExistingDnsRoutingGroups()
			Expect(err).NotTo(HaveOccurred())
			Expect(existingGroups["test-group"]).To(ConsistOf("example.com", "test.com"))
		})

		It("should get existing groups via REST API", func() {
			groups := []MockDnsRoutingGroup{
				{Name: "group1", Domains: []string{"example.com", "test.com"}},
				{Name: "group2", Domains: []string{"google.com"}},
			}
			routes := []MockDnsProxyRoute{
				{GroupName: "group1", InterfaceID: "Wireguard0", Mode: "auto"},
			}

			server := NewMockRouterServer(WithDnsRoutingGroups(groups, routes))
			defer server.Close()
			SetupTestConfig(server.URL)
			defer CleanupTestConfig()
			Expect(Common.Auth()).To(Succeed())

			existingGroups, err := DnsRouting.GetExistingDnsRoutingGroups()
			Expect(err).NotTo(HaveOccurred())
			Expect(existingGroups).To(HaveLen(2))
			Expect(existingGroups["group1"]).To(ConsistOf("example.com", "test.com"))
			Expect(existingGroups["group2"]).To(ConsistOf("google.com"))
		})

		It("should clean up unwanted domains from groups", func() {
			server := NewMockRouterServer(WithVersion("5.0.1"))
			defer server.Close()
			SetupTestConfig(server.URL)
			defer CleanupTestConfig()
			Expect(Common.Auth()).To(Succeed())

			tmpDir := GinkgoT().TempDir()
			initialFile := filepath.Join(tmpDir, "initial.txt")
			Expect(os.WriteFile(initialFile, []byte("example.com\ntest.com\nold-domain.com\n"), 0644)).To(Succeed())

			Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{
				{Name: "test-group", DomainFile: []string{initialFile}, InterfaceID: "Wireguard0"},
			})).To(Succeed())

			existingGroups, err := DnsRouting.GetExistingDnsRoutingGroups()
			Expect(err).NotTo(HaveOccurred())
			Expect(existingGroups["test-group"]).To(ConsistOf("example.com", "test.com", "old-domain.com"))

			updatedFile := filepath.Join(tmpDir, "updated.txt")
			Expect(os.WriteFile(updatedFile, []byte("example.com\ntest.com\nnew-domain.com\n"), 0644)).To(Succeed())

			Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{
				{Name: "test-group", DomainFile: []string{updatedFile}, InterfaceID: "Wireguard0"},
			})).To(Succeed())

			existingGroups, err = DnsRouting.GetExistingDnsRoutingGroups()
			Expect(err).NotTo(HaveOccurred())
			Expect(existingGroups["test-group"]).To(ConsistOf("example.com", "test.com", "new-domain.com"))
			Expect(existingGroups["test-group"]).NotTo(ContainElement("old-domain.com"))
		})

		It("should get existing dns-proxy routes", func() {
			groups := []MockDnsRoutingGroup{
				{Name: "group1", Domains: []string{"example.com"}},
			}
			routes := []MockDnsProxyRoute{
				{GroupName: "group1", InterfaceID: "Wireguard0", Mode: "auto"},
				{GroupName: "group2", InterfaceID: "ISP", Mode: "auto"},
			}

			server := NewMockRouterServer(WithDnsRoutingGroups(groups, routes))
			defer server.Close()
			SetupTestConfig(server.URL)
			defer CleanupTestConfig()
			Expect(Common.Auth()).To(Succeed())

			existingRoutes, err := DnsRouting.GetExistingDnsProxyRoutes()
			Expect(err).NotTo(HaveOccurred())
			Expect(existingRoutes).To(HaveLen(2))
			Expect(existingRoutes["group1"]).To(Equal("Wireguard0"))
			Expect(existingRoutes["group2"]).To(Equal("ISP"))
		})

		It("should skip existing dns-proxy route", func() {
			groups := []MockDnsRoutingGroup{
				{Name: "test-group", Domains: []string{"example.com"}},
			}
			routes := []MockDnsProxyRoute{
				{GroupName: "test-group", InterfaceID: "Wireguard0", Mode: "auto"},
			}

			server := NewMockRouterServer(WithVersion("5.0.1"), WithDnsRoutingGroups(groups, routes))
			defer server.Close()
			SetupTestConfig(server.URL)
			defer CleanupTestConfig()
			Expect(Common.Auth()).To(Succeed())

			tmpDir := GinkgoT().TempDir()
			domainFile := filepath.Join(tmpDir, "domains.txt")
			Expect(os.WriteFile(domainFile, []byte("example.com\n"), 0644)).To(Succeed())

			Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{
				{Name: "test-group", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"},
			})).To(Succeed())

			existingRoutes, err := DnsRouting.GetExistingDnsProxyRoutes()
			Expect(err).NotTo(HaveOccurred())
			Expect(existingRoutes["test-group"]).To(Equal("Wireguard0"))
		})

		It("should update dns-proxy route when interface changes", func() {
			groups := []MockDnsRoutingGroup{
				{Name: "test-group", Domains: []string{"example.com"}},
			}
			routes := []MockDnsProxyRoute{
				{GroupName: "test-group", InterfaceID: "Wireguard0", Mode: "auto"},
			}

			server := NewMockRouterServer(WithVersion("5.0.1"), WithDnsRoutingGroups(groups, routes))
			defer server.Close()
			SetupTestConfig(server.URL)
			defer CleanupTestConfig()
			Expect(Common.Auth()).To(Succeed())

			tmpDir := GinkgoT().TempDir()
			domainFile := filepath.Join(tmpDir, "domains.txt")
			Expect(os.WriteFile(domainFile, []byte("example.com\n"), 0644)).To(Succeed())

			Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{
				{Name: "test-group", DomainFile: []string{domainFile}, InterfaceID: "ISP"},
			})).To(Succeed())

			existingRoutes, err := DnsRouting.GetExistingDnsProxyRoutes()
			Expect(err).NotTo(HaveOccurred())
			Expect(existingRoutes["test-group"]).To(Equal("ISP"))
		})
	})
})
