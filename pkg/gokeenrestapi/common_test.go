package gokeenrestapi

import (
	"os"
	"sync"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ExecutePostParse", func() {
	AfterEach(func() {
		CleanupTestConfig()
	})

	It("should skip batch and return error when response body is not valid JSON", func() {
		server := SetupMockRouterForTest(WithRciBody([]byte("not valid json")))
		DeferCleanup(server.Close)

		responses, err := Common.ExecutePostParse(
			gokeenrestapimodels.ParseRequest{Parse: "interface Wireguard0 up"},
		)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid"))
		// No partial results should leak through on unmarshal failure
		Expect(responses).To(BeEmpty())
	})
})

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

var _ = Describe("GetApiClient", func() {
	AfterEach(func() {
		CleanupTestConfig()
		restyClient = nil
		restyClientOnce = sync.Once{}
		cleanedOldCache = false
	})

	It("should return error instead of panicking when auth cache directory is unreadable", func() {
		// Set up a DataDir that exists but has no read permission so getAuthCookie fails
		tmpDir, err := os.MkdirTemp("", "gokeenapi-test-noperm-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			_ = os.Chmod(tmpDir, 0o755)
			_ = os.RemoveAll(tmpDir)
		})
		Expect(os.Chmod(tmpDir, 0o000)).To(Succeed())

		config.Cfg = config.GokeenapiConfig{
			Keenetic: config.Keenetic{URL: "http://127.0.0.1:9999", Login: "admin", Password: "pass"},
			DataDir:  tmpDir,
		}
		restyClient = nil
		restyClientOnce = sync.Once{}
		cleanedOldCache = false

		client, getErr := Common.GetApiClient()
		Expect(getErr).To(HaveOccurred())
		Expect(client).To(BeNil())
	})
})
