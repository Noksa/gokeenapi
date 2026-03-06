package cmd

import (
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/config"
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
})
