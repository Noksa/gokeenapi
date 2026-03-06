package config

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"
)

// Generator utilities for configuration property-based testing

// genValidURL generates valid router URLs
func genValidURL() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		protocol := rapid.SampledFrom([]string{"http", "https"}).Draw(t, "protocol")

		useIP := rapid.Bool().Draw(t, "useIP")
		var host string
		if useIP {
			octet1 := rapid.IntRange(1, 255).Draw(t, "octet1")
			octet2 := rapid.IntRange(0, 255).Draw(t, "octet2")
			octet3 := rapid.IntRange(0, 255).Draw(t, "octet3")
			octet4 := rapid.IntRange(1, 254).Draw(t, "octet4")
			host = fmt.Sprintf("%d.%d.%d.%d", octet1, octet2, octet3, octet4)
		} else {
			host = rapid.StringMatching(`[a-z][a-z0-9-]{0,10}\.[a-z]{2,5}`).Draw(t, "hostname")
		}

		includePort := rapid.Bool().Draw(t, "includePort")
		if includePort {
			port := rapid.IntRange(1, 65535).Draw(t, "port")
			return fmt.Sprintf("%s://%s:%d", protocol, host, port)
		}

		return fmt.Sprintf("%s://%s", protocol, host)
	})
}

// genLogin generates valid login strings
func genLogin() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_-]{2,20}`)
}

// genPassword generates valid password strings
func genPassword() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*()_+=-]{4,30}`)
}

// genKeeneticConfig generates valid Keenetic configuration
func genKeeneticConfig() *rapid.Generator[Keenetic] {
	return rapid.Custom(func(t *rapid.T) Keenetic {
		return Keenetic{
			URL:      genValidURL().Draw(t, "url"),
			Login:    genLogin().Draw(t, "login"),
			Password: genPassword().Draw(t, "password"),
		}
	})
}

// genInterfaceID generates valid interface IDs
func genInterfaceID() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		prefix := rapid.SampledFrom([]string{"Wireguard", "Ethernet", "Bridge", "Vlan"}).Draw(t, "prefix")
		number := rapid.IntRange(0, 9).Draw(t, "number")
		return fmt.Sprintf("%s%d", prefix, number)
	})
}

// genBatFilePath generates valid bat file paths
func genBatFilePath() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		depth := rapid.IntRange(1, 4).Draw(t, "depth")
		components := make([]string, depth)

		for i := 0; i < depth-1; i++ {
			components[i] = rapid.StringMatching(`[a-z][a-z0-9_-]{2,10}`).Draw(t, fmt.Sprintf("dir%d", i))
		}

		filename := rapid.StringMatching(`[a-z][a-z0-9_-]{2,15}`).Draw(t, "filename")
		components[depth-1] = filename + ".bat"

		isAbsolute := rapid.Bool().Draw(t, "isAbsolute")
		if isAbsolute {
			return "/" + filepath.Join(components...)
		}
		return filepath.Join(components...)
	})
}

// genYAMLFilePath generates valid YAML file paths
func genYAMLFilePath() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		depth := rapid.IntRange(1, 3).Draw(t, "depth")
		components := make([]string, depth)

		for i := 0; i < depth-1; i++ {
			components[i] = rapid.StringMatching(`[a-z][a-z0-9_-]{2,10}`).Draw(t, fmt.Sprintf("dir%d", i))
		}

		filename := rapid.StringMatching(`[a-z][a-z0-9_-]{2,15}`).Draw(t, "filename")
		ext := rapid.SampledFrom([]string{".yaml", ".yml"}).Draw(t, "ext")
		components[depth-1] = filename + ext

		isAbsolute := rapid.Bool().Draw(t, "isAbsolute")
		if isAbsolute {
			return "/" + filepath.Join(components...)
		}
		return filepath.Join(components...)
	})
}

// genBatFileList generates a list of bat file paths (mix of .bat and .yaml files)
func genBatFileList() *rapid.Generator[[]string] {
	return rapid.Custom(func(t *rapid.T) []string {
		size := rapid.IntRange(0, 10).Draw(t, "size")
		files := make([]string, size)

		for i := range size {
			useYAML := rapid.Bool().Draw(t, fmt.Sprintf("useYAML%d", i))
			if useYAML {
				files[i] = genYAMLFilePath().Draw(t, fmt.Sprintf("yamlFile%d", i))
			} else {
				files[i] = genBatFilePath().Draw(t, fmt.Sprintf("batFile%d", i))
			}
		}

		return files
	})
}

