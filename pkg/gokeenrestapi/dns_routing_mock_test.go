package gokeenrestapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noksa/gokeenapi/pkg/config"
	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleParse_DnsRoutingObjectGroup verifies that creating DNS-routing object-groups works
func TestHandleParse_DnsRoutingObjectGroup(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Create object-group
	requests := []gokeenrestapimodels.ParseRequest{
		{Parse: "object-group fqdn social-media"},
	}

	body, err := json.Marshal(requests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	require.NoError(t, err)

	// Verify success response
	require.Len(t, responses, 1)
	require.Len(t, responses[0].Parse.Status, 1)
	assert.Equal(t, "ok", responses[0].Parse.Status[0].Status)
	assert.Contains(t, responses[0].Parse.Status[0].Message, "social-media")
}

// TestHandleParse_DnsRoutingAddDomain verifies that adding domains to object-groups works
func TestHandleParse_DnsRoutingAddDomain(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Create object-group first
	createRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "object-group fqdn social-media"},
	}

	body, err := json.Marshal(createRequests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	_ = resp.Body.Close()

	// Add domain to group
	addDomainRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "object-group fqdn social-media include facebook.com"},
	}

	body, err = json.Marshal(addDomainRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	require.NoError(t, err)

	// Verify success response
	require.Len(t, responses, 1)
	require.Len(t, responses[0].Parse.Status, 1)
	assert.Equal(t, "ok", responses[0].Parse.Status[0].Status)
	assert.Contains(t, responses[0].Parse.Status[0].Message, "facebook.com")
}

// TestHandleParse_DnsRoutingCreateRoute verifies that creating dns-proxy routes works
func TestHandleParse_DnsRoutingCreateRoute(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Create object-group first
	createRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "object-group fqdn social-media"},
		{Parse: "dns-proxy route object-group social-media Wireguard0 auto"},
	}

	body, err := json.Marshal(createRequests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	require.NoError(t, err)

	// Verify both responses are successful
	require.Len(t, responses, 2)
	assert.Equal(t, "ok", responses[0].Parse.Status[0].Status)
	assert.Equal(t, "ok", responses[1].Parse.Status[0].Status)
	assert.Contains(t, responses[1].Parse.Status[0].Message, "social-media")
	assert.Contains(t, responses[1].Parse.Status[0].Message, "Wireguard0")
}

// TestHandleParse_DnsRoutingDeleteRoute verifies that deleting dns-proxy routes works
func TestHandleParse_DnsRoutingDeleteRoute(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Create object-group and route first
	createRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "object-group fqdn social-media"},
		{Parse: "dns-proxy route object-group social-media Wireguard0 auto"},
	}

	body, err := json.Marshal(createRequests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	_ = resp.Body.Close()

	// Delete the route
	deleteRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "no dns-proxy route object-group social-media Wireguard0"},
	}

	body, err = json.Marshal(deleteRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	require.NoError(t, err)

	// Verify success response
	require.Len(t, responses, 1)
	require.Len(t, responses[0].Parse.Status, 1)
	assert.Equal(t, "ok", responses[0].Parse.Status[0].Status)
}

// TestHandleParse_DnsRoutingDeleteObjectGroup verifies that deleting object-groups works
func TestHandleParse_DnsRoutingDeleteObjectGroup(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Create object-group first
	createRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "object-group fqdn social-media"},
	}

	body, err := json.Marshal(createRequests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	_ = resp.Body.Close()

	// Delete the object-group
	deleteRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "no object-group fqdn social-media"},
	}

	body, err = json.Marshal(deleteRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	require.NoError(t, err)

	// Verify success response
	require.Len(t, responses, 1)
	require.Len(t, responses[0].Parse.Status, 1)
	assert.Equal(t, "ok", responses[0].Parse.Status[0].Status)
}

