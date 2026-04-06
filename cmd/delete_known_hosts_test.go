package cmd

import (
	"net/http/httptest"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteKnownHosts", func() {
	var server *httptest.Server

	BeforeEach(func() {
		server = setupMockRouter()
	})

	AfterEach(func() {
		cleanupMockRouter(server)
	})

	It("should create command with correct attributes and flags", func() {
		cmd := newDeleteKnownHostsCmd()

		Expect(cmd.Use).To(Equal(CmdDeleteKnownHosts))
		Expect(cmd.Aliases).To(Equal(AliasesDeleteKnownHosts))
		Expect(cmd.Short).NotTo(BeEmpty())
		Expect(cmd.RunE).NotTo(BeNil())

		Expect(cmd.Flags().Lookup("name-pattern")).NotTo(BeNil())
		Expect(cmd.Flags().Lookup("mac-pattern")).NotTo(BeNil())

		forceFlag := cmd.Flags().Lookup("force")
		Expect(forceFlag).NotTo(BeNil())
		Expect(forceFlag.Value.Type()).To(Equal("bool"))
	})

	It("should fail when no pattern is specified", func() {
		cmd := newDeleteKnownHostsCmd()

		err := cmd.RunE(cmd, []string{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exactly one of --name-pattern or --mac-pattern must be specified"))
	})

	It("should fail when both patterns are specified", func() {
		cmd := newDeleteKnownHostsCmd()
		_ = cmd.Flags().Set("name-pattern", "test")
		_ = cmd.Flags().Set("mac-pattern", "aa:bb")

		err := cmd.RunE(cmd, []string{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("exactly one of --name-pattern or --mac-pattern must be specified"))
	})

	It("should fail with invalid regex", func() {
		cmd := newDeleteKnownHostsCmd()
		_ = cmd.Flags().Set("name-pattern", "[invalid")

		Expect(cmd.RunE(cmd, []string{})).To(HaveOccurred())
	})

	It("should delete hosts matching name pattern and verify state", func() {
		// Mock has: test-device-1 (aa:bb:cc:dd:ee:ff), test-device-2 (11:22:33:44:55:66)
		hotspotBefore, err := gokeenrestapi.Ip.GetAllHotspots()
		Expect(err).NotTo(HaveOccurred())
		Expect(hotspotBefore.Host).To(HaveLen(2))

		cmd := newDeleteKnownHostsCmd()
		_ = cmd.Flags().Set("name-pattern", "test-device-1")
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		hotspotAfter, err := gokeenrestapi.Ip.GetAllHotspots()
		Expect(err).NotTo(HaveOccurred())
		Expect(hotspotAfter.Host).To(HaveLen(1))
		Expect(hotspotAfter.Host[0].Name).To(Equal("test-device-2"))
	})

	It("should delete hosts matching mac pattern and verify state", func() {
		cmd := newDeleteKnownHostsCmd()
		_ = cmd.Flags().Set("mac-pattern", "^11:22")
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		hotspotAfter, err := gokeenrestapi.Ip.GetAllHotspots()
		Expect(err).NotTo(HaveOccurred())
		Expect(hotspotAfter.Host).To(HaveLen(1))
		Expect(hotspotAfter.Host[0].Mac).To(Equal("aa:bb:cc:dd:ee:ff"))
	})

	It("should delete all hosts matching wildcard pattern", func() {
		cmd := newDeleteKnownHostsCmd()
		_ = cmd.Flags().Set("name-pattern", "test-device-.*")
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		hotspotAfter, err := gokeenrestapi.Ip.GetAllHotspots()
		Expect(err).NotTo(HaveOccurred())
		Expect(hotspotAfter.Host).To(BeEmpty())
	})

	It("should handle no matching hosts gracefully", func() {
		cmd := newDeleteKnownHostsCmd()
		_ = cmd.Flags().Set("name-pattern", "nonexistent")
		_ = cmd.Flags().Set("force", "true")
		Expect(cmd.RunE(cmd, []string{})).To(Succeed())

		// All hosts should remain
		hotspot, err := gokeenrestapi.Ip.GetAllHotspots()
		Expect(err).NotTo(HaveOccurred())
		Expect(hotspot.Host).To(HaveLen(2))
	})
})
