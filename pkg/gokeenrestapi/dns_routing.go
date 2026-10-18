package gokeenrestapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/hashicorp/go-version"
	"github.com/noksa/gokeenapi/internal/gokeencache"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenspinner"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
)

// urlFetchResult holds the parsed result of a single URL fetch
type urlFetchResult struct {
	domains         []string
	skipped         int
	checksumChanged bool
	fromCache       bool
}

const (
	minDnsRoutingVersion = "5.0.1"
	// maxDomainsPerGroup is the router's limit for domains in a single DNS-routing object-group
	maxDomainsPerGroup = 300
)

// versionNumericRe extracts the leading numeric version (e.g. "5.1" from "5.1 Beta 4")
var versionNumericRe = regexp.MustCompile(`^[\d]+(?:\.[\d]+)*`)

var (
	// DnsRouting provides DNS-routing functionality for domain-based routing policies
	DnsRouting keeneticDnsRouting
)

type keeneticDnsRouting struct{}

// CheckDnsRoutingSupport validates that the router firmware version supports DNS-routing (>= 5.0.1)
func (*keeneticDnsRouting) CheckDnsRoutingSupport() error {
	runtime := gokeencache.GetRuntimeConfig()
	versionInfo := runtime.RouterInfo.Version

	// Use Title which contains the user-friendly version (e.g., "4.3.6.3")
	routerVersion := versionInfo.Title
	if routerVersion == "" {
		return errors.New("router version information not available. Please authenticate first")
	}

	// Strip non-numeric suffixes like "Beta 4" from "5.1 Beta 4"
	numericVersion := versionNumericRe.FindString(routerVersion)
	if numericVersion == "" {
		return fmt.Errorf("failed to parse router version '%s': no numeric version found", routerVersion)
	}

	currentVer, err := version.NewVersion(numericVersion)
	if err != nil {
		return fmt.Errorf("failed to parse router version '%s': %w", routerVersion, err)
	}

	minVer, err := version.NewVersion(minDnsRoutingVersion)
	if err != nil {
		return fmt.Errorf("failed to parse minimum version '%s': %w", minDnsRoutingVersion, err)
	}

	if currentVer.LessThan(minVer) {
		return fmt.Errorf("DNS-routing requires Keenetic firmware version %s or higher. Current version: %s", minDnsRoutingVersion, routerVersion)
	}

	return nil
}

// GetExistingDnsRoutingGroups retrieves current DNS-routing object-groups from router via REST API
// Returns a map of group names to their domain lists
func (*keeneticDnsRouting) GetExistingDnsRoutingGroups() (map[string][]string, error) {
	var groups map[string][]string

	err := gokeenspinner.WrapWithSpinner(
		fmt.Sprintf("Fetching existing %v", color.CyanString("DNS-routing groups")),
		func() error {
			var objectGroupResponse gokeenrestapimodels.ObjectGroupFqdnResponse

			body, fetchErr := Common.ExecuteGetSubPath("/rci/object-group/fqdn")
			if fetchErr != nil {
				return fetchErr
			}

			if fetchErr := json.Unmarshal(body, &objectGroupResponse); fetchErr != nil {
				return fmt.Errorf("failed to parse object-group response: %w", fetchErr)
			}

			groups = make(map[string][]string)
			for groupName, group := range objectGroupResponse {
				domains := make([]string, 0, len(group.Include))
				for _, entry := range group.Include {
					domains = append(domains, entry.Address)
				}
				groups[groupName] = domains
			}

			return nil
		},
	)

	return groups, err
}

// getExistingDnsProxyRoutes retrieves every route, including multiple routes for one group.
func getExistingDnsProxyRoutes() (gokeenrestapimodels.DnsProxyRouteResponse, error) {
	var routes gokeenrestapimodels.DnsProxyRouteResponse

	err := gokeenspinner.WrapWithSpinner(
		fmt.Sprintf("Fetching existing %v", color.CyanString("dns-proxy routes")),
		func() error {
			body, fetchErr := Common.ExecuteGetSubPath("/rci/dns-proxy/route")
			if fetchErr != nil {
				return fetchErr
			}

			if fetchErr := json.Unmarshal(body, &routes); fetchErr != nil {
				return fmt.Errorf("failed to parse dns-proxy route response: %w", fetchErr)
			}
			return nil
		},
	)

	return routes, err
}