// TestHandleParse_DnsRoutingFullWorkflow verifies the complete DNS-routing workflow
func TestHandleParse_DnsRoutingFullWorkflow(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Create object-group, add domains, create route
	createRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "object-group fqdn social-media"},
		{Parse: "object-group fqdn social-media include facebook.com"},
		{Parse: "object-group fqdn social-media include instagram.com"},
		{Parse: "object-group fqdn social-media include twitter.com"},
		{Parse: "dns-proxy route object-group social-media Wireguard0 auto"},
	}

	body, err := json.Marshal(createRequests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	require.NoError(t, err)

	// Verify all responses are successful
	require.Len(t, responses, 5)
	for i, response := range responses {
		assert.Equal(t, "ok", response.Parse.Status[0].Status, "Response %d should be successful", i)
	}

	// Verify running config includes DNS-routing configuration
	resp, err = http.Get(server.URL + "/rci/show/running-config")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var runningConfig gokeenrestapimodels.RunningConfig
	err = json.NewDecoder(resp.Body).Decode(&runningConfig)
	require.NoError(t, err)

	// Check that running config contains our DNS-routing entries
	configStr := strings.Join(runningConfig.Message, "\n")
	assert.Contains(t, configStr, "object-group fqdn social-media")
	assert.Contains(t, configStr, "include facebook.com")
	assert.Contains(t, configStr, "include instagram.com")
	assert.Contains(t, configStr, "include twitter.com")
	assert.Contains(t, configStr, "dns-proxy route object-group social-media Wireguard0 auto")
}

// TestHandleParse_DnsRoutingInvalidInterface verifies error handling for non-existent interfaces
func TestHandleParse_DnsRoutingInvalidInterface(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Create object-group first
	createRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "object-group fqdn social-media"},
		{Parse: "dns-proxy route object-group social-media NonExistentInterface auto"},
	}

	body, err := json.Marshal(createRequests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	require.NoError(t, err)

	// First response should be successful (object-group creation)
	require.Len(t, responses, 2)
	assert.Equal(t, "ok", responses[0].Parse.Status[0].Status)

	// Second response should be an error (invalid interface)
	assert.Equal(t, "error", responses[1].Parse.Status[0].Status)
	assert.Contains(t, responses[1].Parse.Status[0].Message, "does not exist")
}

