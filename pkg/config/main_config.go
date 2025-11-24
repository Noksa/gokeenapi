package config

import (
	"errors"
	"os"
	"path/filepath"

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
}

// Logs contains logging configuration options
type Logs struct {
	// Debug enables debug-level logging for troubleshooting
	Debug bool `yaml:"debug"`
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

	return &batLists, nil
}