// GetExistingDnsProxyRoutes returns a map of group names to interface IDs.
// Multiple routes for one group cannot be represented by the map, so fail safely.
func (*keeneticDnsRouting) GetExistingDnsProxyRoutes() (map[string]string, error) {
	routeList, err := getExistingDnsProxyRoutes()
	if err != nil {
		return nil, err
	}
	routes := make(map[string]string, len(routeList))
	for _, route := range routeList {
		if previous, exists := routes[route.Group]; exists {
			return nil, fmt.Errorf("multiple dns-proxy routes for group %q (%q and %q)", route.Group, previous, route.Interface)
		}
		routes[route.Group] = route.Interface
	}
	return routes, nil
}

// validateDomainWithIDNA uses the same rules as final config validation.
func validateDomainWithIDNA(domain string) (bool, string) {
	// Check cache first - but still need to recompute reason for invalid domains
	if valid, cached := gokeencache.GetDomainValidation(domain); cached {
		if valid {
			return true, ""
		}
		// For invalid cached domains, fall through to recompute the reason
	}

	_, err := config.NormalizeDnsRoutingDomain(domain)
	if err != nil {
		gokeencache.SetDomainValidation(domain, false)
		return false, err.Error()
	}

	gokeencache.SetDomainValidation(domain, true)
	return true, ""
}

// parseDomainLines parses lines of text and extracts valid domain names
// Returns the list of valid domains and count of skipped lines
func parseDomainLines(lines []string, source string) ([]string, int) {
	var domains []string
	var skipped int

	for _, line := range lines {
		// Trim whitespace
		originalLine := strings.TrimSpace(line)

		// Skip empty lines and comments
		if originalLine == "" || strings.HasPrefix(originalLine, "#") {
			continue
		}

		processedLine := originalLine

		// Handle lines with attributes (e.g., "domain.com @cn" or "domain.com attribute")
		// Take only the first field (the domain part)
		if strings.Contains(processedLine, " ") || strings.Contains(processedLine, "\t") {
			fields := strings.Fields(processedLine)
			if len(fields) > 0 {
				processedLine = fields[0]
			}
		}

		// Only full: and domain: carry a domain usable by the router. Regex,
		// keyword and include rules cannot be translated without changing meaning.
		if idx := strings.Index(processedLine, ":"); idx > 0 {
			prefix := processedLine[:idx]
			if prefix == "regexp" || prefix == "keyword" || prefix == "include" {
				skipped++
				if config.Cfg.Logs.Debug {
					gokeenlog.InfoSubStepf("Skipped unsupported domain rule from %s: %s", source, originalLine)
				}
				continue
			}
			if prefix == "full" || prefix == "domain" {
				processedLine = processedLine[idx+1:]
			}
		}

		// Validate domain name using IDNA
		// This automatically rejects invalid formats including:
		// - Bare names without TLD (youtube, instagram)
		// - Lines starting with @ or other invalid characters
		// - regexp patterns (after prefix removal, these will fail IDNA validation)
		canonical, err := config.NormalizeDnsRoutingDomain(processedLine)
		if err != nil {
			skipped++
			if config.Cfg.Logs.Debug {
				gokeenlog.InfoSubStepf("Skipped invalid domain from %s: %s (%s)", source, originalLine, err)
			}
			continue
		}

		domains = append(domains, canonical)
	}

	return domains, skipped
}

func rejectAllInvalidDomains(domains []string, skipped int, source string) error {
	if len(domains) == 0 && skipped > 0 {
		return fmt.Errorf("%s contains no usable domains: %d invalid domain or IP address entries were skipped", source, skipped)
	}
	return nil
}

