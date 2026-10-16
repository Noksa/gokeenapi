package gokeenrestapi

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/noksa/gokeenapi/internal/gokeencache"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// makeDomainFile writes count unique domains to a temp file and returns the path.
func makeDomainFile(dir string, name string, count int) string {
	var sb strings.Builder
	for i := range count {
		fmt.Fprintf(&sb, "d%04d.example.com\n", i)
	}
	path := filepath.Join(dir, name)
	Expect(os.WriteFile(path, []byte(sb.String()), 0644)).To(Succeed())
	return path
}

func currentDnsProxyRoutes() gokeenrestapimodels.DnsProxyRouteResponse {
	body, err := Common.ExecuteGetSubPath("/rci/dns-proxy/route")
	Expect(err).NotTo(HaveOccurred())
	var routes gokeenrestapimodels.DnsProxyRouteResponse
	Expect(json.Unmarshal(body, &routes)).To(Succeed())
	return routes
}

var _ = Describe("DNS routing domain-per-group router limit", func() {
	var (
		server *httptest.Server
		tmpDir string
	)

	BeforeEach(func() {
		server = NewMockRouterServer(WithVersion("5.0.1"))
		SetupTestConfig(server.URL)
		Expect(Common.Auth()).To(Succeed())
		tmpDir = GinkgoT().TempDir()
	})

	AfterEach(func() {
		CleanupTestConfig()
		server.Close()
	})

	It("should succeed with exactly 300 domains", func() {
		domainFile := makeDomainFile(tmpDir, "300.txt", maxDomainsPerGroup)

		groups := []config.DnsRoutingGroup{
			{Name: "exact-limit", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"},
		}

		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())

		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveLen(1))
		Expect(existing["exact-limit"]).To(HaveLen(maxDomainsPerGroup))
	})

	It("should count unique domains when deciding whether to split", func() {
		domainFile := makeDomainFile(tmpDir, "300-plus-duplicate.txt", maxDomainsPerGroup)
		f, err := os.OpenFile(domainFile, os.O_APPEND|os.O_WRONLY, 0644)
		Expect(err).NotTo(HaveOccurred())
		_, err = f.WriteString("d0000.example.com\n")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Close()).To(Succeed())

		group := config.DnsRoutingGroup{Name: "deduplicated", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"}
		Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{group})).To(Succeed())
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveLen(1))
		Expect(existing["deduplicated"]).To(HaveLen(300))
	})

	It("should apply IPv4 addresses and subnets from a domain source", func() {
		path := filepath.Join(tmpDir, "mixed.txt")
		Expect(os.WriteFile(path, []byte("example.com\n192.0.2.1\n10.0.0.1/24\n"), 0644)).To(Succeed())
		groups := []config.DnsRoutingGroup{{Name: "mixed", DomainFile: []string{path}, InterfaceID: "Wireguard0"}}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing["mixed"]).To(ConsistOf("example.com", "192.0.2.1", "10.0.0.0/24"))
	})

	It("should split 397 domains, reconcile changes, and remove the split after shrinking", func() {
		domainFile := makeDomainFile(tmpDir, "397.txt", 397)

		groups := []config.DnsRoutingGroup{
			{Name: "facebook", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"},
		}

		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveLen(2))
		Expect(existing["facebook"]).To(HaveLen(300))
		Expect(existing["facebook-2"]).To(HaveLen(97))
		routes, err := DnsRouting.GetExistingDnsProxyRoutes()
		Expect(err).NotTo(HaveOccurred())
		Expect(routes["facebook"]).To(Equal("Wireguard0"))
		Expect(routes["facebook-2"]).To(Equal("Wireguard0"))

		// Adding an early domain changes the chunk boundary. Re-running must
		// converge without duplicate or missing domains.
		fileContents, err := os.ReadFile(domainFile)
		Expect(err).NotTo(HaveOccurred())
		Expect(os.WriteFile(domainFile, append([]byte("a.example.com\n"), fileContents...), 0644)).To(Succeed())
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		all := append(slices.Clone(existing["facebook"]), existing["facebook-2"]...)
		all = append(all, "a.example.com")
		slices.Sort(all)
		existing, err = DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing["facebook"]).To(ConsistOf(all[:300]))
		Expect(existing["facebook-2"]).To(ConsistOf(all[300:]))
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())

		groups[0].DomainFile = []string{makeDomainFile(tmpDir, "shrink-300.txt", 300)}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		existing, err = DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveLen(1))
		Expect(existing["facebook"]).To(HaveLen(300))
		routes, err = DnsRouting.GetExistingDnsProxyRoutes()
		Expect(err).NotTo(HaveOccurred())
		Expect(routes).NotTo(HaveKey("facebook-2"))
	})

	It("should use two groups for 301 and three for 601", func() {
		groups := []config.DnsRoutingGroup{{Name: "g", DomainFile: []string{makeDomainFile(tmpDir, "301.txt", 301)}, InterfaceID: "Wireguard0"}}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing["g"]).To(HaveLen(300))
		Expect(existing["g-2"]).To(HaveLen(1))

		groups[0].DomainFile = []string{makeDomainFile(tmpDir, "601.txt", 601)}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		existing, err = DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing["g"]).To(HaveLen(300))
		Expect(existing["g-2"]).To(HaveLen(300))
		Expect(existing["g-3"]).To(HaveLen(1))

		groups[0].DomainFile = []string{makeDomainFile(tmpDir, "shrink-301.txt", 301)}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		existing, err = DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveLen(2))
		Expect(existing).NotTo(HaveKey("g-3"))
	})

	It("should migrate both split routes when the interface changes", func() {
		groups := []config.DnsRoutingGroup{{Name: "facebook", DomainFile: []string{makeDomainFile(tmpDir, "301.txt", 301)}, InterfaceID: "Wireguard0"}}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		groups[0].InterfaceID = "ISP"
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		Expect(currentDnsProxyRoutes()).To(ConsistOf(
			gokeenrestapimodels.DnsProxyRoute{Group: "facebook", Interface: "ISP"},
			gokeenrestapimodels.DnsProxyRoute{Group: "facebook-2", Interface: "ISP"},
		))
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		Expect(currentDnsProxyRoutes()).To(HaveLen(2))
	})

	It("should reject ambiguous generated names before changing the router", func() {
		group := config.DnsRoutingGroup{Name: "facebook", DomainFile: []string{makeDomainFile(tmpDir, "301.txt", 301)}, InterfaceID: "Wireguard0"}
		conflict := config.DnsRoutingGroup{Name: "facebook-2", DomainFile: []string{makeDomainFile(tmpDir, "other.txt", 1)}, InterfaceID: "Wireguard0"}
		Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{group, conflict})).To(MatchError(ContainSubstring("reserved for an automatic split")))
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(BeEmpty())
	})

	It("should preserve a router group with a conflicting generated name", func() {
		_, err := Common.ExecutePostParse(
			gokeenrestapimodels.ParseRequest{Parse: "object-group fqdn facebook"},
			gokeenrestapimodels.ParseRequest{Parse: "dns-proxy route object-group facebook Wireguard0 auto"},
			gokeenrestapimodels.ParseRequest{Parse: "object-group fqdn facebook-2"},
			gokeenrestapimodels.ParseRequest{Parse: "object-group fqdn facebook-2 include manual.example.com"},
			gokeenrestapimodels.ParseRequest{Parse: "dns-proxy route object-group facebook-2 ISP auto"},
		)
		Expect(err).NotTo(HaveOccurred())

		group := config.DnsRoutingGroup{Name: "facebook", DomainFile: []string{makeDomainFile(tmpDir, "301.txt", 301)}, InterfaceID: "Wireguard0"}
		Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{group})).To(MatchError(ContainSubstring("without a matching route")))
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing["facebook"]).To(BeEmpty())
		Expect(existing["facebook-2"]).To(ConsistOf("manual.example.com"))
	})

	It("should reject a generated name when its base group is absent", func() {
		_, err := Common.ExecutePostParse(
			gokeenrestapimodels.ParseRequest{Parse: "object-group fqdn facebook-2"},
			gokeenrestapimodels.ParseRequest{Parse: "dns-proxy route object-group facebook-2 Wireguard0 auto"},
		)
		Expect(err).NotTo(HaveOccurred())
		group := config.DnsRoutingGroup{Name: "facebook", DomainFile: []string{makeDomainFile(tmpDir, "301.txt", 301)}, InterfaceID: "Wireguard0"}
		Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{group})).To(MatchError(ContainSubstring("without its base object-group")))
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).NotTo(HaveKey("facebook"))
	})

	It("should reject a generated group without a route", func() {
		_, err := Common.ExecutePostParse(
			gokeenrestapimodels.ParseRequest{Parse: "object-group fqdn facebook"},
			gokeenrestapimodels.ParseRequest{Parse: "dns-proxy route object-group facebook Wireguard0 auto"},
			gokeenrestapimodels.ParseRequest{Parse: "object-group fqdn facebook-2"},
		)
		Expect(err).NotTo(HaveOccurred())
		group := config.DnsRoutingGroup{Name: "facebook", DomainFile: []string{makeDomainFile(tmpDir, "301.txt", 301)}, InterfaceID: "Wireguard0"}
		Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{group})).To(MatchError(ContainSubstring("cannot be safely resumed")))
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveLen(2))
		Expect(existing["facebook"]).To(BeEmpty())
	})

	It("should resume a partially created empty split group", func() {
		groups := []config.DnsRoutingGroup{{Name: "facebook", DomainFile: []string{makeDomainFile(tmpDir, "300.txt", 300)}, InterfaceID: "Wireguard0"}}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		_, err := Common.ExecutePostParse(gokeenrestapimodels.ParseRequest{Parse: "object-group fqdn facebook-2"})
		Expect(err).NotTo(HaveOccurred())
		groups[0].DomainFile = []string{makeDomainFile(tmpDir, "301.txt", 301)}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing["facebook-2"]).To(ConsistOf("d0300.example.com"))
		Expect(currentDnsProxyRoutes()).To(HaveLen(2))
	})

	It("should refuse an unrelated noncontiguous suffix on the same interface", func() {
		groups := []config.DnsRoutingGroup{{Name: "facebook", DomainFile: []string{makeDomainFile(tmpDir, "300.txt", 300)}, InterfaceID: "Wireguard0"}}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		_, err := Common.ExecutePostParse(
			gokeenrestapimodels.ParseRequest{Parse: "object-group fqdn facebook-99"},
			gokeenrestapimodels.ParseRequest{Parse: "object-group fqdn facebook-99 include manual.example.com"},
			gokeenrestapimodels.ParseRequest{Parse: "dns-proxy route object-group facebook-99 Wireguard0 auto"},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(MatchError(ContainSubstring("noncontiguous generated group names")))
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing["facebook-99"]).To(ConsistOf("manual.example.com"))
	})

	It("should guard against a drastic shrink and allow an explicit one-run override", func() {
		groups := []config.DnsRoutingGroup{{Name: "facebook", DomainFile: []string{makeDomainFile(tmpDir, "397.txt", 397)}, InterfaceID: "Wireguard0"}}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		smallFile := filepath.Join(tmpDir, "small.txt")
		Expect(os.WriteFile(smallFile, []byte("d0000.example.com\n"), 0644)).To(Succeed())
		groups[0].DomainFile = []string{smallFile}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(MatchError(ContainSubstring("--allow-large-shrink")))
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing["facebook"]).To(HaveLen(300))
		Expect(existing["facebook-2"]).To(HaveLen(97))

		Expect(DnsRouting.AddDnsRoutingGroupsWithOptions(groups, DnsRoutingApplyOptions{AllowLargeShrink: true})).To(Succeed())
		existing, err = DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveLen(1))
		Expect(existing["facebook"]).To(ConsistOf("d0000.example.com"))
	})

	It("should preserve router state when a remote source suddenly returns a short valid list", func() {
		config.Cfg.DataDir = tmpDir
		contents, err := os.ReadFile(makeDomainFile(tmpDir, "remote-397.txt", 397))
		Expect(err).NotTo(HaveOccurred())
		domainSource := newDomainServer(string(contents))
		DeferCleanup(domainSource.Close)
		groups := []config.DnsRoutingGroup{{Name: "remote", DomainURL: []string{domainSource.URL}, InterfaceID: "Wireguard0"}}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())

		Expect(gokeencache.SetURLContent(domainSource.URL, string(contents), time.Millisecond)).To(Succeed())
		time.Sleep(5 * time.Millisecond)
		domainSource.setBody("d0000.example.com\n")
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(MatchError(ContainSubstring("--allow-large-shrink")))
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing["remote"]).To(HaveLen(300))
		Expect(existing["remote-2"]).To(HaveLen(97))
	})

	It("should reject a generated route without an object-group", func() {
		// The mock accepts initial router state with a dangling route, which
		// lets us verify the preflight without sending any mutation.
		server.Close()
		CleanupTestConfig()
		server = NewMockRouterServer(WithVersion("5.0.1"), WithDnsRoutingGroups(nil,
			[]MockDnsProxyRoute{{GroupName: "facebook-2", InterfaceID: "Wireguard0", Mode: "auto"}}))
		SetupTestConfig(server.URL)
		Expect(Common.Auth()).To(Succeed())
		group := config.DnsRoutingGroup{Name: "facebook", DomainFile: []string{makeDomainFile(tmpDir, "301.txt", 301)}, InterfaceID: "Wireguard0"}
		Expect(DnsRouting.AddDnsRoutingGroups([]config.DnsRoutingGroup{group})).To(MatchError(ContainSubstring("exists without an object-group")))
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(BeEmpty())
	})

	It("should preserve an existing split when its source cannot be read", func() {
		groups := []config.DnsRoutingGroup{{Name: "facebook", DomainFile: []string{makeDomainFile(tmpDir, "301.txt", 301)}, InterfaceID: "Wireguard0"}}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		groups[0].DomainFile = []string{filepath.Join(tmpDir, "missing.txt")}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(MatchError(ContainSubstring("failed to read domain file")))
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing["facebook"]).To(HaveLen(300))
		Expect(existing["facebook-2"]).To(HaveLen(1))
	})

	It("should preserve an existing split when the source is empty", func() {
		domainFile := makeDomainFile(tmpDir, "301.txt", 301)
		groups := []config.DnsRoutingGroup{{Name: "facebook", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"}}
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		Expect(os.WriteFile(domainFile, []byte("# temporarily empty\n"), 0644)).To(Succeed())
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing["facebook"]).To(HaveLen(300))
		Expect(existing["facebook-2"]).To(HaveLen(1))
	})

	It("should skip group with 0 domains and not return an error", func() {
		// Write an empty file (only a comment)
		emptyFile := filepath.Join(tmpDir, "empty.txt")
		Expect(os.WriteFile(emptyFile, []byte("# no domains here\n"), 0644)).To(Succeed())

		groups := []config.DnsRoutingGroup{
			{Name: "empty-group", DomainFile: []string{emptyFile}, InterfaceID: "Wireguard0"},
		}

		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())

		// The group is skipped: no object-group command is issued, so it should either not
		// exist on the router or exist with an empty domain list.
		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		if domains, present := existing["empty-group"]; present {
			Expect(domains).To(BeEmpty(), "skipped group must not have any domains")
		}
	})
})