// TestHandleParse_DnsRoutingNonExistentGroup verifies error handling for non-existent groups
func TestHandleParse_DnsRoutingNonExistentGroup(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Try to add domain to non-existent group
	requests := []gokeenrestapimodels.ParseRequest{
		{Parse: "object-group fqdn nonexistent include facebook.com"},
	}

	body, err := json.Marshal(requests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	require.NoError(t, err)

	// Should return error
	require.Len(t, responses, 1)
	assert.Equal(t, "error", responses[0].Parse.Status[0].Status)
	assert.Contains(t, responses[0].Parse.Status[0].Message, "does not exist")
}

// TestWithDnsRoutingGroups verifies the WithDnsRoutingGroups option works
func TestWithDnsRoutingGroups(t *testing.T) {
	groups := []MockDnsRoutingGroup{
		{
			Name:    "test-group",
			Domains: []string{"example.com", "test.com"},
		},
	}
	routes := []MockDnsProxyRoute{
		{
			GroupName:   "test-group",
			InterfaceID: "Wireguard0",
			Mode:        "auto",
		},
	}

	server := NewMockRouterServer(WithDnsRoutingGroups(groups, routes))
	defer server.Close()

	// Verify running config includes the pre-configured DNS-routing
	resp, err := http.Get(server.URL + "/rci/show/running-config")
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var runningConfig gokeenrestapimodels.RunningConfig
	err = json.NewDecoder(resp.Body).Decode(&runningConfig)
	require.NoError(t, err)

	configStr := strings.Join(runningConfig.Message, "\n")
	assert.Contains(t, configStr, "object-group fqdn test-group")
	assert.Contains(t, configStr, "include example.com")
	assert.Contains(t, configStr, "include test.com")
	assert.Contains(t, configStr, "dns-proxy route object-group test-group Wireguard0 auto")
}

// TestHandleParse_DnsRoutingIdempotent verifies that adding the same groups twice is idempotent
func TestHandleParse_DnsRoutingIdempotent(t *testing.T) {
	server := NewMockRouterServer(WithVersion("5.0.1"))
	defer server.Close()

	SetupTestConfig(server.URL)
	defer CleanupTestConfig()

	err := Common.Auth()
	require.NoError(t, err)

	// Create temporary domain file
	tmpDir := t.TempDir()
	domainFile := filepath.Join(tmpDir, "domains.txt")
	err = os.WriteFile(domainFile, []byte("example.com\ntest.com\n"), 0644)
	require.NoError(t, err)

	// Create test groups
	groups := []config.DnsRoutingGroup{
		{
			Name:        "test-group",
			DomainFile:  []string{domainFile},
			InterfaceID: "Wireguard0",
		},
	}

	// Add groups first time
	err = DnsRouting.AddDnsRoutingGroups(groups)
	require.NoError(t, err)

	// Verify groups were created
	existingGroups, err := DnsRouting.GetExistingDnsRoutingGroups()
	require.NoError(t, err)
	assert.Contains(t, existingGroups, "test-group")
	assert.ElementsMatch(t, []string{"example.com", "test.com"}, existingGroups["test-group"])

	// Add same groups again - should be idempotent
	err = DnsRouting.AddDnsRoutingGroups(groups)
	require.NoError(t, err)

	// Verify groups still exist with same domains
	existingGroups, err = DnsRouting.GetExistingDnsRoutingGroups()
	require.NoError(t, err)
	assert.Contains(t, existingGroups, "test-group")
	assert.ElementsMatch(t, []string{"example.com", "test.com"}, existingGroups["test-group"])
}

// TestHandleParse_DnsRoutingPartialUpdate verifies that adding new domains to existing group works
func TestHandleParse_DnsRoutingPartialUpdate(t *testing.T) {
	server := NewMockRouterServer(WithVersion("5.0.1"))
	defer server.Close()

	SetupTestConfig(server.URL)
	defer CleanupTestConfig()
	err := Common.Auth()
	require.NoError(t, err)

	// Create temporary domain files
	tmpDir := t.TempDir()

	initialFile := filepath.Join(tmpDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("example.com\n"), 0644)
	require.NoError(t, err)

	// Create test group with initial domains
	initialGroups := []config.DnsRoutingGroup{
		{
			Name:        "test-group",
			DomainFile:  []string{initialFile},
			InterfaceID: "Wireguard0",
		},
	}

	// Add initial group
	err = DnsRouting.AddDnsRoutingGroups(initialGroups)
	require.NoError(t, err)

	// Verify initial state
	existingGroups, err := DnsRouting.GetExistingDnsRoutingGroups()
	require.NoError(t, err)
	assert.Contains(t, existingGroups, "test-group")
	assert.ElementsMatch(t, []string{"example.com"}, existingGroups["test-group"])

	// Update the domain file with additional domain
	updatedFile := filepath.Join(tmpDir, "updated.txt")
	err = os.WriteFile(updatedFile, []byte("example.com\ntest.com\n"), 0644)
	require.NoError(t, err)

	// Add same group with additional domain
	updatedGroups := []config.DnsRoutingGroup{
		{
			Name:        "test-group",
			DomainFile:  []string{updatedFile},
			InterfaceID: "Wireguard0",
		},
	}

	err = DnsRouting.AddDnsRoutingGroups(updatedGroups)
	require.NoError(t, err)

	// Verify both domains exist
	existingGroups, err = DnsRouting.GetExistingDnsRoutingGroups()
	require.NoError(t, err)
	assert.Contains(t, existingGroups, "test-group")
	assert.ElementsMatch(t, []string{"example.com", "test.com"}, existingGroups["test-group"])
}

// TestGetExistingDnsRoutingGroups verifies the REST API endpoint works correctly
func TestGetExistingDnsRoutingGroups(t *testing.T) {
	groups := []MockDnsRoutingGroup{
		{
			Name:    "group1",
			Domains: []string{"example.com", "test.com"},
		},
		{
			Name:    "group2",
			Domains: []string{"google.com"},
		},
	}
	routes := []MockDnsProxyRoute{
		{
			GroupName:   "group1",
			InterfaceID: "Wireguard0",
			Mode:        "auto",
		},
	}

	server := NewMockRouterServer(WithDnsRoutingGroups(groups, routes))
	defer server.Close()

	SetupTestConfig(server.URL)
	defer CleanupTestConfig()
	err := Common.Auth()
	require.NoError(t, err)

	// Get existing groups via REST API
	existingGroups, err := DnsRouting.GetExistingDnsRoutingGroups()
	require.NoError(t, err)

	// Verify groups were retrieved correctly
	assert.Len(t, existingGroups, 2)
	assert.Contains(t, existingGroups, "group1")
	assert.Contains(t, existingGroups, "group2")
	assert.ElementsMatch(t, []string{"example.com", "test.com"}, existingGroups["group1"])
	assert.ElementsMatch(t, []string{"google.com"}, existingGroups["group2"])
}

// TestHandleParse_DnsRoutingDomainCleanup verifies that unwanted domains are removed from groups
func TestHandleParse_DnsRoutingDomainCleanup(t *testing.T) {
	server := NewMockRouterServer(WithVersion("5.0.1"))
	defer server.Close()

	SetupTestConfig(server.URL)
	defer CleanupTestConfig()
	err := Common.Auth()
	require.NoError(t, err)

	// Create temporary domain files
	tmpDir := t.TempDir()

	initialFile := filepath.Join(tmpDir, "initial.txt")
	err = os.WriteFile(initialFile, []byte("example.com\ntest.com\nold-domain.com\n"), 0644)
	require.NoError(t, err)

	// Create test group with initial domains
	initialGroups := []config.DnsRoutingGroup{
		{
			Name:        "test-group",
			DomainFile:  []string{initialFile},
			InterfaceID: "Wireguard0",
		},
	}

	// Add initial group
	err = DnsRouting.AddDnsRoutingGroups(initialGroups)
	require.NoError(t, err)

	// Verify initial state
	existingGroups, err := DnsRouting.GetExistingDnsRoutingGroups()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"example.com", "test.com", "old-domain.com"}, existingGroups["test-group"])

	// Update domain file - remove old-domain.com and add new-domain.com
	updatedFile := filepath.Join(tmpDir, "updated.txt")
	err = os.WriteFile(updatedFile, []byte("example.com\ntest.com\nnew-domain.com\n"), 0644)
	require.NoError(t, err)

	// Update group
	updatedGroups := []config.DnsRoutingGroup{
		{
			Name:        "test-group",
			DomainFile:  []string{updatedFile},
			InterfaceID: "Wireguard0",
		},
	}

	err = DnsRouting.AddDnsRoutingGroups(updatedGroups)
	require.NoError(t, err)

	// Verify old-domain.com was removed and new-domain.com was added
	existingGroups, err = DnsRouting.GetExistingDnsRoutingGroups()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"example.com", "test.com", "new-domain.com"}, existingGroups["test-group"])
	assert.NotContains(t, existingGroups["test-group"], "old-domain.com")
}