// LoadDomainsFromFile reads domains from a .txt file (one domain per line)
// Supports comments (lines starting with #) and empty lines
func (*keeneticDnsRouting) LoadDomainsFromFile(filePath string) ([]string, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read domain file '%s': %w", filePath, err)
	}

	lines := strings.Split(string(b), "\n")
	domains, skipped := parseDomainLines(lines, fmt.Sprintf("file %s", filepath.Base(filePath)))
	if err := rejectAllInvalidDomains(domains, skipped, fmt.Sprintf("file '%s'", filePath)); err != nil {
		return nil, err
	}

	if skipped > 0 {
		gokeenlog.InfoSubStepf("Skipped %v invalid domain(s) from file: %v",
			color.YellowString("%d", skipped),
			color.CyanString(filepath.Base(filePath)))
	}

	return domains, nil
}

// LoadDomainsFromURL downloads a .txt file from a URL and returns the domains
// Tracks content changes via checksum and reports when remote lists are updated
func (*keeneticDnsRouting) LoadDomainsFromURL(url string) ([]string, error) {
	// Check cache first
	if cached, ok := gokeencache.GetURLContent(url); ok {
		lines := strings.Split(cached, "\n")
		domains, skipped := parseDomainLines(lines, "URL")
		if err := rejectAllInvalidDomains(domains, skipped, fmt.Sprintf("URL '%s'", url)); err != nil {
			return nil, err
		}
		gokeenlog.InfoSubStepf("Loaded %v domains from cache, URL: %v",
			color.GreenString("%d", len(domains)),
			color.CyanString(url))
		gokeenlog.HorizontalLine()
		return domains, nil
	}

	// Get previous checksum to detect changes
	previousChecksum := gokeencache.GetURLChecksum(url)

	rClient := GetURLClient() // shared client for external URLs; always verifies TLS certificates

	var domains []string
	var skipped int
	var checksumChanged bool

	err := gokeenspinner.WrapWithSpinnerAndOptions(
		fmt.Sprintf("Fetching %v url", color.CyanString(url)),
		func(opts *gokeenspinner.SpinnerOptions) error {
			response, err := rClient.R().Get(url)
			if err != nil {
				return err
			}
			if response.StatusCode() != 200 {
				return fmt.Errorf("status code %d", response.StatusCode())
			}

			content := string(response.Body())
			lines := strings.Split(content, "\n")
			domains, skipped = parseDomainLines(lines, "URL")
			if err := rejectAllInvalidDomains(domains, skipped, fmt.Sprintf("URL '%s'", url)); err != nil {
				return err
			}

			// Calculate checksum and compare with previous
			currentChecksum := fmt.Sprintf("%x", gokeencache.ComputeChecksum([]byte(content)))
			if previousChecksum != "" && previousChecksum != currentChecksum {
				checksumChanged = true
			}

			// Cache with configured TTL
			if err := gokeencache.SetURLContent(url, content, config.GetURLCacheTTL()); err != nil {
				gokeenlog.InfoSubStepf("Warning: failed to cache URL content for %v: %v", url, err)
			}

			opts.AddActionAfterSpinner(func() {
				if checksumChanged {
					gokeenlog.InfoSubStepf("Domain list updated (checksum changed): %v",
						color.YellowString(url))
				}
				if skipped > 0 {
					gokeenlog.InfoSubStepf("Skipped %v invalid domain(s)",
						color.YellowString("%d", skipped))
				}
				gokeenlog.InfoSubStepf("Loaded %v domains",
					color.GreenString("%d", len(domains)))
			})

			return nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch domain URL '%s': %w", url, err)
	}

	return domains, nil
}

// fetchURLDomainsRaw fetches and parses domains from a URL without printing anything.
// Retries up to 3 times with linear backoff on failure.
// Results are cached the same way as LoadDomainsFromURL.
func fetchURLDomainsRaw(rawURL string) (urlFetchResult, error) {
	// Serve from cache if available
	if cached, ok := gokeencache.GetURLContent(rawURL); ok {
		lines := strings.Split(cached, "\n")
		domains, skipped := parseDomainLines(lines, "URL")
		if err := rejectAllInvalidDomains(domains, skipped, fmt.Sprintf("URL '%s'", rawURL)); err != nil {
			return urlFetchResult{}, err
		}
		return urlFetchResult{domains: domains, skipped: skipped, fromCache: true}, nil
	}

	previousChecksum := gokeencache.GetURLChecksum(rawURL)
	rClient := GetURLClient()

	const maxRetries = 3
	var lastErr error
	for attempt := range maxRetries {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}

		response, err := rClient.R().Get(rawURL)
		if err != nil {
			lastErr = err
			continue
		}
		if response.StatusCode() != 200 {
			lastErr = fmt.Errorf("status code %d", response.StatusCode())
			continue
		}

		content := string(response.Body())
		lines := strings.Split(content, "\n")
		domains, skipped := parseDomainLines(lines, "URL")
		if err := rejectAllInvalidDomains(domains, skipped, fmt.Sprintf("URL '%s'", rawURL)); err != nil {
			return urlFetchResult{}, err
		}
		currentChecksum := fmt.Sprintf("%x", gokeencache.ComputeChecksum([]byte(content)))
		checksumChanged := previousChecksum != "" && previousChecksum != currentChecksum

		if err := gokeencache.SetURLContent(rawURL, content, config.GetURLCacheTTL()); err != nil {
			gokeenlog.InfoSubStepf("Warning: failed to cache URL content for %v: %v", rawURL, err)
		}

		return urlFetchResult{
			domains:         domains,
			skipped:         skipped,
			checksumChanged: checksumChanged,
			fromCache:       false,
		}, nil
	}

	return urlFetchResult{}, fmt.Errorf("failed to fetch domain URL '%s' after %d attempts: %w", rawURL, maxRetries, lastErr)
}

