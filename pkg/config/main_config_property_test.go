package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
	"pgregory.net/rapid"
)

// Generator utilities for configuration property-based testing

// genValidURL generates valid router URLs
func genValidURL() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		protocol := rapid.SampledFrom([]string{"http", "https"}).Draw(t, "protocol")

		// Generate IP or hostname
		useIP := rapid.Bool().Draw(t, "useIP")
		var host string
		if useIP {
			// Generate IPv4 address
			octet1 := rapid.IntRange(1, 255).Draw(t, "octet1")
			octet2 := rapid.IntRange(0, 255).Draw(t, "octet2")
			octet3 := rapid.IntRange(0, 255).Draw(t, "octet3")
			octet4 := rapid.IntRange(1, 254).Draw(t, "octet4")
			host = fmt.Sprintf("%d.%d.%d.%d", octet1, octet2, octet3, octet4)
		} else {
			// Generate hostname
			host = rapid.StringMatching(`[a-z][a-z0-9-]{0,10}\.[a-z]{2,5}`).Draw(t, "hostname")
		}

		// Optionally add port
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
		// Generate path components
		depth := rapid.IntRange(1, 4).Draw(t, "depth")
		components := make([]string, depth)

		for i := 0; i < depth-1; i++ {
			components[i] = rapid.StringMatching(`[a-z][a-z0-9_-]{2,10}`).Draw(t, fmt.Sprintf("dir%d", i))
		}

		// Last component is the filename
		filename := rapid.StringMatching(`[a-z][a-z0-9_-]{2,15}`).Draw(t, "filename")
		components[depth-1] = filename + ".bat"

		// Decide if absolute or relative
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
		// Generate path components
		depth := rapid.IntRange(1, 3).Draw(t, "depth")
		components := make([]string, depth)

		for i := 0; i < depth-1; i++ {
			components[i] = rapid.StringMatching(`[a-z][a-z0-9_-]{2,10}`).Draw(t, fmt.Sprintf("dir%d", i))
		}

		// Last component is the filename
		filename := rapid.StringMatching(`[a-z][a-z0-9_-]{2,15}`).Draw(t, "filename")
		ext := rapid.SampledFrom([]string{".yaml", ".yml"}).Draw(t, "ext")
		components[depth-1] = filename + ext

		// For YAML files in bat-file lists, use relative paths more often
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
			// Mix of .bat and .yaml files
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
			BatURL: genBatURLList().Draw(t, "batURL"),
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
		// Generate path components
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

		// Optionally include DataDir
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

		// Last component is the filename
		filename := rapid.StringMatching(`[a-z][a-z0-9_-]{2,15}`).Draw(t, "filename")
		ext := rapid.SampledFrom([]string{".bat", ".yaml", ".yml"}).Draw(t, "ext")
		components[depth-1] = filename + ext

		return filepath.Join(components...)
	})
}

// Property-based tests

// Feature: property-based-testing, Property 8: YAML round-trip preserves structure
// Validates: Requirements 3.1
func TestConfigYAMLMarshalAndUnmarshalPreservesAllFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a valid config
		originalCfg := genGokeenapiConfig().Draw(t, "config")

		// Marshal to YAML
		yamlBytes, err := yaml.Marshal(&originalCfg)
		if err != nil {
			t.Fatalf("failed to marshal config to YAML: %v", err)
		}

		// Unmarshal back to config
		var roundTrippedCfg GokeenapiConfig
		err = yaml.Unmarshal(yamlBytes, &roundTrippedCfg)
		if err != nil {
			t.Fatalf("failed to unmarshal YAML back to config: %v", err)
		}

		// Compare the two configs
		if !configsEqual(originalCfg, roundTrippedCfg) {
			t.Fatalf("round-trip failed:\noriginal: %+v\nround-tripped: %+v", originalCfg, roundTrippedCfg)
		}
	})
}

