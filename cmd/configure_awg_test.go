package cmd

import (
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConfigureAwg", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes and flags", func() {
		cmd := newUpdateAwgCmd()

		Expect(cmd.Use).To(Equal(CmdUpdateAwg))
		Expect(cmd.Aliases).To(Equal(AliasesUpdateAwg))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("conf-file")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("interface-id")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("dry-run")).NotTo(BeNil())
	})

	It("should fail when conf-file is missing", func() {
		cmd := newUpdateAwgCmd()
		_ = cmd.Flags().Set("interface-id", "Wireguard0")

		err := cmd.RunE(cmd, []string{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("--conf-file flag is required"))
	})

	It("should fail when interface-id is missing", func() {
		cmd := newUpdateAwgCmd()
		_ = cmd.Flags().Set("conf-file", "/tmp/test.conf")

		err := cmd.RunE(cmd, []string{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("--interface-id flag is required"))
	})

	It("should support dry-run mode", func() {
		confContent := `[Interface]
PrivateKey = abc
Address = 10.0.0.2/24
Jc = 99
Jmin = 5
Jmax = 95
S1 = 10
S2 = 20
H1 = 1
H2 = 2
H3 = 3
H4 = 4

[Peer]
PublicKey = gN65BkIKy1eCE9pP1wdc8ROUunkiVXrBvGAKBEKdOQI=
Endpoint = example.com:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25`

		tmpDir := GinkgoT().TempDir()
		confPath := tmpDir + "/dry.conf"
		Expect(os.WriteFile(confPath, []byte(confContent), 0644)).To(Succeed())

		cmd := newUpdateAwgCmd()
		_ = cmd.Flags().Set("conf-file", confPath)
		_ = cmd.Flags().Set("interface-id", "Wireguard0")
		_ = cmd.Flags().Set("dry-run", "true")

		err := cmd.RunE(cmd, []string{})
		Expect(err).NotTo(HaveOccurred())
	})
})
