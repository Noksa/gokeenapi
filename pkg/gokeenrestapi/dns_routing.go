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
	"golang.org/x/net/idna"
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

// GetExistingDnsProxyRoutes retrieves current dns-proxy routes from router via REST API
// Returns a map of group names to their interface IDs
func (*keeneticDnsRouting) GetExistingDnsProxyRoutes() (map[string]string, error) {
	var routes map[string]string

	err := gokeenspinner.WrapWithSpinner(
		fmt.Sprintf("Fetching existing %v", color.CyanString("dns-proxy routes")),
		func() error {
			var dnsProxyRoutes gokeenrestapimodels.DnsProxyRouteResponse

			body, fetchErr := Common.ExecuteGetSubPath("/rci/dns-proxy/route")
			if fetchErr != nil {
				return fetchErr
			}

			if fetchErr := json.Unmarshal(body, &dnsProxyRoutes); fetchErr != nil {
				return fmt.Errorf("failed to parse dns-proxy route response: %w", fetchErr)
			}

			routes = make(map[string]string)
			for _, route := range dnsProxyRoutes {
				routes[route.Group] = route.Interface
			}

			return nil
		},
	)

	return routes, err
}

// validateDomainWithIDNA validates a domain name using IDNA (Internationalized Domain Names in Applications)
// This handles both ASCII and internationalized domain names correctly
// Returns true if valid, false otherwise. If debug logging is enabled, logs the reason for rejection.
// Results are cached in memory to avoid redundant IDNA lookups.
func validateDomainWithIDNA(domain string) (bool, string) {
	// Check cache first - but still need to recompute reason for invalid domains
	if valid, cached := gokeencache.GetDomainValidation(domain); cached {
		if valid {
			return true, ""
		}
		// For invalid cached domains, fall through to recompute the reason
	}

	// Skip empty domains
	if domain == "" {
		gokeencache.SetDomainValidation(domain, false)
		return false, "empty domain"
	}

	// Domain must contain at least one dot (have a TLD)
	// This rejects bare names like "youtube", "instagram"
	if !strings.Contains(domain, ".") {
		gokeencache.SetDomainValidation(domain, false)
		return false, "missing TLD (no dot)"
	}

	// Try to convert to ASCII using IDNA
	// This validates the domain structure and handles internationalized domains
	_, err := idna.Lookup.ToASCII(domain)
	if err != nil {
		gokeencache.SetDomainValidation(domain, false)
		return false, fmt.Sprintf("IDNA validation failed: %v", err)
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

		// Strip v2fly domain-list-community prefixes (full:, regexp:, domain:, keyword:, include:)
		// These are routing rule modifiers that aren't valid domain names
		// Example: "full:d1unuk07s6td74.cloudfront.net" -> "d1unuk07s6td74.cloudfront.net"
		if idx := strings.Index(processedLine, ":"); idx > 0 {
			prefix := processedLine[:idx]
			// Check if it's a known v2fly prefix
			if prefix == "full" || prefix == "regexp" || prefix == "domain" || prefix == "keyword" || prefix == "include" {
				processedLine = processedLine[idx+1:]
			}
		}

		// Validate domain name using IDNA
		// This automatically rejects invalid formats including:
		// - Bare names without TLD (youtube, instagram)
		// - Lines starting with @ or other invalid characters
		// - regexp patterns (after prefix removal, these will fail IDNA validation)
		valid, reason := validateDomainWithIDNA(processedLine)
		if !valid {
			skipped++
			if config.Cfg.Logs.Debug {
				gokeenlog.InfoSubStepf("Skipped invalid domain from %s: %s (%s)", source, originalLine, reason)
			}
			continue
		}

		domains = append(domains, processedLine)
	}

	return domains, skipped
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
		domains, _ := parseDomainLines(lines, "URL")
		gokeenlog.InfoSubStepf("Loaded %v domains from cache, URL: %v",
			color.GreenString("%d", len(domains)),
			color.CyanString(url))
		gokeenlog.HorizontalLine()
		return domains, nil
	}

	// Get previous checksum to detect changes
	previousChecksum := gokeencache.GetURLChecksum(url)

	rClient := GetURLClient() // shared client (see GetURLClient in common.go) - avoids per-call resty.New, honors TLS skip-verify

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

			// Calculate checksum and compare with previous
			currentChecksum := fmt.Sprintf("%x", gokeencache.ComputeChecksum([]byte(content)))
			if previousChecksum != "" && previousChecksum != currentChecksum {
				checksumChanged = true
			}

			// Cache with configured TTL
			if err := gokeencache.SetURLContent(url, content, config.GetURLCacheTTL()); err != nil {
				gokeenlog.InfoSubStepf("Warning: failed to cache URL content for %v: %v", url, err)
			}

			lines := strings.Split(content, "\n")
			domains, skipped = parseDomainLines(lines, "URL")

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
		currentChecksum := fmt.Sprintf("%x", gokeencache.ComputeChecksum([]byte(content)))
		checksumChanged := previousChecksum != "" && previousChecksum != currentChecksum

		if err := gokeencache.SetURLContent(rawURL, content, config.GetURLCacheTTL()); err != nil {
			gokeenlog.InfoSubStepf("Warning: failed to cache URL content for %v: %v", rawURL, err)
		}

		lines := strings.Split(content, "\n")
		domains, skipped := parseDomainLines(lines, "URL")

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

	err := gokeenspinner.WrapWithSpinnerAndOptions(
		fmt.Sprintf("Fetching %v domain URL(s)", color.CyanString("%d", len(orderedURLs))),
		func(opts *gokeenspinner.SpinnerOptions) error {
			g, _ := errgroup.WithContext(context.Background())
			g.SetLimit(4)

			for _, u := range orderedURLs {
				g.Go(func() error {
					res, err := fetchURLDomainsRaw(u)
					if err != nil {
						return err
					}
					mu.Lock()
					results[u] = res
					mu.Unlock()
					return nil
				})
			}

			fetchErr := g.Wait()

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

		// Check router limit: maximum domains per group
		if len(allDomains) > maxDomainsPerGroup {
			mErr = multierr.Append(mErr, fmt.Errorf("group '%s': exceeds router limit of %d domains (has %d domains)", group.Name, maxDomainsPerGroup, len(allDomains)))
			continue
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

	var parseSlice []gokeenrestapimodels.ParseRequest

	// Track changes for summary
	domainsToRemove := 0
	domainsToAdd := 0

	// Generate commands for each group
	// Order: object-group creation, domain cleanup (remove unwanted), domain adds, then dns-proxy routes
	for _, group := range groups {
		domains := groupDomains[group.Name]
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
	for _, group := range groups {
		existingInterface, routeExists := existingRoutes[group.Name]
		// Add route if it doesn't exist or if the interface changed
		if !routeExists || existingInterface != group.InterfaceID {
			parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
				Parse: fmt.Sprintf("dns-proxy route object-group %s %s auto", group.Name, group.InterfaceID),
			})
		}
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
		fmt.Sprintf("Applying %v DNS-routing groups", color.CyanString("%d", len(groups))),
		func(opts *gokeenspinner.SpinnerOptions) error {
			var executeErr error
			parseResponse, executeErr = Common.ExecutePostParse(parseSlice...)

			opts.AddActionAfterSpinner(func() {
				gokeenlog.PrintParseResponse(parseResponse)
				if executeErr == nil {
					gokeenlog.InfoSubStepf("Successfully applied %v DNS-routing groups", len(groups))
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

	existingRoutes, err := d.GetExistingDnsProxyRoutes()
	if err != nil {
		return fmt.Errorf("failed to get existing dns-proxy routes: %w", err)
	}

	if len(existingGroups) == 0 {
		gokeenlog.Info("No DNS-routing groups found on router")
		return nil
	}

	// Build config.DnsRoutingGroup slice from live router state
	groups := make([]config.DnsRoutingGroup, 0, len(existingGroups))
	for groupName := range existingGroups {
		interfaceID := existingRoutes[groupName] // empty string is fine — DeleteDnsRoutingGroups handles it
		groups = append(groups, config.DnsRoutingGroup{
			Name:        groupName,
			InterfaceID: interfaceID,
		})
	}

	return d.DeleteDnsRoutingGroups(groups)
}

// DeleteDnsRoutingGroups removes dns-proxy routes and object-groups for the specified groups
func (*keeneticDnsRouting) DeleteDnsRoutingGroups(groups []config.DnsRoutingGroup) error {
	if len(groups) == 0 {
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
	for _, group := range groups {
		parseSlice = append(parseSlice, gokeenrestapimodels.ParseRequest{
			Parse: fmt.Sprintf("no dns-proxy route object-group %s %s", group.Name, group.InterfaceID),
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
		fmt.Sprintf("Deleting %v DNS-routing groups", color.CyanString("%d", len(groups))),
		func(opts *gokeenspinner.SpinnerOptions) error {
			var executeErr error
			parseResponse, executeErr = Common.ExecutePostParse(parseSlice...)

			opts.AddActionAfterSpinner(func() {
				gokeenlog.PrintParseResponse(parseResponse)
				if executeErr == nil {
					gokeenlog.InfoSubStepf("Successfully deleted %v DNS-routing groups", len(groups))
				}
			})

			return executeErr
		},
	)

	return err
}
