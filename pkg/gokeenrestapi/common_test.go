package gokeenrestapi

import (
	"os"
	"sync"
	"time"

	"github.com/noksa/gokeenapi/pkg/config"
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

var _ = Describe("cache cleanup", func() {
	var tmpDir string
	var gokeenDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "gokeenapi-cache-test-*")
		Expect(err).NotTo(HaveOccurred())
		config.Cfg = config.GokeenapiConfig{
			Keenetic: config.Keenetic{URL: "http://127.0.0.1:9999", Login: "admin", Password: "pass"},
			DataDir:  tmpDir,
		}
		// GetGokeenDir creates tmpDir/.gokeenapi
		gokeenDir = tmpDir + "/.gokeenapi"
		Expect(os.MkdirAll(gokeenDir, 0700)).To(Succeed())
		restyClient = nil
		restyClientOnce = sync.Once{}
		cleanedOldCache = false
	})

	AfterEach(func() {
		_ = os.RemoveAll(tmpDir)
		cleanedOldCache = false
	})

	It("should not error when a stale file is already deleted by a concurrent process", func() {
		// Create a stale file (modified > cacheCleanupPeriod ago)
		staleFile, err := os.CreateTemp(gokeenDir, "stale-*.json")
		Expect(err).NotTo(HaveOccurred())
		stalePath := staleFile.Name()
		Expect(staleFile.Close()).To(Succeed())

		// Back-date modification time past the cleanup period
		oldTime := time.Now().Add(-(cacheCleanupPeriod + time.Hour))
		Expect(os.Chtimes(stalePath, oldTime, oldTime)).To(Succeed())

		// Simulate concurrent deletion before our WalkDir removes it
		Expect(os.Remove(stalePath)).To(Succeed())

		// getKeeneticCacheFile should succeed even though the file is already gone
		_, err = Common.getKeeneticCacheFile()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should remove stale files that still exist", func() {
		staleFile, err := os.CreateTemp(gokeenDir, "stale-*.json")
		Expect(err).NotTo(HaveOccurred())
		stalePath := staleFile.Name()
		Expect(staleFile.Close()).To(Succeed())

		oldTime := time.Now().Add(-(cacheCleanupPeriod + time.Hour))
		Expect(os.Chtimes(stalePath, oldTime, oldTime)).To(Succeed())

		_, err = Common.getKeeneticCacheFile()
		Expect(err).NotTo(HaveOccurred())
		Expect(stalePath).NotTo(BeAnExistingFile())
	})

	It("should not remove fresh files", func() {
		freshFile, err := os.CreateTemp(gokeenDir, "fresh-*.json")
		Expect(err).NotTo(HaveOccurred())
		freshPath := freshFile.Name()
		Expect(freshFile.Close()).To(Succeed())

		_, err = Common.getKeeneticCacheFile()
		Expect(err).NotTo(HaveOccurred())
		Expect(freshPath).To(BeAnExistingFile())
	})
})