// prefetchURLsWithSpinner fetches all unique URLs across groups in parallel (max 4 concurrent)
// under a single shared spinner. After completion prints per-URL substeps in original order.
func prefetchURLsWithSpinner(groups []config.DnsRoutingGroup) (map[string]urlFetchResult, error) {
	// Collect unique URLs preserving declaration order
	seen := make(map[string]bool)
	var orderedURLs []string
	for _, group := range groups {
		for _, u := range group.DomainURL {
			if !seen[u] {
				seen[u] = true
				orderedURLs = append(orderedURLs, u)
			}
		}
	}

	if len(orderedURLs) == 0 {
		return nil, nil
	}

	results := make(map[string]urlFetchResult, len(orderedURLs))
	var mu sync.Mutex
	var fetchErr error

	err := gokeenspinner.WrapWithSpinnerAndOptions(
		fmt.Sprintf("Fetching %v domain URL(s)", color.CyanString("%d", len(orderedURLs))),
		func(opts *gokeenspinner.SpinnerOptions) error {
			g, _ := errgroup.WithContext(context.Background())
			g.SetLimit(4)

			for _, u := range orderedURLs {
				g.Go(func() error {
					res, err := fetchURLDomainsRaw(u)
					mu.Lock()
					defer mu.Unlock()
					if err != nil {
						fetchErr = multierr.Append(fetchErr, err)
						return nil
					}
					results[u] = res
					return nil
				})
			}

			_ = g.Wait()

			// Register substep logging — show results for all successfully fetched URLs
			opts.AddActionAfterSpinner(func() {
				for _, u := range orderedURLs {
					res, ok := results[u]
					if !ok {
						continue // failed URL — reported via returned error
					}
					if res.fromCache {
						gokeenlog.InfoSubStepf("%v: %v from cache",
							color.CyanString(u),
							color.GreenString("%d domains", len(res.domains)))
					} else {
						gokeenlog.InfoSubStepf("%v: %v",
							color.CyanString(u),
							color.GreenString("%d domains", len(res.domains)))
						if res.checksumChanged {
							gokeenlog.InfoSubStepf("  updated (checksum changed)")
						}
						if res.skipped > 0 {
							gokeenlog.InfoSubStepf("  skipped %v invalid domain(s)",
								color.YellowString("%d", res.skipped))
						}
					}
				}
			})

			return fetchErr
		},
	)

	if err != nil {
		return nil, err
	}
	return results, nil
}

