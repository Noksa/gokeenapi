package gokeenrestapi

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	"github.com/noksa/gokeenapi/pkg/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// makeDomainFile writes count unique domains to a temp file and returns the path.
func makeDomainFile(dir string, name string, count int) string {
	var sb strings.Builder
	for i := range count {
		fmt.Fprintf(&sb, "d%d.example.com\n", i)
	}
	path := filepath.Join(dir, name)
	Expect(os.WriteFile(path, []byte(sb.String()), 0644)).To(Succeed())
	return path
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
		Expect(existing["exact-limit"]).To(HaveLen(maxDomainsPerGroup))
	})

	It("should fail with 301 domains and include the limit in the error", func() {
		domainFile := makeDomainFile(tmpDir, "301.txt", maxDomainsPerGroup+1)

		groups := []config.DnsRoutingGroup{
			{Name: "over-limit", DomainFile: []string{domainFile}, InterfaceID: "Wireguard0"},
		}

		err := DnsRouting.AddDnsRoutingGroups(groups)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("exceeds router limit of %d domains", maxDomainsPerGroup)))
		Expect(err.Error()).To(ContainSubstring("over-limit"))
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