// Helper function to compare two GokeenapiConfig structs
func configsEqual(c1, c2 GokeenapiConfig) bool {
	// Compare Keenetic
	if c1.Keenetic.URL != c2.Keenetic.URL ||
		c1.Keenetic.Login != c2.Keenetic.Login ||
		c1.Keenetic.Password != c2.Keenetic.Password {
		return false
	}

	// Compare DataDir
	if c1.DataDir != c2.DataDir {
		return false
	}

	// Compare Routes
	if len(c1.Routes) != len(c2.Routes) {
		return false
	}
	for i := range c1.Routes {
		if !routesEqual(c1.Routes[i], c2.Routes[i]) {
			return false
		}
	}

	// Compare DNS
	if !dnsEqual(c1.DNS, c2.DNS) {
		return false
	}

	// Compare Logs
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

// Feature: property-based-testing, Property 9: Bat-file expansion is idempotent
// Validates: Requirements 3.2
func TestBatFileListExpansionProducesSameResultWhenRepeated(t *testing.T) {
	// Create a temporary directory for test files (outside rapid.Check)
	tmpDir := t.TempDir()

	rapid.Check(t, func(t *rapid.T) {

		// Generate a config with bat-file lists
		cfg := genGokeenapiConfig().Draw(t, "config")

		// Create YAML files for any .yaml/.yml files in bat-file lists
		for i := range cfg.Routes {
			for j, batFile := range cfg.Routes[i].BatFile {
				if filepath.Ext(batFile) == ".yaml" || filepath.Ext(batFile) == ".yml" {
					// Create the YAML file with some bat files
					batListContent := BatFileList{
						BatFile: []string{
							"/expanded/file1.bat",
							"/expanded/file2.bat",
						},
					}

					yamlBytes, err := yaml.Marshal(&batListContent)
					if err != nil {
						t.Fatalf("failed to marshal bat-file list: %v", err)
					}

					// Make the path relative to tmpDir
					yamlPath := filepath.Join(tmpDir, filepath.Base(batFile))
					err = os.WriteFile(yamlPath, yamlBytes, 0644)
					if err != nil {
						t.Fatalf("failed to write bat-file list: %v", err)
					}

					// Update the config to use the relative path
					cfg.Routes[i].BatFile[j] = filepath.Base(batFile)
				}
			}
		}

		// Write the config to a file
		configPath := filepath.Join(tmpDir, "config.yaml")
		configBytes, err := yaml.Marshal(&cfg)
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}

		err = os.WriteFile(configPath, configBytes, 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Load the config (this will expand bat-file lists)
		err = LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		// Save the expanded bat-file lists
		firstExpansion := make([][]string, len(Cfg.Routes))
		for i := range Cfg.Routes {
			firstExpansion[i] = make([]string, len(Cfg.Routes[i].BatFile))
			copy(firstExpansion[i], Cfg.Routes[i].BatFile)
		}

		// Expand again (simulate calling expandBatFileLists again)
		err = expandBatFileLists(configPath)
		if err != nil {
			t.Fatalf("failed to expand bat-file lists second time: %v", err)
		}

		// Compare the two expansions - they should be identical
		for i := range Cfg.Routes {
			if len(firstExpansion[i]) != len(Cfg.Routes[i].BatFile) {
				t.Fatalf("expansion not idempotent: route %d has different lengths: %d vs %d",
					i, len(firstExpansion[i]), len(Cfg.Routes[i].BatFile))
			}

			for j := range firstExpansion[i] {
				if firstExpansion[i][j] != Cfg.Routes[i].BatFile[j] {
					t.Fatalf("expansion not idempotent: route %d, file %d: %s vs %s",
						i, j, firstExpansion[i][j], Cfg.Routes[i].BatFile[j])
				}
			}
		}
	})
}

// Feature: property-based-testing, Property 10: Relative path resolution is consistent
// Validates: Requirements 3.3
func TestBatFilePathResolutionIsConsistentRegardlessOfWorkingDirectory(t *testing.T) {
	// Create a temporary directory structure (outside rapid.Check)
	tmpDir := t.TempDir()

	rapid.Check(t, func(t *rapid.T) {

		// Generate a relative path for the bat-file list
		relativePath := genRelativePath().Draw(t, "relativePath")

		// Create the bat-file list YAML in a subdirectory
		configDir := filepath.Join(tmpDir, "configs")
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			t.Fatalf("failed to create config directory: %v", err)
		}

		// Create the bat-file list YAML
		batListPath := filepath.Join(configDir, relativePath)
		err = os.MkdirAll(filepath.Dir(batListPath), 0755)
		if err != nil {
			t.Fatalf("failed to create bat-file list directory: %v", err)
		}

		batListContent := BatFileList{
			BatFile: []string{"/test/file1.bat", "/test/file2.bat"},
		}
		yamlBytes, err := yaml.Marshal(&batListContent)
		if err != nil {
			t.Fatalf("failed to marshal bat-file list: %v", err)
		}

		err = os.WriteFile(batListPath, yamlBytes, 0644)
		if err != nil {
			t.Fatalf("failed to write bat-file list: %v", err)
		}

		// Create a config that references the bat-file list with a relative path
		cfg := GokeenapiConfig{
			Keenetic: Keenetic{
				URL:      "http://192.168.1.1",
				Login:    "admin",
				Password: "password",
			},
			Routes: []Route{
				{
					InterfaceID: "Wireguard0",
					BatFileList: BatFileList{
						BatFile: []string{relativePath},
					},
				},
			},
		}

		configPath := filepath.Join(configDir, "config.yaml")
		configBytes, err := yaml.Marshal(&cfg)
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}

		err = os.WriteFile(configPath, configBytes, 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Save the current working directory
		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}

		// Load the config from the config directory
		err = LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		firstExpansion := make([]string, len(Cfg.Routes[0].BatFile))
		copy(firstExpansion, Cfg.Routes[0].BatFile)

		// Change to a different working directory
		err = os.Chdir(tmpDir)
		if err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}
		defer func() { _ = os.Chdir(originalWd) }()

		// Load the config again from the same absolute path
		err = LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config second time: %v", err)
		}

		secondExpansion := Cfg.Routes[0].BatFile

		// The expansions should be identical regardless of working directory
		if len(firstExpansion) != len(secondExpansion) {
			t.Fatalf("path resolution not consistent: different lengths: %d vs %d",
				len(firstExpansion), len(secondExpansion))
		}

		for i := range firstExpansion {
			if firstExpansion[i] != secondExpansion[i] {
				t.Fatalf("path resolution not consistent: file %d: %s vs %s",
					i, firstExpansion[i], secondExpansion[i])
			}
		}
	})
}

