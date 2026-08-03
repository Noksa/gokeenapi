package cmd

import (
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteAllDnsRouting", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = gokeenrestapi.SetupMockRouterForTest(
			gokeenrestapi.WithVersion("5.0.1"),
			gokeenrestapi.WithDnsRoutingGroups(
				[]gokeenrestapi.MockDnsRoutingGroup{
					{Name: "social-media", Domains: []string{"facebook.com", "instagram.com"}},
					{Name: "streaming", Domains: []string{"youtube.com", "netflix.com"}},
				},
				[]gokeenrestapi.MockDnsProxyRoute{
					{GroupName: "social-media", InterfaceID: "Wireguard0", Mode: "auto"},
					{GroupName: "streaming", InterfaceID: "Wireguard0", Mode: "auto"},
				},
			),
		)
		Expect(gokeenrestapi.Common.Auth()).To(Succeed())
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes and flags", func() {
		cmd := newDeleteAllDnsRoutingCmd()

		Expect(cmd.Use).To(Equal(CmdDeleteAllDnsRouting))
		Expect(cmd.Aliases).To(Equal(AliasesDeleteAllDnsRouting))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.Long).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())

		forceFlag := cmd.Flags().Lookup("force")
		Expect(forceFlag).NotTo(BeNil())
		Expect(forceFlag.Value.Type()).To(Equal("bool"))
		Expect(forceFlag.DefValue).To(Equal("false"))
	})

	It("should delete all groups with force flag", func() {
		groupsBefore, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(groupsBefore).To(HaveLen(2))

		cmd := newDeleteAllDnsRoutingCmd()
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		groupsAfter, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(groupsAfter).To(BeEmpty())
	})

	It("should cancel when not forced and user declines", func() {
		cmd := newDeleteAllDnsRoutingCmd()
		err := cmd.RunE(cmd, []string{})
		// In non-interactive mode confirmAction returns an error reading stdin
		Expect(err).To(HaveOccurred())
	})

	It("should succeed when router has no DNS-routing groups", func() {
		cleanupMockRouter(server)
		server = gokeenrestapi.SetupMockRouterForTest(
			gokeenrestapi.WithVersion("5.0.1"),
		)
		Expect(gokeenrestapi.Common.Auth()).To(Succeed())

		cmd := newDeleteAllDnsRoutingCmd()
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should delete groups across different interfaces", func() {
		cleanupMockRouter(server)
		server = gokeenrestapi.SetupMockRouterForTest(
			gokeenrestapi.WithVersion("5.0.1"),
			gokeenrestapi.WithDnsRoutingGroups(
				[]gokeenrestapi.MockDnsRoutingGroup{
					{Name: "vpn-group", Domains: []string{"example.com"}},
					{Name: "isp-group", Domains: []string{"gov.ru"}},
				},
				[]gokeenrestapi.MockDnsProxyRoute{
					{GroupName: "vpn-group", InterfaceID: "Wireguard0", Mode: "auto"},
					{GroupName: "isp-group", InterfaceID: "ISP", Mode: "auto"},
				},
			),
		)
		Expect(gokeenrestapi.Common.Auth()).To(Succeed())

		cmd := newDeleteAllDnsRoutingCmd()
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		groupsAfter, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(groupsAfter).To(BeEmpty())
	})

	It("should fail on unsupported firmware version", func() {
		cleanupMockRouter(server)
		server = gokeenrestapi.SetupMockRouterForTest(
			gokeenrestapi.WithVersion("4.3.6"),
		)
		Expect(gokeenrestapi.Common.Auth()).To(Succeed())

		cmd := newDeleteAllDnsRoutingCmd()
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(HaveOccurred())
	})
})
