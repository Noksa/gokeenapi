package cmd

import (
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteDnsRecords", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes and flags", func() {
		cmd := newDeleteDnsRecordsCmd()

		Expect(cmd.Use).To(Equal(CmdDeleteDnsRecords))
		Expect(cmd.Aliases).To(Equal(AliasesDeleteDnsRecords))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())

		forceFlag := cmd.Flags().Lookup("force")
		Expect(forceFlag).NotTo(BeNil())
		Expect(forceFlag.Value.Type()).To(Equal("bool"))
	})

	It("should execute with force flag", func() {
		config.Cfg.DNS = config.DNS{
			Records: []config.DnsRecord{
				{Domain: "test.local", IP: []string{"192.168.1.100"}},
			},
		}

		cmd := newDeleteDnsRecordsCmd()
		_ = cmd.Flags().Set("force", "true")

		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should handle empty records", func() {
		config.Cfg.DNS = config.DNS{Records: []config.DnsRecord{}}

		cmd := newDeleteDnsRecordsCmd()
		_ = cmd.Flags().Set("force", "true")

		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should actually remove matching DNS records from router", func() {
		// Mock has default records: example.com -> 1.2.3.4, test.local -> 192.168.1.50
		runningBefore, err := gokeenrestapi.Common.ShowRunningConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(runningBefore.Message).To(ContainElement("ip host example.com 1.2.3.4"))

		config.Cfg.DNS = config.DNS{
			Records: []config.DnsRecord{
				{Domain: "example.com", IP: []string{"1.2.3.4"}},
			},
		}

		cmd := newDeleteDnsRecordsCmd()
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		runningAfter, err := gokeenrestapi.Common.ShowRunningConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(runningAfter.Message).NotTo(ContainElement("ip host example.com 1.2.3.4"))
		// test.local should remain
		Expect(runningAfter.Message).To(ContainElement("ip host test.local 192.168.1.50"))
	})

	It("should skip records not present on router", func() {
		config.Cfg.DNS = config.DNS{
			Records: []config.DnsRecord{
				{Domain: "nonexistent.com", IP: []string{"9.9.9.9"}},
			},
		}

		cmd := newDeleteDnsRecordsCmd()
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		// All original records should remain
		running, err := gokeenrestapi.Common.ShowRunningConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(running.Message).To(ContainElement("ip host example.com 1.2.3.4"))
		Expect(running.Message).To(ContainElement("ip host test.local 192.168.1.50"))
	})
})
