package config

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"gopkg.in/yaml.v3"
)

var (
	Cfg = GokeenapiConfig{}
)

// Runtime holds runtime configuration that is not persisted to YAML
type Runtime struct {
	RouterInfo struct {
		Version gokeenrestapimodels.Version `yaml:"-"`
	} `yaml:"-"`
}

// GokeenapiConfig represents the main configuration structure for the application
type GokeenapiConfig struct {
	// Keenetic router connection settings
	Keenetic Keenetic `yaml:"keenetic"`
	// DataDir specifies custom data directory for storing application data (optional)
	DataDir string `yaml:"dataDir,omitempty"`
	// Routes contains list of routing configurations for different interfaces
	Routes []Route `yaml:"routes"`
	// DNS contains DNS records configuration
	DNS DNS `yaml:"dns"`
	// Logs contains logging configuration (optional)
	Logs Logs `yaml:"logs,omitempty"`
}

// Keenetic holds connection parameters for the Keenetic router
type Keenetic struct {
	// URL of the router (IP address or KeenDNS hostname with http/https)
	URL string `yaml:"url"`
	// Login for router admin access (can be overridden by GOKEENAPI_KEENETIC_LOGIN env var)
	Login string `yaml:"login"`
	// Password for router admin access (can be overridden by GOKEENAPI_KEENETIC_PASSWORD env var)
	Password string `yaml:"password"`
}

// BatFileList represents the structure of a YAML file containing bat-file paths
type BatFileList struct {
	// BatFile contains list of paths to .bat files or .yaml/.yml files
	// When a .yaml/.yml file is specified, it's loaded and expanded to its contained bat-file paths
	// This allows sharing common bat-file lists across multiple configurations
	// Example YAML structure: bat-file: ["/path/to/file1.bat", "/path/to/file2.bat"]
	BatFile []string `yaml:"bat-file"`
}

// BatURLList represents the structure of a YAML file containing bat-url paths
type BatURLList struct {
	// BatURL contains list of URLs to remote .bat files or .yaml/.yml files
	// When a .yaml/.yml file is specified, it's loaded and expanded to its contained bat-url paths
	// This allows sharing common bat-url lists across multiple configurations
	// Example YAML structure: bat-url: ["https://example.com/file1.bat", "https://example.com/file2.bat"]
	BatURL []string `yaml:"bat-url"`
}

// BatLists combines both bat-file and bat-url lists for efficient loading
type BatLists struct {
	BatFileList `yaml:",inline"`
	BatURLList  `yaml:",inline"`
}

// DomainFileList represents the structure of a YAML file containing domain-file paths
type DomainFileList struct {
	// DomainFile contains list of paths to .txt files or .yaml/.yml files
	// When a .yaml/.yml file is specified, it's loaded and expanded to its contained domain-file paths
	// This allows sharing common domain-file lists across multiple configurations
	// Example YAML structure: domain-file: ["/path/to/file1.txt", "/path/to/file2.txt"]
	DomainFile []string `yaml:"domain-file"`
}

// DomainURLList represents the structure of a YAML file containing domain-url paths
type DomainURLList struct {
	// DomainURL contains list of URLs to remote .txt files or .yaml/.yml files
	// When a .yaml/.yml file is specified, it's loaded and expanded to its contained domain-url paths
	// This allows sharing common domain-url lists across multiple configurations
	// Example YAML structure: domain-url: ["https://example.com/file1.txt", "https://example.com/file2.txt"]
	DomainURL []string `yaml:"domain-url"`
}

// DomainLists combines both domain-file and domain-url lists for efficient loading
type DomainLists struct {
	DomainFileList `yaml:",inline"`
	DomainURLList  `yaml:",inline"`
}

// Route defines routing configuration for a specific interface
type Route struct {
	// InterfaceID specifies the target interface (e.g., Wireguard0)
	InterfaceID string `yaml:"interfaceId"`
	// BatFileList is embedded to reuse the BatFile field definition
	BatFileList `yaml:",inline"`
	// BatURLList is embedded to reuse the BatURL field definition
	BatURLList `yaml:",inline"`
}

// DnsRecord represents a single DNS record with domain and IP addresses
type DnsRecord struct {
	// Domain name for the DNS record
	Domain string `yaml:"domain"`
	// IP addresses associated with the domain (supports multiple IPs)
	IP []string `yaml:"ip"`
}

// DNS contains DNS-related configuration
type DNS struct {
	// Records contains list of DNS records to manage
	Records []DnsRecord `yaml:"records"`
	// Routes contains DNS-routing configuration
	Routes DnsRoutes `yaml:"routes"`
}

