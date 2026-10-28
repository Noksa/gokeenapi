package config

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Exporter configuration", func() {
	var dir, path string
	BeforeEach(func() {
		dir = GinkgoT().TempDir()
		path = filepath.Join(dir, "exporter.yaml")
		Expect(os.WriteFile(filepath.Join(dir, "router.yaml"), []byte("keenetic:\n  url: http://127.0.0.1:1234\n  login: monitor\n  password: secret\ndns:\n  routes:\n    groups: [missing.yaml]\n"), 0o600)).To(Succeed())
	})
	write := func(body string) {
		Expect(os.WriteFile(path, []byte(body), 0o600)).To(Succeed())
	}
	It("loads defaults and relative router paths without expanding unrelated lists or mutating global config", func() {
		before := Cfg
		write("routers:\n  - name: home\n    config: router.yaml\n")
		cfg, err := LoadExporterConfig(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Listen).To(Equal("127.0.0.1:9479"))
		Expect(cfg.Collectors["dns"].Interval).To(Equal(30 * time.Second))
		Expect(cfg.Collectors["hosts"].Interval).To(Equal(time.Minute))
		Expect(cfg.Collectors["routing"].Interval).To(Equal(5 * time.Minute))
		Expect(cfg.Collectors["info"].Interval).To(Equal(time.Hour))
		Expect(cfg.Routers[0].Config).To(Equal(filepath.Join(dir, "router.yaml")))
		Expect(cfg.Routers[0].Connection.Login).To(Equal("monitor"))
		Expect(Cfg).To(Equal(before))
	})
	It("applies overrides, disabled collectors, and credential environment variables", func() {
		GinkgoT().Setenv("GOKEENAPI_KEENETIC_LOGIN", "env-user")
		GinkgoT().Setenv("GOKEENAPI_KEENETIC_PASSWORD", "env-password")
		write("collectors:\n  dns:\n    interval: 45s\n  routing:\n    enabled: false\nrouters:\n  - name: home\n    config: router.yaml\n    interfaces: [Wireguard0]\n")
		cfg, err := LoadExporterConfig(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Collectors["dns"].Interval).To(Equal(45 * time.Second))
		Expect(cfg.Collectors["dns"].MaxAge).To(Equal(90 * time.Second))
		Expect(*cfg.Collectors["routing"].Enabled).To(BeFalse())
		Expect(cfg.Routers[0].Connection.Password).To(Equal("env-password"))
	})
	DescribeTable("rejects invalid exporter settings", func(prefix string) {
		write(prefix + "\nrouters:\n  - name: home\n    config: router.yaml\n")
		_, err := LoadExporterConfig(path)
		Expect(err).To(HaveOccurred())
	},
		Entry("unknown field", "typo: true"),
		Entry("unknown collector", "collectors:\n  typo: {}"),
		Entry("too frequent", "collectors:\n  dns:\n    interval: 1ms"),
		Entry("negative interval", "collectors:\n  dns:\n    interval: -1s"),
		Entry("expiry shorter than interval", "collectors:\n  dns:\n    max_age: 1s"),
		Entry("unbounded concurrency", "max_concurrent_routers: -1"),
		Entry("invalid listen address", "listen: bad-address"),
		Entry("zero timeout", "timeout: 0s"),
	)
	It("rejects duplicate router names", func() {
		write("routers:\n  - {name: home, config: router.yaml}\n  - {name: home, config: router.yaml}\n")
		_, err := LoadExporterConfig(path)
		Expect(err).To(MatchError(ContainSubstring("duplicate")))
	})
	It("rejects missing routers", func() {
		write("routers: []\n")
		_, err := LoadExporterConfig(path)
		Expect(err).To(HaveOccurred())
	})
	It("rejects credentials in router URLs without disclosing them", func() {
		Expect(os.WriteFile(filepath.Join(dir, "router.yaml"), []byte("keenetic:\n  url: http://user:do-not-disclose@localhost\n  login: a\n  password: b\n"), 0o600)).To(Succeed())
		write("routers:\n  - {name: home, config: router.yaml}\n")
		_, err := LoadExporterConfig(path)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).NotTo(ContainSubstring("do-not-disclose"))
	})
})
