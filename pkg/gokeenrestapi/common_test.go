package gokeenrestapi

import (
	"os"
	"sync"
	"time"

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
		cachedCookieMu.Lock()
		cachedCookie = ""
		cachedCookieMu.Unlock()
	})

	It("should surface auth cache error on request when DataDir is unreadable", func() {
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
		cachedCookieMu.Lock()
		cachedCookie = ""
		cachedCookieMu.Unlock()

		// GetApiClient itself no longer fails; the error surfaces on the first request
		// because cookie injection runs in OnBeforeRequest.
		client, getErr := Common.GetApiClient()
		Expect(getErr).NotTo(HaveOccurred())
		Expect(client).NotTo(BeNil())

		_, reqErr := Common.ExecuteGetSubPath("/rci/show/version")
		Expect(reqErr).To(HaveOccurred())
	})

	It("should be safe for concurrent use", func() {
		server := SetupMockRouterForTest()
		DeferCleanup(server.Close)

		const goroutines = 20
		errs := make(chan error, goroutines)
		var wg sync.WaitGroup
		wg.Add(goroutines)
		for range goroutines {
			go func() {
				defer wg.Done()
				_, err := Common.Version()
				errs <- err
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			Expect(err).NotTo(HaveOccurred())
		}
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
