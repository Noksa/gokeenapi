package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BatFile Expansion", func() {
	Context("bat-file list expansion", func() {
		It("should expand YAML bat-file list", func() {
			tmpDir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(tmpDir, "all-of-them.yaml"), []byte(`bat-file:
  - /path/to/discord.bat
  - /path/to/youtube.bat
  - /path/to/instagram.bat`), 0644)).To(Succeed())

			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "all-of-them.yaml"
      - "/path/to/extra.bat"`), 0644)).To(Succeed())

			Expect(LoadConfig(configPath)).To(Succeed())
			Expect(Cfg.Routes).To(HaveLen(1))
			Expect(Cfg.Routes[0].BatFile).To(HaveLen(4))
			Expect(Cfg.Routes[0].BatFile).To(ContainElements(
				"/path/to/discord.bat", "/path/to/youtube.bat",
				"/path/to/instagram.bat", "/path/to/extra.bat",
			))
		})

		It("should expand YAML bat-file list with absolute path", func() {
			tmpDir := GinkgoT().TempDir()
			batListPath := filepath.Join(tmpDir, "subdir", "batlist.yaml")
			Expect(os.MkdirAll(filepath.Dir(batListPath), 0755)).To(Succeed())
			Expect(os.WriteFile(batListPath, []byte(`bat-file:
  - /absolute/path/to/file1.bat
  - /absolute/path/to/file2.bat`), 0644)).To(Succeed())

			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "`+batListPath+`"`), 0644)).To(Succeed())

			Expect(LoadConfig(configPath)).To(Succeed())
			Expect(Cfg.Routes[0].BatFile).To(HaveLen(2))
			Expect(Cfg.Routes[0].BatFile).To(ContainElements("/absolute/path/to/file1.bat", "/absolute/path/to/file2.bat"))
		})

		It("should fail for non-existent YAML bat-file list", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "nonexistent.yaml"`), 0644)).To(Succeed())

			err := LoadConfig(configPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to read bat-list"))
		})

		It("should handle mixed bat-files and YAML", func() {
			tmpDir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(tmpDir, "list.yaml"), []byte(`bat-file:
  - /path/from/yaml1.bat
  - /path/from/yaml2.bat`), 0644)).To(Succeed())

			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "/direct/path/file1.bat"
      - "list.yaml"
      - "/direct/path/file2.bat"`), 0644)).To(Succeed())

			Expect(LoadConfig(configPath)).To(Succeed())
			Expect(Cfg.Routes[0].BatFile).To(HaveLen(4))
			Expect(Cfg.Routes[0].BatFile[0]).To(Equal("/direct/path/file1.bat"))
		})

		It("should support .yml extension", func() {
			tmpDir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(tmpDir, "list.yml"), []byte(`bat-file:
  - /path/to/file1.bat
  - /path/to/file2.bat`), 0644)).To(Succeed())

			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "list.yml"`), 0644)).To(Succeed())

			Expect(LoadConfig(configPath)).To(Succeed())
			Expect(Cfg.Routes[0].BatFile).To(HaveLen(2))
		})

		It("should handle empty bat-file list in YAML", func() {
			tmpDir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(tmpDir, "empty.yaml"), []byte(`bat-file: []`), 0644)).To(Succeed())

			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "empty.yaml"
      - "/some/file.bat"`), 0644)).To(Succeed())

			Expect(LoadConfig(configPath)).To(Succeed())
			Expect(Cfg.Routes[0].BatFile).To(HaveLen(1))
			Expect(Cfg.Routes[0].BatFile[0]).To(Equal("/some/file.bat"))
		})

		It("should handle invalid YAML structure gracefully", func() {
			tmpDir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(`wrong-key:
  - /some/file.bat`), 0644)).To(Succeed())

			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "invalid.yaml"`), 0644)).To(Succeed())

			Expect(LoadConfig(configPath)).To(Succeed())
			Expect(Cfg.Routes[0].BatFile).To(BeEmpty())
		})

		It("should not recursively expand nested YAML", func() {
			tmpDir := GinkgoT().TempDir()
			nestedPath := filepath.Join(tmpDir, "nested.yaml")
			Expect(os.WriteFile(nestedPath, []byte(`bat-file:
  - /nested/file1.bat
  - /nested/file2.bat`), 0644)).To(Succeed())

			Expect(os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(`bat-file:
  - /main/file1.bat
  - nested.yaml
  - /main/file2.bat`), 0644)).To(Succeed())

			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "main.yaml"`), 0644)).To(Succeed())

			Expect(LoadConfig(configPath)).To(Succeed())
			Expect(Cfg.Routes[0].BatFile).To(HaveLen(3))
			Expect(Cfg.Routes[0].BatFile[0]).To(Equal("/main/file1.bat"))
			Expect(Cfg.Routes[0].BatFile[1]).To(Equal(nestedPath))
			Expect(Cfg.Routes[0].BatFile[2]).To(Equal("/main/file2.bat"))
		})

		It("should expand bat-files for multiple routes", func() {
			tmpDir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(tmpDir, "list1.yaml"), []byte(`bat-file:
  - /route1/file1.bat
  - /route1/file2.bat`), 0644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, "list2.yaml"), []byte(`bat-file:
  - /route2/file1.bat`), 0644)).To(Succeed())

			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file:
      - "list1.yaml"
  - interfaceId: "Wireguard1"
    bat-file:
      - "list2.yaml"
      - "/direct/file.bat"`), 0644)).To(Succeed())

			Expect(LoadConfig(configPath)).To(Succeed())
			Expect(Cfg.Routes).To(HaveLen(2))
			Expect(Cfg.Routes[0].BatFile).To(HaveLen(2))
			Expect(Cfg.Routes[1].BatFile).To(HaveLen(2))
		})
	})
})
