package cmd

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func writeDnsRoutingRootConfig(dir, routerURL, groupsYAML string) string {
	path := filepath.Join(dir, "router.yaml")
	content := fmt.Sprintf("keenetic:\n  url: %q\n  login: admin\n  password: password\ndataDir: %q\ndns:\n  routes:\n    groups:\n%s", routerURL, dir, groupsYAML)
	ExpectWithOffset(1, os.WriteFile(path, []byte(content), 0600)).To(Succeed())
	return path
}

func executeDnsRoutingRoot(command, configPath string, extraArgs ...string) error {
	root := NewRootCmd()
	root.SetArgs(append([]string{command, "--config", configPath}, extraArgs...))
	return root.Execute()
}

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

	It("splits and reconciles domains through repeated root-command runs", func() {
		tmpDir := GinkgoT().TempDir()
		var source strings.Builder
		for i := range 397 {
			fmt.Fprintf(&source, "d%04d.example.com\n", i)
		}
		domainFile := writeTempFile(tmpDir, "facebook.txt", source.String())
		groupsYAML := fmt.Sprintf("      - name: facebook\n        domain-file: [%q]\n        interfaceId: Wireguard0\n", domainFile)
		configPath := writeDnsRoutingRootConfig(tmpDir, server.URL, groupsYAML)

		Expect(executeDnsRoutingRoot("adnsr", configPath)).To(Succeed())
		existing, err := gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveLen(2))
		Expect(existing["facebook"]).To(HaveLen(300))
		Expect(existing["facebook-2"]).To(HaveLen(97))
		routes, err := gokeenrestapi.DnsRouting.GetExistingDnsProxyRoutes()
		Expect(err).NotTo(HaveOccurred())
		Expect(routes["facebook"]).To(Equal("Wireguard0"))
		Expect(routes["facebook-2"]).To(Equal("Wireguard0"))

		// A new domain before the old boundary must move domains between chunks.
		Expect(os.WriteFile(domainFile, []byte("a.example.com\n"+source.String()), 0644)).To(Succeed())
		Expect(executeDnsRoutingRoot("adnsr", configPath)).To(Succeed())
		allDomains := append(slices.Clone(existing["facebook"]), existing["facebook-2"]...)
		allDomains = append(allDomains, "a.example.com")
		slices.Sort(allDomains)
		existing, err = gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing["facebook"]).To(ConsistOf(allDomains[:300]))
		Expect(existing["facebook-2"]).To(ConsistOf(allDomains[300:]))
		Expect(executeDnsRoutingRoot("adnsr", configPath)).To(Succeed())

		var shrunk strings.Builder
		for i := range 300 {
			fmt.Fprintf(&shrunk, "d%04d.example.com\n", i)
		}
		Expect(os.WriteFile(domainFile, []byte(shrunk.String()), 0644)).To(Succeed())
		Expect(executeDnsRoutingRoot("adnsr", configPath)).To(Succeed())
		existing, err = gokeenrestapi.DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveLen(1))
		Expect(existing["facebook"]).To(HaveLen(300))
		routes, err = gokeenrestapi.DnsRouting.GetExistingDnsProxyRoutes()
		Expect(err).NotTo(HaveOccurred())
		Expect(routes).NotTo(HaveKey("facebook-2"))
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