// This prevents routing conflicts where the same domain would be routed through different interfaces
func validateNoDuplicateDomainsAcrossGroups(groupDomains map[string][]string) {
	domainToGroups := make(map[string][]string)
	for groupName, domains := range groupDomains {
		for _, domain := range domains {
			domainToGroups[domain] = append(domainToGroups[domain], groupName)
		}
	}

	var duplicates []string
	for domain, groupNames := range domainToGroups {
		if len(groupNames) > 1 {
			duplicates = append(duplicates, domain)
		}
	}

	if len(duplicates) > 0 {
		slices.Sort(duplicates)
		gokeenlog.Infof("%s: domains cannot appear in multiple groups", color.RedString("Misconfiguration found"))
		for _, domain := range duplicates {
			groupNames := domainToGroups[domain]
			gokeenlog.InfoSubStepf("%s appears in groups: %s",
				color.YellowString(domain),
				color.CyanString(strings.Join(groupNames, ", ")))
		}
		gokeenlog.Info("Each domain must belong to exactly one DNS-routing group to avoid routing conflicts")
		gokeenlog.Info("Continue anyway - but keep in mind that it should be fixed")
		gokeenlog.HorizontalLine()
	}
}

// splitDnsRoutingGroups gives each sorted chunk a stable name. The configured
// name remains the first group so existing routes survive the first split.
func splitDnsRoutingGroups(groups []config.DnsRoutingGroup, domainsByGroup map[string][]string) ([]config.DnsRoutingGroup, map[string][]string, error) {
	configured := make(map[string]bool, len(groups))
	for _, group := range groups {
		configured[group.Name] = true
	}
	for _, group := range groups {
		for name := range configured {
			if name != group.Name && splitGroupIndex(group.Name, name) >= 2 {
				return nil, nil, fmt.Errorf("group '%s': name '%s' is reserved for an automatic split", group.Name, name)
			}
		}
	}

	var expanded []config.DnsRoutingGroup
	chunks := make(map[string][]string)
	for _, group := range groups {
		domains, ok := domainsByGroup[group.Name]
		if !ok {
			continue // An empty source is skipped, as before.
		}
		for start, index := 0, 1; start < len(domains); start, index = start+maxDomainsPerGroup, index+1 {
			name := group.Name
			if index > 1 {
				name = fmt.Sprintf("%s-%d", group.Name, index)
			}
			expanded = append(expanded, config.DnsRoutingGroup{Name: name, InterfaceID: group.InterfaceID})
			chunks[name] = domains[start:min(start+maxDomainsPerGroup, len(domains))]
		}
	}
	return expanded, chunks, nil
}

func splitGroupIndex(base, name string) int {
	suffix, ok := strings.CutPrefix(name, base+"-")
	if !ok {
		return 0
	}
	index, err := strconv.Atoi(suffix)
	if err != nil || index < 2 || strconv.Itoa(index) != suffix {
		return 0
	}
	return index
}