// DnsRoutes contains DNS-routing configuration
type DnsRoutes struct {
	// Groups contains list of domain groups with routing policies
	Groups []DnsRoutingGroup `yaml:"groups"`
}

// DnsRoutingGroup represents a domain group with associated routing policy
type DnsRoutingGroup struct {
	// Name is the unique identifier for the object-group
	Name string `yaml:"name"`
	// DomainFile contains list of local .txt files with domains (one per line)
	// Can also reference .yaml files containing lists of domain-file paths
	DomainFile []string `yaml:"domain-file"`
	// DomainURL contains list of remote URLs serving .txt files with domains
	DomainURL []string `yaml:"domain-url"`
	// InterfaceID specifies the target interface for routing
	InterfaceID string `yaml:"interfaceId"`
}

// Logs contains logging configuration options
type Logs struct {
	// Debug enables debug-level logging for troubleshooting
	Debug bool `yaml:"debug"`
}

// isValidDomain validates a domain name according to RFC 1035 and RFC 1123
func isValidDomain(domain string) bool {
	// Domain must not be empty
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}

	// Remove trailing dot if present (FQDN)
	domain = strings.TrimSuffix(domain, ".")

	// Domain must contain at least one dot (have a TLD)
	if !strings.Contains(domain, ".") {
		return false
	}

	// Split into labels
	labels := strings.Split(domain, ".")

	// Must have at least 2 labels (domain + TLD)
	if len(labels) < 2 {
		return false
	}

	// Validate each label
	labelRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
	for _, label := range labels {
		// Label must not be empty
		if len(label) == 0 {
			return false
		}

		// Label must not exceed 63 characters
		if len(label) > 63 {
			return false
		}

		// Label must match the pattern: start with alphanumeric, end with alphanumeric,
		// middle can contain hyphens
		if !labelRegex.MatchString(label) {
			return false
		}
	}

	// TLD (last label) should not be all numeric
	tld := labels[len(labels)-1]
	allNumeric := true
	for _, r := range tld {
		if r < '0' || r > '9' {
			allNumeric = false
			break
		}
	}
	if allNumeric {
		return false
	}

	return true
}

// isValidIP validates an IPv4 address
func isValidIP(ip string) bool {
	// Use net.ParseIP for validation
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check if it's IPv4 (we only support IPv4 for now)
	if parsedIP.To4() == nil {
		return false
	}

	// Additional validation: no leading zeros in octets
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}

	for _, part := range parts {
		// Check for leading zeros (except "0" itself)
		if len(part) > 1 && part[0] == '0' {
			return false
		}
	}

	return true
}

// ValidateDnsRoutingGroups validates DNS routing group configurations
func ValidateDnsRoutingGroups(groups []DnsRoutingGroup) error {
	// Track group names to check for duplicates
	seenNames := make(map[string]int)

	for i, group := range groups {
		// Check for empty or whitespace-only group names
		if len(group.Name) == 0 {
			return errors.New("DNS routing group name cannot be empty")
		}

		// Check if name contains only whitespace
		trimmed := ""
		for _, r := range group.Name {
			if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
				trimmed += string(r)
			}
		}
		if len(trimmed) == 0 {
			return errors.New("DNS routing group name cannot contain only whitespace")
		}

		// Check for duplicate group names
		if firstIndex, exists := seenNames[group.Name]; exists {
			return errors.New("duplicate DNS routing group name '" + group.Name + "' found at positions " + string(rune(firstIndex)) + " and " + string(rune(i)))
		}
		seenNames[group.Name] = i

		// Check for empty domain sources (must have at least one domain-file or domain-url)
		if len(group.DomainFile) == 0 && len(group.DomainURL) == 0 {
			return errors.New("DNS routing group must contain at least one domain-file or domain-url")
		}

		// Check for empty interface ID
		if len(group.InterfaceID) == 0 {
			return errors.New("interface ID cannot be empty in DNS routing group " + group.Name + " at position " + string(rune(i)))
		}
	}

	return nil
}

// ValidateDomainList validates a list of domains/IPs
func ValidateDomainList(domains []string, groupName string) error {
	for j, domain := range domains {
		if len(domain) == 0 {
			return errors.New("domain or IP address cannot be empty in DNS routing group " + groupName)
		}

		// Check if domain contains only whitespace
		domainTrimmed := ""
		for _, r := range domain {
			if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
				domainTrimmed += string(r)
			}
		}
		if len(domainTrimmed) == 0 {
			return errors.New("domain or IP address cannot contain only whitespace in DNS routing group " + groupName + " at position " + string(rune(j)))
		}

		// Validate domain or IP format
		// Try to parse as IP first, then as domain
		if !isValidIP(domain) && !isValidDomain(domain) {
			return errors.New("invalid domain or IP address '" + domain + "' in DNS routing group " + groupName)
		}
	}

	return nil
}

