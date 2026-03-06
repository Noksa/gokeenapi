package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BatURL Expansion", func() {
	It("should expand YAML bat-url list", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(tmpDir, "all-urls.yaml"), []byte(`bat-url:
  - https://example.com/discord.bat
  - https://example.com/youtube.bat
  - https://example.com/instagram.bat`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "all-urls.yaml"
      - "https://example.com/extra.bat"`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Routes[0].BatURL).To(HaveLen(4))
		Expect(Cfg.Routes[0].BatURL).To(ContainElements(
			"https://example.com/discord.bat", "https://example.com/youtube.bat",
			"https://example.com/instagram.bat", "https://example.com/extra.bat",
		))
	})

	It("should expand YAML bat-url list with absolute path", func() {
		tmpDir := GinkgoT().TempDir()
		batURLListPath := filepath.Join(tmpDir, "subdir", "urllist.yaml")
		Expect(os.MkdirAll(filepath.Dir(batURLListPath), 0755)).To(Succeed())
		Expect(os.WriteFile(batURLListPath, []byte(`bat-url:
  - https://example.com/file1.bat
  - https://example.com/file2.bat`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "`+batURLListPath+`"`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Routes[0].BatURL).To(HaveLen(2))
	})

	It("should fail for non-existent YAML bat-url list", func() {
		tmpDir := GinkgoT().TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "nonexistent.yaml"`), 0644)).To(Succeed())

		err := LoadConfig(configPath)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to read bat-list"))
	})

	It("should handle mixed bat-urls and YAML", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(tmpDir, "urllist.yaml"), []byte(`bat-url:
  - https://example.com/yaml1.bat
  - https://example.com/yaml2.bat`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "https://direct.com/file1.bat"
      - "urllist.yaml"
      - "https://direct.com/file2.bat"`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Routes[0].BatURL).To(HaveLen(4))
		Expect(Cfg.Routes[0].BatURL[0]).To(Equal("https://direct.com/file1.bat"))
	})

	It("should support .yml extension for bat-url", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(tmpDir, "urllist.yml"), []byte(`bat-url:
  - https://example.com/file1.bat
  - https://example.com/file2.bat`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "urllist.yml"`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Routes[0].BatURL).To(HaveLen(2))
	})

	It("should handle empty bat-url list in YAML", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(tmpDir, "empty.yaml"), []byte(`bat-url: []`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "empty.yaml"
      - "https://example.com/file.bat"`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Routes[0].BatURL).To(HaveLen(1))
		Expect(Cfg.Routes[0].BatURL[0]).To(Equal("https://example.com/file.bat"))
	})

	It("should handle invalid YAML structure for bat-url", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(`wrong-key:
  - https://example.com/file.bat`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "invalid.yaml"`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Routes[0].BatURL).To(BeEmpty())
	})

	It("should expand bat-urls for multiple routes", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(tmpDir, "urllist1.yaml"), []byte(`bat-url:
  - https://route1.com/file1.bat
  - https://route1.com/file2.bat`), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(tmpDir, "urllist2.yaml"), []byte(`bat-url:
  - https://route2.com/file1.bat`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "urllist1.yaml"
  - interfaceId: "Wireguard1"
    bat-url:
      - "urllist2.yaml"
      - "https://direct.com/file.bat"`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Routes).To(HaveLen(2))
		Expect(Cfg.Routes[0].BatURL).To(HaveLen(2))
		Expect(Cfg.Routes[1].BatURL).To(HaveLen(2))
	})
})

var _ = Describe("Combined BatFile and BatURL", func() {
	It("should expand both bat-file and bat-url together", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(tmpDir, "filelist.yaml"), []byte(`bat-file:
  - /path/to/file1.bat
  - /path/to/file2.bat`), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(tmpDir, "urllist.yaml"), []byte(`bat-url:
  - https://example.com/file1.bat
  - https://example.com/file2.bat`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "filelist.yaml"
    bat-url:
      - "urllist.yaml"`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Routes[0].BatFile).To(HaveLen(2))
		Expect(Cfg.Routes[0].BatURL).To(HaveLen(2))
	})

	It("should expand combined YAML with both bat-file and bat-url", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(tmpDir, "combined.yaml"), []byte(`bat-file:
  - /path/to/file1.bat
  - /path/to/file2.bat
bat-url:
  - https://example.com/url1.bat
  - https://example.com/url2.bat`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "combined.yaml"
      - "/extra/file.bat"
    bat-url:
      - "combined.yaml"
      - "https://extra.com/url.bat"`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Routes[0].BatFile).To(HaveLen(3))
		Expect(Cfg.Routes[0].BatURL).To(HaveLen(3))
	})

	It("should only expand bat-file from YAML when referenced in bat-file", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(tmpDir, "combined.yaml"), []byte(`bat-file:
  - /path/to/file1.bat
  - /path/to/file2.bat
bat-url:
  - https://example.com/url1.bat
  - https://example.com/url2.bat`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "combined.yaml"
      - "/extra/file.bat"`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Routes[0].BatFile).To(HaveLen(3))
		Expect(Cfg.Routes[0].BatURL).To(BeEmpty())
	})

	It("should only expand bat-url from YAML when referenced in bat-url", func() {
		tmpDir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(tmpDir, "combined.yaml"), []byte(`bat-file:
  - /path/to/file1.bat
  - /path/to/file2.bat
bat-url:
  - https://example.com/url1.bat
  - https://example.com/url2.bat`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-url:
      - "combined.yaml"
      - "https://extra.com/url.bat"`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.Routes[0].BatFile).To(BeEmpty())
		Expect(Cfg.Routes[0].BatURL).To(HaveLen(3))
	})
})
