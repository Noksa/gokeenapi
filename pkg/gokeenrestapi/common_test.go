package gokeenrestapi

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CheckRouterMode", func() {
	AfterEach(func() {
		CleanupTestConfig()
	})

	It("should succeed in router mode", func() {
		server := SetupMockRouterForTest(WithSystemMode(MockSystemMode{Active: "router", Selected: "router"}))
		DeferCleanup(server.Close)

		active, selected, err := Common.CheckRouterMode()
		Expect(err).NotTo(HaveOccurred())
		Expect(active).To(Equal("router"))
		Expect(selected).To(Equal("router"))
	})

	It("should fail in extender mode", func() {
		server := SetupMockRouterForTest(WithSystemMode(MockSystemMode{Active: "extender", Selected: "extender"}))
		DeferCleanup(server.Close)

		active, selected, err := Common.CheckRouterMode()
		Expect(err).To(HaveOccurred())
		Expect(active).To(Equal("extender"))
		Expect(selected).To(Equal("extender"))
		Expect(err.Error()).To(ContainSubstring("router is not in router mode"))
	})

	It("should fail in mixed mode", func() {
		server := SetupMockRouterForTest(WithSystemMode(MockSystemMode{Active: "router", Selected: "extender"}))
		DeferCleanup(server.Close)

		active, selected, err := Common.CheckRouterMode()
		Expect(err).To(HaveOccurred())
		Expect(active).To(Equal("router"))
		Expect(selected).To(Equal("extender"))
		Expect(err.Error()).To(ContainSubstring("router is not in router mode"))
	})
})
