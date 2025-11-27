package gokeenrestapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/go-version"
	"github.com/noksa/gokeenapi/internal/gokeencache"
	"github.com/noksa/gokeenapi/internal/gokeenlog"
	"github.com/noksa/gokeenapi/internal/gokeenspinner"
	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"go.uber.org/multierr"
)

const (
	minDnsRoutingVersion = "5.0.1"
)

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

	currentVer, err := version.NewVersion(routerVersion)
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

			body, fetchErr := Common.ExecuteGetSubPath("/rci/show/object-group/fqdn")
			if fetchErr != nil {
				return fetchErr
			}

			if fetchErr := json.Unmarshal(body, &objectGroupResponse); fetchErr != nil {
				return fmt.Errorf("failed to parse object-group response: %w", fetchErr)
			}

			groups = make(map[string][]string)
			for _, group := range objectGroupResponse.Group {
				domains := make([]string, 0, len(group.Entry))
				for _, entry := range group.Entry {
					domains = append(domains, entry.Fqdn)
				}
				groups[group.GroupName] = domains
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

// LoadDomainsFromFile reads domains from a .txt file (one domain per line)
// Supports comments (lines starting with #) and empty lines
func (*keeneticDnsRouting) LoadDomainsFromFile(filePath string) ([]string, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read domain file '%s': %w", filePath, err)
	}

	var domains []string
	lines := strings.SplitSeq(string(b), "\n")
	for line := range lines {
		// Trim whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		domains = append(domains, line)
	}

	return domains, nil
}

// LoadDomainsFromURL downloads a .txt file from a URL and returns the domains
func (*keeneticDnsRouting) LoadDomainsFromURL(url string) ([]string, error) {
	rClient := resty.New()
	rClient.SetDisableWarn(true)
	rClient.SetTimeout(time.Second * 5)

	var response *resty.Response
	var err error

	err = gokeenspinner.WrapWithSpinner(
		fmt.Sprintf("Fetching %v url", color.CyanString(url)),
		func() error {
			response, err = rClient.R().Get(url)
			return err
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch domain URL '%s': %w", url, err)
	}

	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to fetch domain URL '%s': status code %d", url, response.StatusCode())
	}

	var domains []string
	lines := strings.SplitSeq(string(response.Body()), "\n")
	for line := range lines {
		// Trim whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		domains = append(domains, line)
	}

	return domains, nil
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

	// Validate all interfaces exist before generating commands
	for _, group := range groups {
		if err := Checks.CheckInterfaceId(group.InterfaceID); err != nil {
			return fmt.Errorf("group '%s': %w", group.Name, err)
		}
		if err := Checks.CheckInterfaceExists(group.InterfaceID); err != nil {
			return fmt.Errorf("group '%s': %w", group.Name, err)
		}
	}

	// Load domains from files and URLs for each group
	var mErr error
	groupDomains := make(map[string][]string)

	for _, group := range groups {
		var allDomains []string

		// Load domains from local files (YAML expansion already done at config load time)
		for _, file := range group.DomainFile {
			absFilePath, err := filepath.Abs(file)
			if err != nil {
				mErr = multierr.Append(mErr, fmt.Errorf("group '%s': failed to resolve file path '%s': %w", group.Name, file, err))
				continue
			}

			domains, err := DnsRouting.LoadDomainsFromFile(absFilePath)
			if err != nil {
				mErr = multierr.Append(mErr, fmt.Errorf("group '%s': %w", group.Name, err))
				continue
			}
			allDomains = append(allDomains, domains...)
		}

		// Load domains from URLs
		for _, url := range group.DomainURL {
			domains, err := DnsRouting.LoadDomainsFromURL(url)
			if err != nil {
				mErr = multierr.Append(mErr, fmt.Errorf("group '%s': %w", group.Name, err))
				continue
			}
			allDomains = append(allDomains, domains...)
		}

		if len(allDomains) == 0 {
			gokeenlog.InfoSubStepf("Skipping group '%s': no domains loaded", group.Name)
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
		gokeenlog.Info("All DNS-routing groups and domains already exist")
		return nil
	}

	// Ensure save config is at the end
	parseSlice = Common.EnsureSaveConfigAtEnd(parseSlice)

	var parseResponse []gokeenrestapimodels.ParseResponse
	err = gokeenspinner.WrapWithSpinner(
		fmt.Sprintf("Applying %v DNS-routing groups", color.CyanString("%d", len(groups))),
		func() error {
			var executeErr error
			parseResponse, executeErr = Common.ExecutePostParse(parseSlice...)
			return executeErr
		},
	)

	gokeenlog.PrintParseResponse(parseResponse)

	if err == nil {
		gokeenlog.InfoSubStepf("Successfully applied %v DNS-routing groups", len(groups))
	}

	return err
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
	err := gokeenspinner.WrapWithSpinner(
		fmt.Sprintf("Deleting %v DNS-routing groups", color.CyanString("%d", len(groups))),
		func() error {
			var executeErr error
			parseResponse, executeErr = Common.ExecutePostParse(parseSlice...)
			return executeErr
		},
	)

	gokeenlog.PrintParseResponse(parseResponse)

	if err == nil {
		gokeenlog.InfoSubStepf("Successfully deleted %v DNS-routing groups", len(groups))
	}

	return err
}
