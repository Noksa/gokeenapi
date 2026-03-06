package cmd

import (
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ShowInterfaces", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes and flags", func() {
		cmd := newShowInterfacesCmd()

		Expect(cmd.Use).To(Equal(CmdShowInterfaces))
		Expect(cmd.Aliases).To(Equal(AliasesShowInterfaces))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())

		typeFlag := cmd.Flags().Lookup("type")
		Expect(typeFlag).NotTo(BeNil())
		Expect(typeFlag.Value.Type()).To(Equal("stringSlice"))
	})

	It("should execute and show interfaces", func() {
		cmd := newShowInterfacesCmd()
		output, err := captureOutput(cmd, []string{})

		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("Wireguard0"))
		Expect(output).To(ContainSubstring("ISP"))
	})

	It("should filter by type", func() {
		cmd := newShowInterfacesCmd()
		_ = cmd.Flags().Set("type", gokeenrestapi.InterfaceTypeWireguard)
		output, err := captureOutput(cmd, []string{})

		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("Wireguard0"))
	})
})
