package gokeenrestapi

import (
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteDnsRoutingGroups", func() {
	var server *httptest.Server

	makeFile := func(domains ...string) string {
		dir := GinkgoT().TempDir()
		content := ""
		for _, d := range domains {
			content += d + "\n"
		}
		p := filepath.Join(dir, "domains.txt")
		Expect(os.WriteFile(p, []byte(content), 0644)).To(Succeed())
		return p
	}

	BeforeEach(func() {
		server = NewMockRouterServer(WithVersion("5.0.1"))
		SetupTestConfig(server.URL)
		Expect(Common.Auth()).To(Succeed())
	})

	AfterEach(func() {
		CleanupTestConfig()
		server.Close()
	})

	It("should be a no-op when groups slice is empty", func() {
		Expect(DnsRouting.DeleteDnsRoutingGroups([]config.DnsRoutingGroup{})).To(Succeed())
	})

	It("should delete an existing group and its dns-proxy route", func() {
		// First add a group so there is something to delete
		domainFile := makeFile("example.com", "test.com")
		groups := []config.DnsRoutingGroup{
			{Name: "to-delete", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"},
		}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())

		// Verify it exists
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveKey("to-delete"))

		// Delete it
		Expect(DnsRouting.DeleteDnsRoutingGroups([]config.DnsRoutingGroup{
			{Name: "to-delete", InterfaceID: "Wireguard0"},
		})).To(Succeed())

		// Verify it is gone
		existing, err = DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).NotTo(HaveKey("to-delete"))
	})

	It("should delete multiple groups in a single call", func() {
		file1 := makeFile("alpha.com")
		file2 := makeFile("beta.com")

		Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{
			{Name: "group-alpha", DomainFile: []string{file1}, InterfaceID: "Wireguard0"},
			{Name: "group-beta", DomainFile: []string{file2}, InterfaceID: "ISP"},
		})).To(Succeed())

		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveKey("group-alpha"))
		Expect(existing).To(HaveKey("group-beta"))

		Expect(DnsRouting.DeleteDnsRoutingGroups([]config.DnsRoutingGroup{
			{Name: "group-alpha", InterfaceID: "Wireguard0"},
			{Name: "group-beta", InterfaceID: "ISP"},
		})).To(Succeed())

		existing, err = DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).NotTo(HaveKey("group-alpha"))
		Expect(existing).NotTo(HaveKey("group-beta"))
	})

	It("should succeed (idempotent) when deleting a non-existent group", func() {
		// Deleting a group that was never created should not error
		Expect(DnsRouting.DeleteDnsRoutingGroups([]config.DnsRoutingGroup{
			{Name: "never-existed", InterfaceID: "Wireguard0"},
		})).To(Succeed())
	})

	It("should also remove the dns-proxy route", func() {
		domainFile := makeFile("foo.com")
		Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{
			{Name: "route-group", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"},
		})).To(Succeed())

		// Verify route exists before deletion
		routes, err := DnsRouting.GetExistingDnsProxyRoutes()
		Expect(err).NotTo(HaveOccurred())
		Expect(routes).To(HaveKey("route-group"))

		Expect(DnsRouting.DeleteDnsRoutingGroups([]config.DnsRoutingGroup{
			{Name: "route-group", InterfaceID: "Wireguard0"},
		})).To(Succeed())

		routes, err = DnsRouting.GetExistingDnsProxyRoutes()
		Expect(err).NotTo(HaveOccurred())
		Expect(routes).NotTo(HaveKey("route-group"))
	})

	It("should fail on firmware older than 5.0.1", func() {
		// Restart server with old firmware
		server.Close()
		CleanupTestConfig()
		server = NewMockRouterServer(WithVersion("4.3.6.3"))
		SetupTestConfig(server.URL)
		Expect(Common.Auth()).To(Succeed())

		err := DnsRouting.DeleteDnsRoutingGroups([]config.DnsRoutingGroup{
			{Name: "any-group", InterfaceID: "Wireguard0"},
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("DNS-routing requires Keenetic firmware"))
	})

	It("should leave unrelated groups untouched", func() {
		file1 := makeFile("keep.com")
		file2 := makeFile("remove.com")

		Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{
			{Name: "keep-group", DomainFile: []string{file1}, InterfaceID: "Wireguard0"},
			{Name: "remove-group", DomainFile: []string{file2}, InterfaceID: "ISP"},
		})).To(Succeed())

		Expect(DnsRouting.DeleteDnsRoutingGroups([]config.DnsRoutingGroup{
			{Name: "remove-group", InterfaceID: "ISP"},
		})).To(Succeed())

		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveKey("keep-group"))
		Expect(existing).NotTo(HaveKey("remove-group"))
	})
})
