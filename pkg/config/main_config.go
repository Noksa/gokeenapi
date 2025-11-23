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

// Route defines routing configuration for a specific interface
type Route struct {
	// InterfaceID specifies the target interface (e.g., Wireguard0)
	InterfaceID string `yaml:"interfaceId"`
	// BatFileList is embedded to reuse the BatFile field definition
	BatFileList `yaml:",inline"`
	// BatURL contains URLs to remote .bat files with route definitions
	BatURL []string `yaml:"bat-url"`
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

	// Expand YAML files in bat-file lists
	err = expandBatFileLists(configPath)
	if err != nil {
		return err
	}

	return nil
}

// expandBatFileLists expands .yaml files in bat-file arrays to their contained bat file lists
func expandBatFileLists(configPath string) error {
	for i := range Cfg.Routes {
		expandedBatFiles := []string{}

		for _, batFile := range Cfg.Routes[i].BatFile {
			// Check if this is a YAML file
			if filepath.Ext(batFile) == ".yaml" || filepath.Ext(batFile) == ".yml" {
				// Load bat files from YAML
				batFiles, err := loadBatFileListFromYAML(batFile, configPath)
				if err != nil {
					return err
				}
				expandedBatFiles = append(expandedBatFiles, batFiles...)
			} else {
				// Regular .bat file, keep as is
				expandedBatFiles = append(expandedBatFiles, batFile)
			}
		}

		Cfg.Routes[i].BatFile = expandedBatFiles
	}
	return nil
}

// loadBatFileListFromYAML loads bat file paths from a YAML file
func loadBatFileListFromYAML(listPath, configPath string) ([]string, error) {
	// If listPath is relative, resolve it relative to the config file directory
	if !filepath.IsAbs(listPath) {
		configDir := filepath.Dir(configPath)
		listPath = filepath.Join(configDir, listPath)
	}

	b, err := os.ReadFile(listPath)
	if err != nil {
		return nil, errors.New("failed to read bat-file-list from " + listPath + ": " + err.Error())
	}

	var batFileList BatFileList
	err = yaml.Unmarshal(b, &batFileList)
	if err != nil {
		return nil, errors.New("failed to parse bat-file-list from " + listPath + ": " + err.Error())
	}

	return batFileList.BatFile, nil
}