func LoadConfig(configPath string) error {
	if configPath == "" {
		v, ok := os.LookupEnv("GOKEENAPI_CONFIG")
		if ok {
			configPath = v
		} else {
			return errors.New("config path is empty. Specify it via --config flag or GOKEENAPI_CONFIG environment variable")
		}
	}
	b, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(b, &Cfg)
	if err != nil {
		return err
	}
	// read some sensitive variables and replace values if found
	v, ok := os.LookupEnv("GOKEENAPI_KEENETIC_LOGIN")
	if ok {
		Cfg.Keenetic.Login = v
	}
	v, ok = os.LookupEnv("GOKEENAPI_KEENETIC_PASSWORD")
	if ok {
		Cfg.Keenetic.Password = v
	}
	_, ok = os.LookupEnv("GOKEENAPI_INSIDE_DOCKER")
	if ok {
		Cfg.DataDir = "/etc/gokeenapi"
	}

	// Expand YAML files in bat-file and bat-url lists
	err = expandBatLists(configPath)
	if err != nil {
		return err
	}

	// Expand YAML files in domain-file and domain-url lists
	err = expandDomainLists(configPath)
	if err != nil {
		return err
	}

	return nil
}

// expandBatLists expands .yaml files in bat-file and bat-url arrays to their contained lists
// This function reads each YAML file only once and extracts both bat-file and bat-url lists
func expandBatLists(configPath string) error {
	// Cache for loaded YAML files to avoid reading the same file multiple times
	yamlCache := make(map[string]*BatLists)

	for i := range Cfg.Routes {
		expandedBatFiles := []string{}
		expandedBatURLs := []string{}

		// Collect all unique YAML file paths from both bat-file and bat-url
		yamlFiles := make(map[string]bool)
		for _, batFile := range Cfg.Routes[i].BatFile {
			if filepath.Ext(batFile) == ".yaml" || filepath.Ext(batFile) == ".yml" {
				yamlFiles[batFile] = true
			}
		}
		for _, batURL := range Cfg.Routes[i].BatURL {
			if filepath.Ext(batURL) == ".yaml" || filepath.Ext(batURL) == ".yml" {
				yamlFiles[batURL] = true
			}
		}

		// Load all YAML files into cache
		for yamlFile := range yamlFiles {
			if _, exists := yamlCache[yamlFile]; !exists {
				batLists, err := loadBatListsFromYAML(yamlFile, configPath)
				if err != nil {
					return err
				}
				yamlCache[yamlFile] = batLists
			}
		}

		// Expand bat-file list
		for _, batFile := range Cfg.Routes[i].BatFile {
			if filepath.Ext(batFile) == ".yaml" || filepath.Ext(batFile) == ".yml" {
				// Load bat files from cached YAML
				if batLists, exists := yamlCache[batFile]; exists {
					expandedBatFiles = append(expandedBatFiles, batLists.BatFile...)
				}
			} else {
				// Regular .bat file, keep as is
				expandedBatFiles = append(expandedBatFiles, batFile)
			}
		}

		// Expand bat-url list
		for _, batURL := range Cfg.Routes[i].BatURL {
			if filepath.Ext(batURL) == ".yaml" || filepath.Ext(batURL) == ".yml" {
				// Load bat URLs from cached YAML
				if batLists, exists := yamlCache[batURL]; exists {
					expandedBatURLs = append(expandedBatURLs, batLists.BatURL...)
				}
			} else {
				// Regular URL, keep as is
				expandedBatURLs = append(expandedBatURLs, batURL)
			}
		}

		Cfg.Routes[i].BatFile = expandedBatFiles
		Cfg.Routes[i].BatURL = expandedBatURLs
	}
	return nil
}

// loadBatListsFromYAML loads both bat-file and bat-url lists from a YAML file
func loadBatListsFromYAML(listPath, configPath string) (*BatLists, error) {
	// If listPath is relative, resolve it relative to the config file directory
	if !filepath.IsAbs(listPath) {
		configDir := filepath.Dir(configPath)
		listPath = filepath.Join(configDir, listPath)
	}

	b, err := os.ReadFile(listPath)
	if err != nil {
		return nil, errors.New("failed to read bat-list from " + listPath + ": " + err.Error())
	}

	var batLists BatLists
	err = yaml.Unmarshal(b, &batLists)
	if err != nil {
		return nil, errors.New("failed to parse bat-list from " + listPath + ": " + err.Error())
	}

	// Resolve paths in bat-file relative to the YAML file's directory
	yamlDir := filepath.Dir(listPath)
	for i, batFile := range batLists.BatFile {
		if !filepath.IsAbs(batFile) {
			batLists.BatFile[i] = filepath.Join(yamlDir, batFile)
		}
	}

	return &batLists, nil
}