// TestHandleParse_RemoveDomainFromGroup verifies the domain removal command works
func TestHandleParse_RemoveDomainFromGroup(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Create object-group with domains
	createRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "object-group fqdn test-group"},
		{Parse: "object-group fqdn test-group include example.com"},
		{Parse: "object-group fqdn test-group include test.com"},
	}

	body, err := json.Marshal(createRequests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	_ = resp.Body.Close()

	// Remove one domain
	removeRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "no object-group fqdn test-group include example.com"},
	}

	body, err = json.Marshal(removeRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	require.NoError(t, err)

	// Verify success response
	require.Len(t, responses, 1)
	assert.Equal(t, "ok", responses[0].Parse.Status[0].Status)
	assert.Contains(t, responses[0].Parse.Status[0].Message, "example.com")
	assert.Contains(t, responses[0].Parse.Status[0].Message, "removed")
}

// TestGetExistingDnsProxyRoutes verifies the REST API endpoint for dns-proxy routes works
func TestGetExistingDnsProxyRoutes(t *testing.T) {
	groups := []MockDnsRoutingGroup{
		{
			Name:    "group1",
			Domains: []string{"example.com"},
		},
	}
	routes := []MockDnsProxyRoute{
		{
			GroupName:   "group1",
			InterfaceID: "Wireguard0",
			Mode:        "auto",
		},
		{
			GroupName:   "group2",
			InterfaceID: "ISP",
			Mode:        "auto",
		},
	}

	server := NewMockRouterServer(WithDnsRoutingGroups(groups, routes))
	defer server.Close()

	SetupTestConfig(server.URL)
	defer CleanupTestConfig()
	err := Common.Auth()
	require.NoError(t, err)

	// Get existing routes via REST API
	existingRoutes, err := DnsRouting.GetExistingDnsProxyRoutes()
	require.NoError(t, err)

	// Verify routes were retrieved correctly
	assert.Len(t, existingRoutes, 2)
	assert.Equal(t, "Wireguard0", existingRoutes["group1"])
	assert.Equal(t, "ISP", existingRoutes["group2"])
}

