package cmd

import (
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddDnsRecords", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes", func() {
		cmd := newAddDnsRecordsCmd()

		Expect(cmd.Use).To(Equal(CmdAddDnsRecords))
		Expect(cmd.Aliases).To(Equal(AliasesAddDnsRecords))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())
	})

	It("should execute with configured DNS records", func() {
		config.Cfg.DNS = config.DNS{
			Records: []config.DnsRecord{
				{Domain: "test.local", IP: []string{"192.168.1.100"}},
			},
		}

		cmd := newAddDnsRecordsCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should handle empty records config", func() {
		config.Cfg.DNS = config.DNS{Records: []config.DnsRecord{}}

		cmd := newAddDnsRecordsCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())
	})

	It("should add new DNS records and verify in running config", func() {
		config.Cfg.DNS = config.DNS{
			Records: []config.DnsRecord{
				{Domain: "newdomain.com", IP: []string{"5.6.7.8"}},
			},
		}

		cmd := newAddDnsRecordsCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		running, err := gokeenrestapi.Common.ShowRunningConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(running.Message).To(ContainElement("ip host newdomain.com 5.6.7.8"))
	})

	It("should add multiple IPs for the same domain", func() {
		config.Cfg.DNS = config.DNS{
			Records: []config.DnsRecord{
				{Domain: "multi.com", IP: []string{"1.1.1.1", "2.2.2.2"}},
			},
		}

		cmd := newAddDnsRecordsCmd()
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		running, err := gokeenrestapi.Common.ShowRunningConfig()
		Expect(err).NotTo(HaveOccurred())
		Expect(running.Message).To(ContainElement("ip host multi.com 2.2.2.2"))
	})
})