// expandDomainLists expands .yaml files in domain-file and domain-url arrays to their contained lists
// This function reads each YAML file only once and extracts both domain-file and domain-url lists
func expandDomainLists(configPath string) error {
	// Cache for loaded YAML files to avoid reading the same file multiple times
	yamlCache := make(map[string]*DomainLists)

	for i := range Cfg.DNS.Routes.Groups {
		expandedDomainFiles := []string{}
		expandedDomainURLs := []string{}

		// Collect all unique YAML file paths from both domain-file and domain-url
		yamlFiles := make(map[string]bool)
		for _, domainFile := range Cfg.DNS.Routes.Groups[i].DomainFile {
			if filepath.Ext(domainFile) == ".yaml" || filepath.Ext(domainFile) == ".yml" {
				yamlFiles[domainFile] = true
			}
		}
		for _, domainURL := range Cfg.DNS.Routes.Groups[i].DomainURL {
			if filepath.Ext(domainURL) == ".yaml" || filepath.Ext(domainURL) == ".yml" {
				yamlFiles[domainURL] = true
			}
		}

		// Load all YAML files into cache
		for yamlFile := range yamlFiles {
			if _, exists := yamlCache[yamlFile]; !exists {
				domainLists, err := loadDomainListsFromYAML(yamlFile, configPath)
				if err != nil {
					return err
				}
				yamlCache[yamlFile] = domainLists
			}
		}

		// Expand domain-file list
		for _, domainFile := range Cfg.DNS.Routes.Groups[i].DomainFile {
			if filepath.Ext(domainFile) == ".yaml" || filepath.Ext(domainFile) == ".yml" {
				// Load domain files from cached YAML
				if domainLists, exists := yamlCache[domainFile]; exists {
					expandedDomainFiles = append(expandedDomainFiles, domainLists.DomainFile...)
				}
			} else {
				// Regular .txt file - resolve relative paths relative to config file
				resolvedPath := domainFile
				if !filepath.IsAbs(resolvedPath) {
					configDir := filepath.Dir(configPath)
					resolvedPath = filepath.Join(configDir, resolvedPath)
				}
				expandedDomainFiles = append(expandedDomainFiles, resolvedPath)
			}
		}

		// Expand domain-url list
		for _, domainURL := range Cfg.DNS.Routes.Groups[i].DomainURL {
			if filepath.Ext(domainURL) == ".yaml" || filepath.Ext(domainURL) == ".yml" {
				// Load domain URLs from cached YAML
				if domainLists, exists := yamlCache[domainURL]; exists {
					expandedDomainURLs = append(expandedDomainURLs, domainLists.DomainURL...)
				}
			} else {
				// Regular URL, keep as is
				expandedDomainURLs = append(expandedDomainURLs, domainURL)
			}
		}

		Cfg.DNS.Routes.Groups[i].DomainFile = expandedDomainFiles
		Cfg.DNS.Routes.Groups[i].DomainURL = expandedDomainURLs
	}
	return nil
}

// loadDomainListsFromYAML loads both domain-file and domain-url lists from a YAML file
func loadDomainListsFromYAML(listPath, configPath string) (*DomainLists, error) {
	// If listPath is relative, resolve it relative to the config file directory
	if !filepath.IsAbs(listPath) {
		configDir := filepath.Dir(configPath)
		listPath = filepath.Join(configDir, listPath)
	}

	b, err := os.ReadFile(listPath)
	if err != nil {
		return nil, errors.New("failed to read domain-list from " + listPath + ": " + err.Error())
	}

	var domainLists DomainLists
	err = yaml.Unmarshal(b, &domainLists)
	if err != nil {
		return nil, errors.New("failed to parse domain-list from " + listPath + ": " + err.Error())
	}

	// Resolve paths in domain-file relative to the YAML file's directory
	yamlDir := filepath.Dir(listPath)
	for i, domainFile := range domainLists.DomainFile {
		if !filepath.IsAbs(domainFile) {
			domainLists.DomainFile[i] = filepath.Join(yamlDir, domainFile)
		}
	}

	return &domainLists, nil
}