// AddDnsRoutingGroups creates object-groups and dns-proxy routes for the specified groups
// This function is idempotent - it only creates groups/domains/routes that don't already exist
func (*keeneticDnsRouting) AddDnsRoutingGroups(groups []config.DnsRoutingGroup) error {
	if len(groups) == 0 {
		gokeenlog.Info("No DNS-routing groups to add")
		return nil
	}

	// Validate configuration
	if err := config.ValidateDnsRoutingGroups(groups); err != nil {
		return err
	}

	// Check router version support
	if err := DnsRouting.CheckDnsRoutingSupport(); err != nil {
		return err
	}

	// Fetch interfaces once for all validations
	interfaces, err := Interface.GetInterfacesViaRciShowInterfaces(false)
	if err != nil {
		return fmt.Errorf("failed to fetch interfaces: %w", err)
	}

	// Validate all interfaces exist before generating commands
	for _, group := range groups {
		if err := Checks.CheckInterfaceId(group.InterfaceID); err != nil {
			return fmt.Errorf("group '%s': %w", group.Name, err)
		}
		// Check if interface exists in the fetched list
		if _, exists := interfaces[group.InterfaceID]; !exists {
			return fmt.Errorf("group '%s': interface '%s' not found", group.Name, group.InterfaceID)
		}
	}

	// Prefetch all URLs in parallel under a single spinner, then use results below
	prefetched, prefetchErr := prefetchURLsWithSpinner(groups)
	if prefetchErr != nil {
		return fmt.Errorf("failed to prefetch domain URLs: %w", prefetchErr)
	}

	// Load domains from files and URLs for each group
	var mErr error
	groupDomains := make(map[string][]string)

	for _, group := range groups {
		var allDomains []string

		// Load domains from local files
		// Paths are already resolved (absolute) during config loading
		for _, file := range group.DomainFile {
			domains, err := DnsRouting.LoadDomainsFromFile(file)
			if err != nil {
				mErr = multierr.Append(mErr, fmt.Errorf("group '%s': %w", group.Name, err))
				continue
			}
			allDomains = append(allDomains, domains...)
		}

		// Use prefetched URL results (already fetched in parallel above)
		for _, url := range group.DomainURL {
			res, ok := prefetched[url]
			if !ok {
				mErr = multierr.Append(mErr, fmt.Errorf("group '%s': missing prefetch result for URL '%s'", group.Name, url))
				continue
			}
			allDomains = append(allDomains, res.domains...)
		}

		if len(allDomains) == 0 {
			gokeenlog.InfoSubStepf("Skipping group '%s': no domains loaded", group.Name)
			continue
		}

		// Deduplicate domains (in case same domain appears in multiple files/URLs)
		// Sort first, then use Compact to remove consecutive duplicates
		originalCount := len(allDomains)
		slices.Sort(allDomains)
		allDomains = slices.Compact(allDomains)
		if duplicates := originalCount - len(allDomains); duplicates > 0 {
			if config.Cfg.Logs.Debug {
				gokeenlog.InfoSubStepf("Removed %v duplicate domain(s) from group %v (deduplicated across sources in memory, not a router change)",
					color.YellowString("%d", duplicates),
					color.CyanString(group.Name))
			}
		}

		// Validate loaded domains
		if err := config.ValidateDomainList(allDomains, group.Name); err != nil {
			mErr = multierr.Append(mErr, err)
			continue
		}

		groupDomains[group.Name] = allDomains
	}

	// If there were errors loading domains, return them
	if mErr != nil {
		return mErr
	}

	// Validate no domain appears in multiple groups (configuration error)
	validateNoDuplicateDomainsAcrossGroups(groupDomains)
	groupsToApply, splitDomains, err := splitDnsRoutingGroups(groups, groupDomains)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if len(groupDomains[group.Name]) > maxDomainsPerGroup {
			var parts []string
			for _, chunk := range groupsToApply {
				if chunk.Name == group.Name || splitGroupIndex(group.Name, chunk.Name) >= 2 {
					parts = append(parts, fmt.Sprintf("%s (%d)", chunk.Name, len(splitDomains[chunk.Name])))
				}
			}
			gokeenlog.InfoSubStepf("Split group %s: %s", group.Name, strings.Join(parts, ", "))
		}
	}

	// Get existing groups from router to make operation idempotent

	existingGroups, err := DnsRouting.GetExistingDnsRoutingGroups()
	if err != nil {
		return fmt.Errorf("failed to get existing DNS-routing groups: %w", err)
	}

	// Get existing dns-proxy routes to check what needs to be added
	existingRoutes, err := DnsRouting.GetExistingDnsProxyRoutes()
	if err != nil {
		return fmt.Errorf("failed to get existing dns-proxy routes: %w", err)
	}

	stale, err := inspectExistingSplits(groups, groupDomains, splitDomains, existingGroups, existingRoutes)
	if err != nil {
		return err
	}

	var parseSlice []gokeenrestapimodels.ParseRequest

	// Track changes for summary
	domainsToRemove := 0
	domainsToAdd := 0

	// Generate commands for each group
	// Order: object-group creation, domain cleanup (remove unwanted), domain adds, then dns-proxy routes
	for _, group := range groupsToApply {
		domains := splitDomains[group.Name]
		existingDomains, groupExists := existingGroups[group.Name]

		// Create object-group only if it doesn't exist
		if !groupExists {
			parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
				Parse: fmt.Sprintf("object-group fqdn %s", group.Name),
			})
		}

		// If group exists, remove domains that shouldn't be there (cleanup)
		if groupExists {
			for _, existingDomain := range existingDomains {
				shouldKeep := slices.Contains(domains, existingDomain)
				// Remove domain if it's not in the config
				if !shouldKeep {
					domainsToRemove++
					gokeenlog.InfoSubStepf("Removing domain %v from group %v",
						color.RedString(existingDomain),
						color.YellowString(group.Name))
					parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
						Parse: fmt.Sprintf("no object-group fqdn %s include %s", group.Name, existingDomain),
					})
				}
			}
		}

		// Add domain includes only for domains that don't already exist
		for _, domain := range domains {
			domainExists := false
			if groupExists {
				if slices.Contains(existingDomains, domain) {
					domainExists = true
				}
			}

			if !domainExists {
				domainsToAdd++
				parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
					Parse: fmt.Sprintf("object-group fqdn %s include %s", group.Name, domain),
				})
			}
		}
	}

	// Add dns-proxy routes after all object-groups are created
	// Only add routes that don't already exist or have different interface
	for _, group := range groupsToApply {
		existingInterface, routeExists := existingRoutes[group.Name]
		// Add route if it doesn't exist or if the interface changed
		if !routeExists || existingInterface != group.InterfaceID {
			if routeExists {
				parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
					Parse: fmt.Sprintf("no dns-proxy route object-group %s %s", group.Name, existingInterface),
				})
			}
			parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
				Parse: fmt.Sprintf("dns-proxy route object-group %s %s auto", group.Name, group.InterfaceID),
			})
		}
	}
	// Remove overflow groups that are no longer needed after shrinking a source.
	// Routes go first, then their object-groups.
	for _, group := range stale {
		gokeenlog.InfoSubStepf("Removing obsolete split group %s", group.Name)
		parseSlice = append(parseSlice,
			gokeenrestapimodels.ParseRequest{Parse: fmt.Sprintf("no dns-proxy route object-group %s %s", group.Name, group.InterfaceID)},
			gokeenrestapimodels.ParseRequest{Parse: fmt.Sprintf("no object-group fqdn %s", group.Name)},
		)
	}

	// If no commands to execute, we're done
	if len(parseSlice) == 0 {
		gokeenlog.Info("All DNS-routing groups and domains are up to date")
		return nil
	}

	// Log summary of changes
	if domainsToRemove > 0 || domainsToAdd > 0 {
		gokeenlog.InfoSubStepf("Changes: %v domains to add, %v domains to remove",
			color.GreenString("%d", domainsToAdd),
			color.RedString("%d", domainsToRemove))
		gokeenlog.HorizontalLine()
	}

	// Ensure save config is at the end
	parseSlice = Common.EnsureSaveConfigAtEnd(parseSlice)

	var parseResponse []gokeenrestapimodels.ParseResponse
	err = gokeenspinner.WrapWithSpinnerAndOptions(
		fmt.Sprintf("Applying %v DNS-routing groups", color.CyanString("%d", len(groupsToApply))),
		func(opts *gokeenspinner.SpinnerOptions) error {
			var executeErr error
			parseResponse, executeErr = Common.ExecutePostParse(parseSlice...)

			opts.AddActionAfterSpinner(func() {
				gokeenlog.PrintParseResponse(parseResponse)
				if executeErr == nil {
					gokeenlog.InfoSubStepf("Successfully applied %v DNS-routing groups", len(groupsToApply))
				}
			})

			return executeErr
		},
	)

	return err
}