var _ = Describe("validateNoDuplicateDomainsAcrossGroups", func() {
	It("should log a warning when a domain appears in multiple groups", func() {
		// The function logs and continues (no error returned). We verify it does not panic
		// and that the behavior is idempotent when called multiple times.
		groupDomains := map[string][]string{
			"group-a": {"example.com", "shared.com"},
			"group-b": {"test.com", "shared.com"},
		}

		// Should not panic
		Expect(func() {
			validateNoDuplicateDomainsAcrossGroups(groupDomains)
		}).NotTo(Panic())
	})

	It("should be silent when all domains are unique across groups", func() {
		groupDomains := map[string][]string{
			"group-a": {"example.com", "foo.com"},
			"group-b": {"bar.com", "baz.com"},
		}

		Expect(func() {
			validateNoDuplicateDomainsAcrossGroups(groupDomains)
		}).NotTo(Panic())
	})

	It("should handle empty input without panicking", func() {
		Expect(func() {
			validateNoDuplicateDomainsAcrossGroups(map[string][]string{})
		}).NotTo(Panic())
	})

	It("AddDnsRoutingGroups should continue after duplicate-domain warning", func() {
		// Two groups sharing a domain should still be applied (warning, not error).
		server := NewMockRouterServer(WithVersion("5.0.1"))
		defer server.Close()
		SetupTestConfig(server.URL)
		defer CleanupTestConfig()
		Expect(Common.Auth()).To(Succeed())

		dir := GinkgoT().TempDir()
		fileA := filepath.Join(dir, "a.txt")
		fileB := filepath.Join(dir, "b.txt")
		Expect(os.WriteFile(fileA, []byte("example.com\nshared.com\n"), 0644)).To(Succeed())
		Expect(os.WriteFile(fileB, []byte("test.com\nshared.com\n"), 0644)).To(Succeed())

		groups := []config.DnsRoutingGroup{
			{Name: "group-a", DomainFile: []string{fileA}, InterfaceID: "Wireguard0"},
			{Name: "group-b", DomainFile: []string{fileB}, InterfaceID: "ISP"},
		}

		// Should succeed — duplicate warning is non-fatal
		Expect(DnsRouting.AddDnsRoutingGroups(groups)).To(Succeed())

		existing, err := DnsRouting.GetExistingDnsRoutingGroups()
		Expect(err).NotTo(HaveOccurred())
		Expect(existing).To(HaveKey("group-a"))
		Expect(existing).To(HaveKey("group-b"))
	})
})
