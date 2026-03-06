package cmd

import (
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func setupDnsRoutingMockRouter() *httptest.Server {
	server := gokeenrestapi.SetupMockRouterForTest(
		gokeenrestapi.WithVersion("5.0.1"),
		gokeenrestapi.WithDnsRoutingGroups(
			[]gokeenrestapi.MockDnsRoutingGroup{
				{Name: "social-media", Domains: []string{"facebook.com", "instagram.com", "twitter.com"}},
				{Name: "streaming", Domains: []string{"youtube.com", "netflix.com", "8.8.8.8"}},
			},
			[]gokeenrestapi.MockDnsProxyRoute{
				{GroupName: "social-media", InterfaceID: "Wireguard0", Mode: "auto"},
				{GroupName: "streaming", InterfaceID: "ISP", Mode: "auto"},
			},
		),
	)
	Expect(gokeenrestapi.Common.Auth()).To(Succeed())
	return server
}

var _ = Describe("DeleteDnsRouting", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupDnsRoutingMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes and flags", func() {
		cmd := newDeleteDnsRoutingCmd()

		Expect(cmd.Use).To(Equal(CmdDeleteDnsRouting))
		Expect(cmd.Aliases).To(Equal(AliasesDeleteDnsRouting))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())

		forceFlag := cmd.Flags().Lookup("force")
		Expect(forceFlag).NotTo(BeNil())
		Expect(forceFlag.Value.Type()).To(Equal("bool"))
		Expect(forceFlag.DefValue).To(Equal("false"))

		interfaceIdFlag := cmd.Flags().Lookup("interface-id")
		Expect(interfaceIdFlag).NotTo(BeNil())
		Expect(interfaceIdFlag.Value.Type()).To(Equal("string"))
		Expect(interfaceIdFlag.DefValue).To(Equal(""))
	})

	Context("with force flag", func() {
		It("should delete a single group", func() {
			tmpDir := GinkgoT().TempDir()
			domainFile := filepath.Join(tmpDir, "social-media.txt")
			Expect(os.WriteFile(domainFile, []byte("facebook.com\ninstagram.com\ntwitter.com\n"), 0644)).To(Succeed())

			config.Cfg.DNS = config.DNS{
				Routes: config.DnsRoutes{
					Groups: []config.DnsRoutingGroup{
						{Name: "social-media", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"},
					},
				},
			}

			cmd := newDeleteDnsRoutingCmd()
			_ = cmd.Flags().Set("force", "true")
			Expect(cmd.RunE(cmd, []string{})).To(Succeed())

			groups, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
			Expect(err).NotTo(HaveOccurred())
			Expect(groups).NotTo(HaveKey("social-media"))
			Expect(groups).To(HaveKey("streaming"))
		})

		It("should delete multiple groups", func() {
			tmpDir := GinkgoT().TempDir()
			socialFile := writeTempFile(tmpDir, "social-media.txt", "facebook.com\ninstagram.com\ntwitter.com\n")
			streamingFile := writeTempFile(tmpDir, "streaming.txt", "youtube.com\nnetflix.com\n8.8.8.8\n")

			config.Cfg.DNS = config.DNS{
				Routes: config.DnsRoutes{
					Groups: []config.DnsRoutingGroup{
						{Name: "social-media", DomainFile: []string{socialFile}, InterfaceID: "Wireguard0"},
						{Name: "streaming", DomainFile: []string{streamingFile}, InterfaceID: "ISP"},
					},
				},
			}

			cmd := newDeleteDnsRoutingCmd()
			_ = cmd.Flags().Set("force", "true")
			Expect(cmd.RunE(cmd, []string{})).To(Succeed())

			groupsAfter, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
			Expect(err).NotTo(HaveOccurred())
			Expect(groupsAfter).To(BeEmpty())
		})
	})

	It("should handle no groups on router", func() {
		cleanupMockRouter(server)
		server = setupMockRouter(gokeenrestapi.WithVersion("5.0.1"))

		cmd := newDeleteDnsRoutingCmd()
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should verify groups exist before and after deletion", func() {
		tmpDir := GinkgoT().TempDir()
		socialFile := writeTempFile(tmpDir, "social-media.txt", "facebook.com\ninstagram.com\ntwitter.com\n")
		streamingFile := writeTempFile(tmpDir, "streaming.txt", "youtube.com\nnetflix.com\n8.8.8.8\n")

		config.Cfg.DNS = config.DNS{
			Routes: config.DnsRoutes{
				Groups: []config.DnsRoutingGroup{
					{Name: "social-media", DomainFile: []string{socialFile}, InterfaceID: "Wireguard0"},
					{Name: "streaming", DomainFile: []string{streamingFile}, InterfaceID: "ISP"},
				},
			},
		}

		groupsBefore, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(groupsBefore).NotTo(BeEmpty())

		cmd := newDeleteDnsRoutingCmd()
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		groupsAfter, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(groupsAfter).To(BeEmpty())
	})

	It("should handle partial match with non-existent group", func() {
		tmpDir := GinkgoT().TempDir()
		socialFile := writeTempFile(tmpDir, "social-media.txt", "facebook.com\ninstagram.com\ntwitter.com\n")
		nonExistentFile := writeTempFile(tmpDir, "non-existent.txt", "example.com\n")

		config.Cfg.DNS = config.DNS{
			Routes: config.DnsRoutes{
				Groups: []config.DnsRoutingGroup{
					{Name: "social-media", DomainFile: []string{socialFile}, InterfaceID: "Wireguard0"},
					{Name: "non-existent-group", DomainFile: []string{nonExistentFile}, InterfaceID: "ISP"},
				},
			},
		}

		cmd := newDeleteDnsRoutingCmd()
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should delete by interface-id", func() {
		config.Cfg.DNS = config.DNS{
			Routes: config.DnsRoutes{Groups: []config.DnsRoutingGroup{}},
		}

		cmd := newDeleteDnsRoutingCmd()
		_ = cmd.Flags().Set("force", "true")
		_ = cmd.Flags().Set("interface-id", "Wireguard0")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		groups, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(groups).NotTo(HaveKey("social-media"))
		Expect(groups).To(HaveKey("streaming"))
	})
})