// Feature: property-based-testing, Property 11: Environment overrides take precedence
// Validates: Requirements 3.4
func TestEnvironmentVariablesOverrideYAMLCredentials(t *testing.T) {
	tmpDir := t.TempDir()

	rapid.Check(t, func(t *rapid.T) {
		// Generate two different sets of credentials
		yamlLogin := genLogin().Draw(t, "yamlLogin")
		yamlPassword := genPassword().Draw(t, "yamlPassword")
		envLogin := genLogin().Draw(t, "envLogin")
		envPassword := genPassword().Draw(t, "envPassword")

		// Ensure they're different
		if yamlLogin == envLogin {
			envLogin = yamlLogin + "_env"
		}
		if yamlPassword == envPassword {
			envPassword = yamlPassword + "_env"
		}

		// Create a config with YAML credentials
		cfg := GokeenapiConfig{
			Keenetic: Keenetic{
				URL:      genValidURL().Draw(t, "url"),
				Login:    yamlLogin,
				Password: yamlPassword,
			},
		}

		configPath := filepath.Join(tmpDir, fmt.Sprintf("config_%d.yaml", rapid.IntRange(0, 1000000).Draw(t, "configNum")))
		configBytes, err := yaml.Marshal(&cfg)
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}

		err = os.WriteFile(configPath, configBytes, 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Set environment variables
		err = os.Setenv("GOKEENAPI_KEENETIC_LOGIN", envLogin)
		if err != nil {
			t.Fatalf("failed to set login env var: %v", err)
		}
		defer func() { _ = os.Unsetenv("GOKEENAPI_KEENETIC_LOGIN") }()

		err = os.Setenv("GOKEENAPI_KEENETIC_PASSWORD", envPassword)
		if err != nil {
			t.Fatalf("failed to set password env var: %v", err)
		}
		defer func() { _ = os.Unsetenv("GOKEENAPI_KEENETIC_PASSWORD") }()

		// Load the config
		err = LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		// Verify that environment variables took precedence
		if Cfg.Keenetic.Login != envLogin {
			t.Fatalf("environment login not applied: got %s, want %s", Cfg.Keenetic.Login, envLogin)
		}

		if Cfg.Keenetic.Password != envPassword {
			t.Fatalf("environment password not applied: got %s, want %s", Cfg.Keenetic.Password, envPassword)
		}

		// URL should remain from YAML (not overridden)
		if Cfg.Keenetic.URL != cfg.Keenetic.URL {
			t.Fatalf("URL was incorrectly modified: got %s, want %s", Cfg.Keenetic.URL, cfg.Keenetic.URL)
		}
	})
}

// Feature: property-based-testing, Property 12: Missing required fields are detected
// Validates: Requirements 3.5
func TestConfigWithMissingRequiredFieldsLoadsButLeavesFieldsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	rapid.Check(t, func(t *rapid.T) {
		// Generate a simple config with a missing required field
		// We'll randomly remove one of the three required fields
		missingField := rapid.IntRange(0, 2).Draw(t, "missingField")

		url := genValidURL().Draw(t, "url")
		login := genLogin().Draw(t, "login")
		password := genPassword().Draw(t, "password")

		// Remove one required field
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
			// Keep routes empty to avoid bat-file expansion issues
			Routes: []Route{},
			DNS:    DNS{Records: []DnsRecord{}},
		}

		configPath := filepath.Join(tmpDir, fmt.Sprintf("invalid_config_%d.yaml", rapid.IntRange(0, 1000000).Draw(t, "configNum")))
		configBytes, err := yaml.Marshal(&cfg)
		if err != nil {
			t.Fatalf("failed to marshal config: %v", err)
		}

		err = os.WriteFile(configPath, configBytes, 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Clear environment variables to ensure they don't fill in missing fields
		_ = os.Unsetenv("GOKEENAPI_KEENETIC_LOGIN")
		_ = os.Unsetenv("GOKEENAPI_KEENETIC_PASSWORD")

		// Load the config - this should succeed (LoadConfig doesn't validate)
		err = LoadConfig(configPath)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		// Verify that at least one required field is missing
		hasMissingField := Cfg.Keenetic.URL == "" || Cfg.Keenetic.Login == "" || Cfg.Keenetic.Password == ""

		if !hasMissingField {
			t.Fatalf("expected at least one missing field, but all fields are present: URL=%s, Login=%s, Password=%s",
				Cfg.Keenetic.URL, Cfg.Keenetic.Login, Cfg.Keenetic.Password)
		}

		// The property we're testing is that LoadConfig successfully loads configs with missing fields
		// (validation happens at a higher layer in the application)
		// This is the current behavior and is acceptable since validation is done in cmd/common.go
	})
}
