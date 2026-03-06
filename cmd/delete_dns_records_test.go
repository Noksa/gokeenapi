package cmd

import (
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/config"
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
})
