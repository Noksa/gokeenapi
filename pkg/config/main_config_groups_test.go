package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Groups Expansion", func() {
	It("should expand groups from YAML file", func() {
		tmpDir := GinkgoT().TempDir()
		domainsDir := filepath.Join(tmpDir, "domains")
		Expect(os.MkdirAll(domainsDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(domainsDir, "youtube.yaml"), []byte(`domain-url:
  - https://example.com/youtube.txt`), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(domainsDir, "telegram.yaml"), []byte(`domain-url:
  - https://example.com/telegram.txt`), 0644)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(tmpDir, "common_groups.yaml"), []byte(`groups:
  - name: youtube
    domain-url:
      - domains/youtube.yaml
    interfaceId: Wireguard0
  - name: telegram
    domain-url:
      - domains/telegram.yaml
    interfaceId: Wireguard0`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
dns:
  routes:
    groups:
      - common_groups.yaml`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.DNS.Routes.Groups).To(HaveLen(2))
		Expect(Cfg.DNS.Routes.Groups[0].Name).To(Equal("youtube"))
		Expect(Cfg.DNS.Routes.Groups[0].InterfaceID).To(Equal("Wireguard0"))
		Expect(Cfg.DNS.Routes.Groups[0].isFileReference).To(BeFalse())
		Expect(Cfg.DNS.Routes.Groups[1].Name).To(Equal("telegram"))
	})

	It("should mix imported and local groups", func() {
		tmpDir := GinkgoT().TempDir()
		domainsDir := filepath.Join(tmpDir, "domains")
		Expect(os.MkdirAll(domainsDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(domainsDir, "youtube.yaml"), []byte(`domain-url:
  - https://example.com/youtube.txt`), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(domainsDir, "local.txt"), []byte("local1.com\nlocal2.com"), 0644)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(tmpDir, "common.yaml"), []byte(`groups:
  - name: youtube
    domain-url:
      - domains/youtube.yaml
    interfaceId: Wireguard0`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
dns:
  routes:
    groups:
      - common.yaml
      - name: local-only
        domain-file:
          - domains/local.txt
        interfaceId: GigabitEthernet0`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.DNS.Routes.Groups).To(HaveLen(2))
		Expect(Cfg.DNS.Routes.Groups[0].Name).To(Equal("youtube"))
		Expect(Cfg.DNS.Routes.Groups[1].Name).To(Equal("local-only"))
		Expect(Cfg.DNS.Routes.Groups[1].InterfaceID).To(Equal("GigabitEthernet0"))
	})

	It("should work with old format (no imports)", func() {
		tmpDir := GinkgoT().TempDir()
		domainsDir := filepath.Join(tmpDir, "domains")
		Expect(os.MkdirAll(domainsDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(domainsDir, "youtube.yaml"), []byte(`domain-url:
  - https://example.com/youtube.txt`), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(domainsDir, "work.txt"), []byte("work1.com\nwork2.com"), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
dns:
  routes:
    groups:
      - name: youtube
        domain-url:
          - domains/youtube.yaml
        interfaceId: Wireguard0
      - name: work
        domain-file:
          - domains/work.txt
        interfaceId: Wireguard0`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.DNS.Routes.Groups).To(HaveLen(2))
		Expect(Cfg.DNS.Routes.Groups[0].Name).To(Equal("youtube"))
		Expect(Cfg.DNS.Routes.Groups[1].Name).To(Equal("work"))
	})

	It("should not treat group with .yaml name as import when it has fields", func() {
		tmpDir := GinkgoT().TempDir()
		domainsDir := filepath.Join(tmpDir, "domains")
		Expect(os.MkdirAll(domainsDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(domainsDir, "test.txt"), []byte("test1.com\ntest2.com"), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
dns:
  routes:
    groups:
      - name: my-group.yaml
        domain-file:
          - domains/test.txt
        interfaceId: Wireguard0`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.DNS.Routes.Groups).To(HaveLen(1))
		Expect(Cfg.DNS.Routes.Groups[0].Name).To(Equal("my-group.yaml"))
		Expect(Cfg.DNS.Routes.Groups[0].DomainFile).To(HaveLen(1))
		Expect(Cfg.DNS.Routes.Groups[0].isFileReference).To(BeFalse())
	})

	It("should resolve domain-file paths relative to groups file", func() {
		tmpDir := GinkgoT().TempDir()
		domainsDir := filepath.Join(tmpDir, "domains")
		Expect(os.MkdirAll(domainsDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(domainsDir, "googleplay.txt"), []byte("play.google.com\nandroid.clients.google.com"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(domainsDir, "work.txt"), []byte("work1.com\nwork2.com"), 0644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(domainsDir, "youtube.yaml"), []byte(`domain-url:
  - https://example.com/youtube.txt`), 0644)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(tmpDir, "common_groups.yaml"), []byte(`groups:
  - name: googleplay
    domain-file:
      - domains/googleplay.txt
    interfaceId: Wireguard0
  - name: work
    domain-file:
      - domains/work.txt
    interfaceId: Wireguard0
  - name: youtube
    domain-url:
      - domains/youtube.yaml
    interfaceId: Wireguard0`), 0644)).To(Succeed())

		configPath := filepath.Join(tmpDir, "config.yaml")
		Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
dns:
  routes:
    groups:
      - common_groups.yaml`), 0644)).To(Succeed())

		Expect(LoadConfig(configPath)).To(Succeed())
		Expect(Cfg.DNS.Routes.Groups).To(HaveLen(3))

		// googleplay path should be absolute and exist
		googleplayPath := Cfg.DNS.Routes.Groups[0].DomainFile[0]
		Expect(filepath.IsAbs(googleplayPath)).To(BeTrue())
		Expect(googleplayPath).To(BeAnExistingFile())

		// work path should be absolute and exist
		workPath := Cfg.DNS.Routes.Groups[1].DomainFile[0]
		Expect(filepath.IsAbs(workPath)).To(BeTrue())
		Expect(workPath).To(BeAnExistingFile())

		// youtube domain-url should be expanded
		Expect(Cfg.DNS.Routes.Groups[2].DomainURL).To(HaveLen(1))
		Expect(Cfg.DNS.Routes.Groups[2].DomainURL[0]).To(Equal("https://example.com/youtube.txt"))
	})
})