// genBatURLList generates a list of bat file URLs
func genBatURLList() *rapid.Generator[[]string] {
	return rapid.Custom(func(t *rapid.T) []string {
		size := rapid.IntRange(0, 5).Draw(t, "size")
		urls := make([]string, size)

		for i := range size {
			protocol := rapid.SampledFrom([]string{"http", "https"}).Draw(t, fmt.Sprintf("protocol%d", i))
			domain := rapid.StringMatching(`[a-z][a-z0-9-]{2,15}\.[a-z]{2,5}`).Draw(t, fmt.Sprintf("domain%d", i))
			path := rapid.StringMatching(`[a-z][a-z0-9/_-]{5,30}`).Draw(t, fmt.Sprintf("path%d", i))
			urls[i] = fmt.Sprintf("%s://%s/%s.bat", protocol, domain, path)
		}

		return urls
	})
}

// genRoute generates a valid Route configuration
func genRoute() *rapid.Generator[Route] {
	return rapid.Custom(func(t *rapid.T) Route {
		return Route{
			InterfaceID: genInterfaceID().Draw(t, "interfaceID"),
			BatFileList: BatFileList{
				BatFile: genBatFileList().Draw(t, "batFile"),
			},
			BatURLList: BatURLList{
				BatURL: genBatURLList().Draw(t, "batURL"),
			},
		}
	})
}

// genRouteList generates a list of Route configurations
func genRouteList() *rapid.Generator[[]Route] {
	return rapid.SliceOfN(genRoute(), 0, 5)
}