// DeleteAllDnsRoutingGroups removes every dns-proxy route and object-group currently on the router.
// It fetches the live state and deletes whatever is present, regardless of the config file.
// This is the DNS-routing equivalent of Ip.DeleteAllRoutes.
func (d *keeneticDnsRouting) DeleteAllDnsRoutingGroups() error {
	// Check router version support
	if err := d.CheckDnsRoutingSupport(); err != nil {
		return err
	}

	// Fetch everything currently on the router
	existingGroups, err := d.GetExistingDnsRoutingGroups()
	if err != nil {
		return fmt.Errorf("failed to get existing DNS-routing groups: %w", err)
	}

	existingRoutes, err := getExistingDnsProxyRoutes()
	if err != nil {
		return fmt.Errorf("failed to get existing dns-proxy routes: %w", err)
	}

	if len(existingGroups) == 0 && len(existingRoutes) == 0 {
		gokeenlog.Info("No DNS-routing groups found on router")
		return nil
	}

	// Remove all live routes, including orphan routes and multiple routes per group.
	groups := make([]config.DnsRoutingGroup, 0, len(existingGroups))
	for groupName := range existingGroups {
		groups = append(groups, config.DnsRoutingGroup{Name: groupName})
	}
	return deleteDnsRoutingGroups(groups, existingRoutes)
}

