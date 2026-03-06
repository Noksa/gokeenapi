package config

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LoadConfig", func() {
	Context("basic loading", func() {
		It("should load a valid config file", func() {
			configContent := `keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file: ["routes.bat"]
dns:
  records:
    - domain: "test.local"
      ip: ["192.168.1.100"]
logs:
  debug: true`

			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(configContent), 0644)).To(Succeed())

			Expect(LoadConfig(configPath)).To(Succeed())
			Expect(Cfg.Keenetic.URL).To(Equal("http://192.168.1.1"))
			Expect(Cfg.Keenetic.Login).To(Equal("admin"))
			Expect(Cfg.Keenetic.Password).To(Equal("password"))
			Expect(Cfg.Routes).To(HaveLen(1))
			Expect(Cfg.Routes[0].InterfaceID).To(Equal("Wireguard0"))
			Expect(Cfg.DNS.Records).To(HaveLen(1))
			Expect(Cfg.DNS.Records[0].Domain).To(Equal("test.local"))
			Expect(Cfg.Logs.Debug).To(BeTrue())
		})

		It("should fail for non-existent file", func() {
			Expect(LoadConfig("/nonexistent/config.yaml")).To(HaveOccurred())
		})

		It("should fail for invalid YAML", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "invalid.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
    bat-file: ["routes.bat"
dns:`), 0644)).To(Succeed())

			Expect(LoadConfig(configPath)).To(HaveOccurred())
		})

		It("should fail for empty path", func() {
			_ = os.Unsetenv("GOKEENAPI_CONFIG")
			err := LoadConfig("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("config path is empty"))
		})

		It("should load from GOKEENAPI_CONFIG env var", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "env_config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"`), 0644)).To(Succeed())

			_ = os.Setenv("GOKEENAPI_CONFIG", configPath)
			DeferCleanup(func() { _ = os.Unsetenv("GOKEENAPI_CONFIG") })

			Expect(LoadConfig("")).To(Succeed())
			Expect(Cfg.Keenetic.URL).To(Equal("http://192.168.1.1"))
		})

		It("should override credentials from env vars", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"`), 0644)).To(Succeed())

			_ = os.Setenv("GOKEENAPI_KEENETIC_LOGIN", "env_admin")
			_ = os.Setenv("GOKEENAPI_KEENETIC_PASSWORD", "env_password")
			DeferCleanup(func() {
				_ = os.Unsetenv("GOKEENAPI_KEENETIC_LOGIN")
				_ = os.Unsetenv("GOKEENAPI_KEENETIC_PASSWORD")
			})

			Expect(LoadConfig(configPath)).To(Succeed())
			Expect(Cfg.Keenetic.URL).To(Equal("http://192.168.1.1"))
			Expect(Cfg.Keenetic.Login).To(Equal("env_admin"))
			Expect(Cfg.Keenetic.Password).To(Equal("env_password"))
		})

		It("should set DataDir in Docker environment", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"`), 0644)).To(Succeed())

			_ = os.Setenv("GOKEENAPI_INSIDE_DOCKER", "true")
			DeferCleanup(func() { _ = os.Unsetenv("GOKEENAPI_INSIDE_DOCKER") })

			Expect(LoadConfig(configPath)).To(Succeed())
			Expect(Cfg.DataDir).To(Equal("/etc/gokeenapi"))
		})

		It("should load routes without bat-file or bat-url", func() {
			tmpDir := GinkgoT().TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			Expect(os.WriteFile(configPath, []byte(`keenetic:
  url: "http://192.168.1.1"
  login: "admin"
  password: "password"
routes:
  - interfaceId: "Wireguard0"
  - interfaceId: "Wireguard1"
dns:
  records:
    - domain: "test.local"
      ip: ["192.168.1.100"]`), 0644)).To(Succeed())

			Expect(LoadConfig(configPath)).To(Succeed())
			Expect(Cfg.Routes).To(HaveLen(2))
			Expect(Cfg.Routes[0].BatFile).To(BeEmpty())
			Expect(Cfg.Routes[0].BatURL).To(BeEmpty())
			Expect(Cfg.Routes[1].BatFile).To(BeEmpty())
			Expect(Cfg.Routes[1].BatURL).To(BeEmpty())
		})
	})
})
