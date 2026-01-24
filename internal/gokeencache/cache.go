package gokeencache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/patrickmn/go-cache"
)

const (
	rciShowInterfaces = "rci_show_interfaces"
	runtimeConfig     = "runtime_config"
	rciShowIpRoute    = "rci_show_ip_route"
	domainValidation  = "domain_validation_"
)

var (
	c = cache.New(cache.NoExpiration, cache.NoExpiration)
)

type urlCacheEntry struct {
	Content   string    `json:"content"`
	Checksum  string    `json:"checksum"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GetGokeenDir returns the .gokeenapi directory path and ensures it exists
func GetGokeenDir() (string, error) {
	var dataDir string
	var err error
	if config.Cfg.DataDir != "" {
		dataDir = path.Clean(config.Cfg.DataDir)
	} else {
		dataDir, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
	}
	gokeenDir := path.Join(dataDir, ".gokeenapi")
	err = os.MkdirAll(gokeenDir, os.ModePerm)
	return gokeenDir, err
}

func urlToCacheFilename(url string) string {
	hash := md5.Sum([]byte(url))
	return fmt.Sprintf("url_%x.json", hash)
}

// ComputeChecksum calculates MD5 checksum of content
func ComputeChecksum(content []byte) [16]byte {
	return md5.Sum(content)
}

func UpdateRuntimeConfig(f func(runtime *config.Runtime)) {
	cfg := GetRuntimeConfig()
	f(cfg)
	c.Set(runtimeConfig, cfg, cache.NoExpiration)
}

func GetRuntimeConfig() *config.Runtime {
	cfg, ok := c.Get(runtimeConfig)
	if ok {
		return cfg.(*config.Runtime)
	}
	return &config.Runtime{}
}

func SetRciShowIpRoute(routes []gokeenrestapimodels.RciShowIpRoute) {
	c.Set(rciShowIpRoute, routes, cache.NoExpiration)
}

func GetRciShowIpRoute() []gokeenrestapimodels.RciShowIpRoute {
	v, ok := c.Get(rciShowIpRoute)
	if !ok {
		return nil
	}
	return v.([]gokeenrestapimodels.RciShowIpRoute)
}

func SetRciShowInterfaces(m map[string]gokeenrestapimodels.RciShowInterface) {
	c.Set(rciShowInterfaces, m, cache.NoExpiration)
}

func GetRciShowInterfaces() map[string]gokeenrestapimodels.RciShowInterface {
	v, ok := c.Get(rciShowInterfaces)
	if !ok {
		return nil
	}
	return v.(map[string]gokeenrestapimodels.RciShowInterface)
}

func SetURLContent(url string, content string, ttl time.Duration) {
	checksum := fmt.Sprintf("%x", md5.Sum([]byte(content)))
	entry := urlCacheEntry{
		Content:   content,
		Checksum:  checksum,
		ExpiresAt: time.Now().Add(ttl),
	}

	gokeenDir, err := GetGokeenDir()
	if err != nil {
		return
	}

	filename := urlToCacheFilename(url)
	filepath := path.Join(gokeenDir, filename)

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	_ = os.WriteFile(filepath, data, 0600)
}

func GetURLContent(url string) (string, bool) {
	gokeenDir, err := GetGokeenDir()
	if err != nil {
		return "", false
	}

	filename := urlToCacheFilename(url)
	filepath := path.Join(gokeenDir, filename)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", false
	}

	var entry urlCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return "", false
	}

	if time.Now().After(entry.ExpiresAt) {
		// Cache expired - return miss but keep file for checksum comparison
		return "", false
	}

	return entry.Content, true
}

// GetURLChecksum returns the cached checksum for a URL, or empty string if not cached
func GetURLChecksum(url string) string {
	gokeenDir, err := GetGokeenDir()
	if err != nil {
		return ""
	}

	filename := urlToCacheFilename(url)
	filepath := path.Join(gokeenDir, filename)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return ""
	}

	var entry urlCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return ""
	}

	return entry.Checksum
}

// SetDomainValidation caches domain validation result (in-memory only)
func SetDomainValidation(domain string, valid bool) {
	c.Set(domainValidation+domain, valid, cache.NoExpiration)
}

// GetDomainValidation retrieves cached domain validation result
func GetDomainValidation(domain string) (bool, bool) {
	v, ok := c.Get(domainValidation + domain)
	if !ok {
		return false, false
	}
	return v.(bool), true
}