// DeleteDnsRoutingGroups removes dns-proxy routes and object-groups for the specified groups
func (*keeneticDnsRouting) DeleteDnsRoutingGroups(groups []config.DnsRoutingGroup) error {
	if len(groups) == 0 {
		gokeenlog.Info("No DNS-routing groups to delete")
		return nil
	}
	if err := DnsRouting.CheckDnsRoutingSupport(); err != nil {
		return err
	}
	// A group can have multiple routes, or its live interface may differ from
	// the caller's snapshot. Remove every route that still references it.
	existingRoutes, err := getExistingDnsProxyRoutes()
	if err != nil {
		return fmt.Errorf("failed to get existing dns-proxy routes: %w", err)
	}
	names := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		names[group.Name] = struct{}{}
	}
	routes := make(gokeenrestapimodels.DnsProxyRouteResponse, 0, len(existingRoutes))
	for _, route := range existingRoutes {
		if _, selected := names[route.Group]; selected {
			routes = append(routes, route)
		}
	}
	return deleteDnsRoutingGroups(groups, routes)
}

func deleteDnsRoutingGroups(groups []config.DnsRoutingGroup, routes gokeenrestapimodels.DnsProxyRouteResponse) error {
	if len(groups) == 0 && len(routes) == 0 {
		gokeenlog.Info("No DNS-routing groups to delete")
		return nil
	}

	// Check router version support
	if err := DnsRouting.CheckDnsRoutingSupport(); err != nil {
		return err
	}

	var parseSlice []gokeenrestapimodels.ParseRequest

	// Generate deletion commands
	// Order: dns-proxy routes first, then object-groups
	for _, route := range routes {
		parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
			Parse: fmt.Sprintf("no dns-proxy route object-group %s %s", route.Group, route.Interface),
		})
	}

	// Delete object-groups after dns-proxy routes
	for _, group := range groups {
		parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
			Parse: fmt.Sprintf("no object-group fqdn %s", group.Name),
		})
	}

	// Ensure save config is at the end
	parseSlice = Common.EnsureSaveConfigAtEnd(parseSlice)

	var parseResponse []gokeenrestapimodels.ParseResponse
	err := gokeenspinner.WrapWithSpinnerAndOptions(
		fmt.Sprintf("Deleting DNS-routing state (groups: %v, routes: %v)", color.CyanString("%d", len(groups)), color.CyanString("%d", len(routes))),
		func(opts *gokeenspinner.SpinnerOptions) error {
			var executeErr error
			parseResponse, executeErr = Common.ExecutePostParse(parseSlice...)

			opts.AddActionAfterSpinner(func() {
				gokeenlog.PrintParseResponse(parseResponse)
				if executeErr == nil {
					gokeenlog.InfoSubStepf("Deleted DNS-routing state: groups: %v, routes: %v", len(groups), len(routes))
				}
			})

			return executeErr
		},
	)

	return err
}
