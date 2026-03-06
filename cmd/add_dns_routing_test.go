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

var _ = Describe("AddDnsRouting", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter(gokeenrestapi.WithVersion("5.0.1"))
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes", func() {
		cmd := newAddDnsRoutingCmd()

		Expect(cmd.Use).To(Equal(CmdAddDnsRouting))
		Expect(cmd.Aliases).To(Equal(AliasesAddDnsRouting))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())
	})

	It("should execute with configured DNS routing groups", func() {
		tmpDir := GinkgoT().TempDir()

		socialMediaFile := filepath.Join(tmpDir, "social-media.txt")
		Expect(os.WriteFile(socialMediaFile, []byte("facebook.com\ninstagram.com\ntwitter.com\n"), 0644)).To(Succeed())

		streamingFile := filepath.Join(tmpDir, "streaming.txt")
		Expect(os.WriteFile(streamingFile, []byte("youtube.com\nnetflix.com\n8.8.8.8\n"), 0644)).To(Succeed())

		config.Cfg.DNS = config.DNS{
			Routes: config.DnsRoutes{
				Groups: []config.DnsRoutingGroup{
					{Name: "social-media", DomainFile: []string{socialMediaFile}, InterfaceID: "Wireguard0"},
					{Name: "streaming", DomainFile: []string{streamingFile}, InterfaceID: "ISP"},
				},
			},
		}

		cmd := newAddDnsRoutingCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should handle empty configuration", func() {
		config.Cfg.DNS = config.DNS{
			Routes: config.DnsRoutes{Groups: []config.DnsRoutingGroup{}},
		}

		cmd := newAddDnsRoutingCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	Context("validation errors", func() {
		It("should reject empty group name", func() {
			tmpDir := GinkgoT().TempDir()
			domainFile := writeTempFile(tmpDir, "domains.txt", "facebook.com\n")

			config.Cfg.DNS = config.DNS{
				Routes: config.DnsRoutes{
					Groups: []config.DnsRoutingGroup{
						{Name: "", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"},
					},
				},
			}

			cmd := newAddDnsRoutingCmd()
			err := cmd.RunE(cmd, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("group name cannot be empty"))
		})

		It("should reject whitespace-only group name", func() {
			tmpDir := GinkgoT().TempDir()
			domainFile := writeTempFile(tmpDir, "domains.txt", "facebook.com\n")

			config.Cfg.DNS = config.DNS{
				Routes: config.DnsRoutes{
					Groups: []config.DnsRoutingGroup{
						{Name: "   ", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"},
					},
				},
			}

			cmd := newAddDnsRoutingCmd()
			err := cmd.RunE(cmd, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("whitespace"))
		})

		It("should reject empty domain list", func() {
			config.Cfg.DNS = config.DNS{
				Routes: config.DnsRoutes{
					Groups: []config.DnsRoutingGroup{
						{Name: "social-media", DomainFile: []string{}, DomainURL: []string{}, InterfaceID: "Wireguard0"},
					},
				},
			}

			cmd := newAddDnsRoutingCmd()
			err := cmd.RunE(cmd, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must contain at least one domain-file or domain-url"))
		})

		It("should reject malformed domain", func() {
			tmpDir := GinkgoT().TempDir()
			domainFile := writeTempFile(tmpDir, "domains.txt", "invalid..domain\n")

			config.Cfg.DNS = config.DNS{
				Routes: config.DnsRoutes{
					Groups: []config.DnsRoutingGroup{
						{Name: "social-media", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"},
					},
				},
			}

			cmd := newAddDnsRoutingCmd()
			err := cmd.RunE(cmd, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid domain"))
		})

		It("should reject invalid IP", func() {
			tmpDir := GinkgoT().TempDir()
			domainFile := writeTempFile(tmpDir, "domains.txt", "999.999.999.999\n")

			config.Cfg.DNS = config.DNS{
				Routes: config.DnsRoutes{
					Groups: []config.DnsRoutingGroup{
						{Name: "streaming", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"},
					},
				},
			}

			cmd := newAddDnsRoutingCmd()
			err := cmd.RunE(cmd, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid domain or IP"))
		})

		It("should reject empty interface ID", func() {
			tmpDir := GinkgoT().TempDir()
			domainFile := writeTempFile(tmpDir, "domains.txt", "facebook.com\n")

			config.Cfg.DNS = config.DNS{
				Routes: config.DnsRoutes{
					Groups: []config.DnsRoutingGroup{
						{Name: "social-media", DomainFile: []string{domainFile}, InterfaceID: ""},
					},
				},
			}

			cmd := newAddDnsRoutingCmd()
			err := cmd.RunE(cmd, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("interface ID cannot be empty"))
		})
	})
})
