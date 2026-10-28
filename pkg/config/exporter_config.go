package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ExporterCollector configures one independently scheduled collection group.
type ExporterCollector struct {
	Enabled  *bool         `yaml:"enabled,omitempty"`
	Interval time.Duration `yaml:"interval,omitempty"`
	MaxAge   time.Duration `yaml:"max_age,omitempty"`
}

// ExporterRouter references an existing router configuration file.
type ExporterRouter struct {
	Name       string   `yaml:"name"`
	Config     string   `yaml:"config"`
	Interfaces []string `yaml:"interfaces,omitempty"`
	Connection Keenetic `yaml:"-"`
}

// ExporterConfig controls the HTTP endpoint and bounded background polling.
type ExporterConfig struct {
	Listen               string                       `yaml:"listen"`
	Timeout              time.Duration                `yaml:"timeout"`
	MaxConcurrentRouters int                          `yaml:"max_concurrent_routers"`
	MaxBackoff           time.Duration                `yaml:"max_backoff"`
	Collectors           map[string]ExporterCollector `yaml:"collectors"`
	Routers              []ExporterRouter             `yaml:"routers"`
}

// LoadExporterConfig loads settings without changing the CLI's global config.
func LoadExporterConfig(path string) (*ExporterConfig, error) {
	path, err := ResolveConfigPath(path)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &ExporterConfig{Listen: "127.0.0.1:9479", Timeout: 10 * time.Second, MaxConcurrentRouters: 4, MaxBackoff: 5 * time.Minute}
	decoder := yaml.NewDecoder(bytes.NewReader(b))
	decoder.KnownFields(true)
	if err := decoder.Decode(cfg); err != nil {
		return nil, errors.New("invalid exporter YAML: check field names and value types")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("exporter config must contain exactly one YAML document")
	}
	if err := ValidateExporterListen(cfg.Listen); err != nil {
		return nil, err
	}
	if cfg.Timeout < time.Second || cfg.Timeout > 5*time.Minute {
		return nil, errors.New("timeout must be between 1s and 5m")
	}
	if cfg.MaxConcurrentRouters < 1 || cfg.MaxConcurrentRouters > 64 {
		return nil, errors.New("max_concurrent_routers must be between 1 and 64")
	}
	if cfg.MaxBackoff < time.Second || cfg.MaxBackoff > 24*time.Hour {
		return nil, errors.New("max_backoff must be between 1s and 24h")
	}
	defaults := map[string]time.Duration{"system": 30 * time.Second, "dns": 30 * time.Second, "interfaces": 30 * time.Second, "hosts": time.Minute, "routing": 5 * time.Minute, "info": time.Hour}
	for name := range cfg.Collectors {
		if _, ok := defaults[name]; !ok {
			return nil, fmt.Errorf("unknown exporter collector %q", name)
		}
	}
	if cfg.Collectors == nil {
		cfg.Collectors = make(map[string]ExporterCollector)
	}
	enabled := 0
	for name, interval := range defaults {
		c := cfg.Collectors[name]
		if c.Interval == 0 {
			c.Interval = interval
		}
		if c.Interval < 5*time.Second || c.Interval > 24*time.Hour {
			return nil, fmt.Errorf("collector %s interval must be between 5s and 24h", name)
		}
		if c.MaxAge == 0 {
			c.MaxAge = 2 * c.Interval
		}
		if c.MaxAge < c.Interval || c.MaxAge > 7*24*time.Hour {
			return nil, fmt.Errorf("collector %s max_age must be at least its interval and at most 7d", name)
		}
		if c.Enabled == nil || *c.Enabled {
			enabled++
		}
		cfg.Collectors[name] = c
	}
	if enabled == 0 || len(cfg.Routers) == 0 {
		return nil, errors.New("at least one router and one enabled collector are required")
	}
	names := make(map[string]bool)
	for i := range cfg.Routers {
		r := &cfg.Routers[i]
		if strings.TrimSpace(r.Name) == "" || r.Config == "" {
			return nil, errors.New("each exporter router needs a name and config path")
		}
		if names[r.Name] {
			return nil, fmt.Errorf("duplicate exporter router name %q", r.Name)
		}
		names[r.Name] = true
		if !filepath.IsAbs(r.Config) {
			r.Config = filepath.Join(filepath.Dir(path), r.Config)
		}
		connection, err := loadExporterConnection(r.Config)
		if err != nil {
			return nil, fmt.Errorf("router %q: %w", r.Name, err)
		}
		r.Connection = connection
		seen := make(map[string]bool)
		for _, id := range r.Interfaces {
			if strings.TrimSpace(id) == "" || seen[id] {
				return nil, fmt.Errorf("router %q: interface IDs must be nonempty and unique", r.Name)
			}
			seen[id] = true
		}
	}
	return cfg, nil
}

// ValidateExporterListen applies the same listener constraints to YAML and CLI overrides.
func ValidateExporterListen(address string) error {
	_, port, err := net.SplitHostPort(address)
	numericPort, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || numericPort < 1 || numericPort > 65535 {
		return errors.New("listen must be a host:port address with a port from 1 to 65535")
	}
	return nil
}

func loadExporterConnection(path string) (Keenetic, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Keenetic{}, fmt.Errorf("read router config: %w", err)
	}
	// Deliberately ignore command data and imported lists; exporter needs only
	// connection settings and must never load or modify the CLI global Cfg.
	var connection struct {
		Keenetic Keenetic `yaml:"keenetic"`
	}
	if err := yaml.Unmarshal(b, &connection); err != nil {
		return Keenetic{}, errors.New("invalid router YAML: check connection settings")
	}
	c := connection.Keenetic
	if value, ok := os.LookupEnv("GOKEENAPI_KEENETIC_LOGIN"); ok {
		c.Login = value
	}
	if value, ok := os.LookupEnv("GOKEENAPI_KEENETIC_PASSWORD"); ok {
		c.Password = value
	}
	if err := ValidateMonitoringConnection(c); err != nil {
		return Keenetic{}, err
	}
	return c, nil
}

// ValidateMonitoringConnection validates an isolated session without printing secrets.
func ValidateMonitoringConnection(c Keenetic) error {
	u, err := url.Parse(c.URL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return errors.New("keenetic.url must be an HTTP(S) origin without credentials, path, query or fragment")
	}
	if c.Login == "" || c.Password == "" {
		return errors.New("keenetic.login and keenetic.password are required")
	}
	if c.Timeout < 0 {
		return errors.New("keenetic.timeout must not be negative")
	}
	return nil
}