// genDomainName generates valid domain names
func genDomainName() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z][a-z0-9-]{1,20}\.[a-z]{2,10}`)
}

// genIPList generates a list of IP addresses
func genIPList() *rapid.Generator[[]string] {
	return rapid.Custom(func(t *rapid.T) []string {
		size := rapid.IntRange(1, 5).Draw(t, "size")
		ips := make([]string, size)

		for i := range size {
			octet1 := rapid.IntRange(1, 255).Draw(t, fmt.Sprintf("octet1_%d", i))
			octet2 := rapid.IntRange(0, 255).Draw(t, fmt.Sprintf("octet2_%d", i))
			octet3 := rapid.IntRange(0, 255).Draw(t, fmt.Sprintf("octet3_%d", i))
			octet4 := rapid.IntRange(1, 254).Draw(t, fmt.Sprintf("octet4_%d", i))
			ips[i] = fmt.Sprintf("%d.%d.%d.%d", octet1, octet2, octet3, octet4)
		}

		return ips
	})
}

// genDNSRecord generates a valid DNS record
func genDNSRecord() *rapid.Generator[DnsRecord] {
	return rapid.Custom(func(t *rapid.T) DnsRecord {
		return DnsRecord{
			Domain: genDomainName().Draw(t, "domain"),
			IP:     genIPList().Draw(t, "ip"),
		}
	})
}

// genDNSRecordList generates a list of DNS records
func genDNSRecordList() *rapid.Generator[[]DnsRecord] {
	return rapid.SliceOfN(genDNSRecord(), 0, 10)
}

// genDNS generates a valid DNS configuration
func genDNS() *rapid.Generator[DNS] {
	return rapid.Custom(func(t *rapid.T) DNS {
		return DNS{
			Records: genDNSRecordList().Draw(t, "records"),
		}
	})
}

// genDnsRoutingGroup generates a valid DNS routing group
func genDnsRoutingGroup() *rapid.Generator[DnsRoutingGroup] {
	return rapid.Custom(func(t *rapid.T) DnsRoutingGroup {
		name := rapid.StringMatching(`[a-z][a-z0-9-]{2,20}`).Draw(t, "name")

		fileCount := rapid.IntRange(1, 3).Draw(t, "fileCount")
		domainFiles := make([]string, fileCount)
		for i := range fileCount {
			domainFiles[i] = rapid.StringMatching(`[a-z0-9-]{5,15}\.txt`).Draw(t, fmt.Sprintf("file%d", i))
		}

		return DnsRoutingGroup{
			Name:        name,
			DomainFile:  domainFiles,
			InterfaceID: genInterfaceID().Draw(t, "interfaceID"),
		}
	})
}

// genWhitespaceString generates strings that are empty or contain only whitespace
func genWhitespaceString() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		choice := rapid.IntRange(0, 5).Draw(t, "choice")
		switch choice {
		case 0:
			return ""
		case 1:
			return " "
		case 2:
			return "  "
		case 3:
			return "\t"
		case 4:
			return "\n"
		case 5:
			length := rapid.IntRange(1, 10).Draw(t, "length")
			chars := make([]rune, length)
			for i := range length {
				chars[i] = rapid.SampledFrom([]rune{' ', '\t', '\n', '\r'}).Draw(t, fmt.Sprintf("char%d", i))
			}
			return string(chars)
		default:
			return ""
		}
	})
}

// genLogs generates a valid Logs configuration
func genLogs() *rapid.Generator[Logs] {
	return rapid.Custom(func(t *rapid.T) Logs {
		return Logs{
			Debug: rapid.Bool().Draw(t, "debug"),
		}
	})
}

// genDataDir generates a valid data directory path
func genDataDir() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		depth := rapid.IntRange(1, 4).Draw(t, "depth")
		components := make([]string, depth)

		for i := range depth {
			components[i] = rapid.StringMatching(`[a-z][a-z0-9_-]{2,10}`).Draw(t, fmt.Sprintf("dir%d", i))
		}

		return "/" + filepath.Join(components...)
	})
}

// genGokeenapiConfig generates a valid complete GokeenapiConfig
func genGokeenapiConfig() *rapid.Generator[GokeenapiConfig] {
	return rapid.Custom(func(t *rapid.T) GokeenapiConfig {
		cfg := GokeenapiConfig{
			Keenetic: genKeeneticConfig().Draw(t, "keenetic"),
			Routes:   genRouteList().Draw(t, "routes"),
			DNS:      genDNS().Draw(t, "dns"),
			Logs:     genLogs().Draw(t, "logs"),
		}

		includeDataDir := rapid.Bool().Draw(t, "includeDataDir")
		if includeDataDir {
			cfg.DataDir = genDataDir().Draw(t, "dataDir")
		}

		return cfg
	})
}

// genRelativePath generates a relative file path
func genRelativePath() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		depth := rapid.IntRange(1, 4).Draw(t, "depth")
		components := make([]string, depth)

		for i := 0; i < depth-1; i++ {
			components[i] = rapid.StringMatching(`[a-z][a-z0-9_-]{2,10}`).Draw(t, fmt.Sprintf("dir%d", i))
		}

		filename := rapid.StringMatching(`[a-z][a-z0-9_-]{2,15}`).Draw(t, "filename")
		ext := rapid.SampledFrom([]string{".bat", ".yaml", ".yml"}).Draw(t, "ext")
		components[depth-1] = filename + ext

		return filepath.Join(components...)
	})
}

// Helper function to compare two GokeenapiConfig structs
func configsEqual(c1, c2 GokeenapiConfig) bool {
	if c1.Keenetic.URL != c2.Keenetic.URL ||
		c1.Keenetic.Login != c2.Keenetic.Login ||
		c1.Keenetic.Password != c2.Keenetic.Password {
		return false
	}

	if c1.DataDir != c2.DataDir {
		return false
	}

	if len(c1.Routes) != len(c2.Routes) {
		return false
	}
	for i := range c1.Routes {
		if !routesEqual(c1.Routes[i], c2.Routes[i]) {
			return false
		}
	}

	if !dnsEqual(c1.DNS, c2.DNS) {
		return false
	}

	if c1.Logs.Debug != c2.Logs.Debug {
		return false
	}

	return true
}

func routesEqual(r1, r2 Route) bool {
	if r1.InterfaceID != r2.InterfaceID {
		return false
	}

	if len(r1.BatFile) != len(r2.BatFile) {
		return false
	}
	for i := range r1.BatFile {
		if r1.BatFile[i] != r2.BatFile[i] {
			return false
		}
	}

	if len(r1.BatURL) != len(r2.BatURL) {
		return false
	}
	for i := range r1.BatURL {
		if r1.BatURL[i] != r2.BatURL[i] {
			return false
		}
	}

	return true
}

func dnsEqual(d1, d2 DNS) bool {
	if len(d1.Records) != len(d2.Records) {
		return false
	}

	for i := range d1.Records {
		if d1.Records[i].Domain != d2.Records[i].Domain {
			return false
		}

		if len(d1.Records[i].IP) != len(d2.Records[i].IP) {
			return false
		}

		for j := range d1.Records[i].IP {
			if d1.Records[i].IP[j] != d2.Records[i].IP[j] {
				return false
			}
		}
	}

	return true
}

var _ = Describe("Property: Config YAML Round-Trip", func() {
	It("should preserve structure through marshal/unmarshal cycle", func() {
		rapid.Check(GinkgoT(), func(t *rapid.T) {
			originalCfg := genGokeenapiConfig().Draw(t, "config")

			yamlBytes, err := yaml.Marshal(&originalCfg)
			Expect(err).NotTo(HaveOccurred(), "failed to marshal config to YAML")

			var roundTrippedCfg GokeenapiConfig
			err = yaml.Unmarshal(yamlBytes, &roundTrippedCfg)
			Expect(err).NotTo(HaveOccurred(), "failed to unmarshal YAML back to config")

			Expect(configsEqual(originalCfg, roundTrippedCfg)).To(BeTrue(),
				"round-trip failed:\noriginal: %+v\nround-tripped: %+v", originalCfg, roundTrippedCfg)
		})
	})
})

var _ = Describe("Property: Environment Variable Overrides", func() {
	It("should override YAML credentials with env vars", func() {
		tmpDir := GinkgoT().TempDir()

		rapid.Check(GinkgoT(), func(t *rapid.T) {
			yamlLogin := genLogin().Draw(t, "yamlLogin")
			yamlPassword := genPassword().Draw(t, "yamlPassword")
			envLogin := genLogin().Draw(t, "envLogin")
			envPassword := genPassword().Draw(t, "envPassword")

			if yamlLogin == envLogin {
				envLogin = yamlLogin + "_env"
			}
			if yamlPassword == envPassword {
				envPassword = yamlPassword + "_env"
			}

			cfg := GokeenapiConfig{
				Keenetic: Keenetic{
					URL:      genValidURL().Draw(t, "url"),
					Login:    yamlLogin,
					Password: yamlPassword,
				},
			}

			configPath := filepath.Join(tmpDir, fmt.Sprintf("config_%d.yaml", rapid.IntRange(0, 1000000).Draw(t, "configNum")))
			configBytes, err := yaml.Marshal(&cfg)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(configPath, configBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			err = os.Setenv("GOKEENAPI_KEENETIC_LOGIN", envLogin)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.Unsetenv("GOKEENAPI_KEENETIC_LOGIN") }()

			err = os.Setenv("GOKEENAPI_KEENETIC_PASSWORD", envPassword)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.Unsetenv("GOKEENAPI_KEENETIC_PASSWORD") }()

			err = LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(Cfg.Keenetic.Login).To(Equal(envLogin),
				"environment login not applied: got %s, want %s", Cfg.Keenetic.Login, envLogin)
			Expect(Cfg.Keenetic.Password).To(Equal(envPassword),
				"environment password not applied: got %s, want %s", Cfg.Keenetic.Password, envPassword)
			Expect(Cfg.Keenetic.URL).To(Equal(cfg.Keenetic.URL),
				"URL was incorrectly modified: got %s, want %s", Cfg.Keenetic.URL, cfg.Keenetic.URL)
		})
	})
})

var _ = Describe("Property: Missing Required Fields", func() {
	It("should load configs with missing fields without error", func() {
		tmpDir := GinkgoT().TempDir()

		rapid.Check(GinkgoT(), func(t *rapid.T) {
			missingField := rapid.IntRange(0, 2).Draw(t, "missingField")

			url := genValidURL().Draw(t, "url")
			login := genLogin().Draw(t, "login")
			password := genPassword().Draw(t, "password")

			switch missingField {
			case 0:
				url = ""
			case 1:
				login = ""
			case 2:
				password = ""
			}

			cfg := GokeenapiConfig{
				Keenetic: Keenetic{
					URL:      url,
					Login:    login,
					Password: password,
				},
				Routes: []Route{},
				DNS:    DNS{Records: []DnsRecord{}},
			}

			configPath := filepath.Join(tmpDir, fmt.Sprintf("invalid_config_%d.yaml", rapid.IntRange(0, 1000000).Draw(t, "configNum")))
			configBytes, err := yaml.Marshal(&cfg)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(configPath, configBytes, 0644)
			Expect(err).NotTo(HaveOccurred())

			_ = os.Unsetenv("GOKEENAPI_KEENETIC_LOGIN")
			_ = os.Unsetenv("GOKEENAPI_KEENETIC_PASSWORD")

			err = LoadConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			hasMissingField := Cfg.Keenetic.URL == "" || Cfg.Keenetic.Login == "" || Cfg.Keenetic.Password == ""
			Expect(hasMissingField).To(BeTrue(),
				"expected at least one missing field, but all fields are present: URL=%s, Login=%s, Password=%s",
				Cfg.Keenetic.URL, Cfg.Keenetic.Login, Cfg.Keenetic.Password)
		})
	})
})
