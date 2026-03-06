package config

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetURLCacheTTL", func() {
	It("should return default TTL when not configured", func() {
		Cfg = GokeenapiConfig{}
		Expect(GetURLCacheTTL()).To(Equal(time.Minute))
	})

	It("should return configured TTL", func() {
		Cfg = GokeenapiConfig{Cache: Cache{URLTTL: 5 * time.Minute}}
		Expect(GetURLCacheTTL()).To(Equal(5 * time.Minute))
	})

	It("should return default TTL when zero", func() {
		Cfg = GokeenapiConfig{Cache: Cache{URLTTL: 0}}
		Expect(GetURLCacheTTL()).To(Equal(time.Minute))
	})

	It("should return default TTL when negative", func() {
		Cfg = GokeenapiConfig{Cache: Cache{URLTTL: -5 * time.Minute}}
		Expect(GetURLCacheTTL()).To(Equal(time.Minute))
	})
})

var _ = Describe("LoadConfig with cache TTL", func() {
	It("should load cache TTL from config", func() {
		tmpDir := GinkgoT().TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(`
keenetic:
  url: http://192.168.1.1
  login: admin
  password: pass
cache:
  urlTtl: 10m
`), 0644)
		Expect(err).NotTo(HaveOccurred())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Cache.URLTTL).To(Equal(10 * time.Minute))
		Expect(GetURLCacheTTL()).To(Equal(10 * time.Minute))
	})

	It("should use default TTL when cache section is absent", func() {
		Cfg = GokeenapiConfig{}
		tmpDir := GinkgoT().TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		err := os.WriteFile(configPath, []byte(`
keenetic:
  url: http://192.168.1.1
  login: admin
  password: pass
`), 0644)
		Expect(err).NotTo(HaveOccurred())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Cache.URLTTL).To(Equal(time.Duration(0)))
		Expect(GetURLCacheTTL()).To(Equal(time.Minute))
	})
})