// TestHandleParse_DnsRoutingSkipExistingRoute verifies that existing dns-proxy routes are not re-added
func TestHandleParse_DnsRoutingSkipExistingRoute(t *testing.T) {
	// Start with a group and route already configured
	groups := []MockDnsRoutingGroup{
		{
			Name:    "test-group",
			Domains: []string{"example.com"},
		},
	}
	routes := []MockDnsProxyRoute{
		{
			GroupName:   "test-group",
			InterfaceID: "Wireguard0",
			Mode:        "auto",
		},
	}

	server := NewMockRouterServer(WithVersion("5.0.1"), WithDnsRoutingGroups(groups, routes))
	defer server.Close()

	SetupTestConfig(server.URL)
	defer CleanupTestConfig()
	err := Common.Auth()
	require.NoError(t, err)

	// Create temporary domain file
	tmpDir := t.TempDir()
	domainFile := filepath.Join(tmpDir, "domains.txt")
	err = os.WriteFile(domainFile, []byte("example.com\n"), 0644)
	require.NoError(t, err)

	// Try to add the same group with same interface - should skip dns-proxy route
	configGroups := []config.DnsRoutingGroup{
		{
			Name:        "test-group",
			DomainFile:  []string{domainFile},
			InterfaceID: "Wireguard0",
		},
	}

	err = DnsRouting.AddDnsRoutingGroups(configGroups)
	require.NoError(t, err)

	// Verify route still exists (wasn't duplicated)
	existingRoutes, err := DnsRouting.GetExistingDnsProxyRoutes()
	require.NoError(t, err)
	assert.Equal(t, "Wireguard0", existingRoutes["test-group"])
}

// TestHandleParse_DnsRoutingUpdateRouteInterface verifies that dns-proxy route is updated when interface changes
func TestHandleParse_DnsRoutingUpdateRouteInterface(t *testing.T) {
	// Start with a group and route configured with Wireguard0
	groups := []MockDnsRoutingGroup{
		{
			Name:    "test-group",
			Domains: []string{"example.com"},
		},
	}
	routes := []MockDnsProxyRoute{
		{
			GroupName:   "test-group",
			InterfaceID: "Wireguard0",
			Mode:        "auto",
		},
	}

	server := NewMockRouterServer(WithVersion("5.0.1"), WithDnsRoutingGroups(groups, routes))
	defer server.Close()

	SetupTestConfig(server.URL)
	defer CleanupTestConfig()
	err := Common.Auth()
	require.NoError(t, err)

	// Create temporary domain file
	tmpDir := t.TempDir()
	domainFile := filepath.Join(tmpDir, "domains.txt")
	err = os.WriteFile(domainFile, []byte("example.com\n"), 0644)
	require.NoError(t, err)

	// Update the group to use ISP interface instead
	configGroups := []config.DnsRoutingGroup{
		{
			Name:        "test-group",
			DomainFile:  []string{domainFile},
			InterfaceID: "ISP",
		},
	}

	err = DnsRouting.AddDnsRoutingGroups(configGroups)
	require.NoError(t, err)

	// Verify route was updated to ISP
	existingRoutes, err := DnsRouting.GetExistingDnsProxyRoutes()
	require.NoError(t, err)
	assert.Equal(t, "ISP", existingRoutes["test-group"])
}
