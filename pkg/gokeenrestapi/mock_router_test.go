package gokeenrestapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMockRouterServer_BasicEndpoints verifies that all endpoints are registered and accessible
func TestNewMockRouterServer_BasicEndpoints(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"Auth GET", "GET", "/auth", http.StatusUnauthorized},
		{"Version", "GET", "/rci/show/version", http.StatusOK},
		{"Interfaces", "GET", "/rci/show/interface", http.StatusOK},
		{"Single Interface", "GET", "/rci/show/interface/Wireguard0", http.StatusOK},
		{"SC Interfaces", "GET", "/rci/show/sc/interface", http.StatusOK},
		{"Routes", "GET", "/rci/ip/route", http.StatusOK},
		{"DNS Records", "GET", "/rci/show/ip/name-server", http.StatusOK},
		{"Hotspot", "GET", "/rci/show/ip/hotspot", http.StatusOK},
		{"Running Config", "GET", "/rci/show/running-config", http.StatusOK},
		{"System Mode", "GET", "/rci/show/system/mode", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, server.URL+tt.path, nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() {
				_ = resp.Body.Close()
			}()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestGetState_ReturnsCurrentState verifies GetState returns a snapshot of the state
func TestGetState_ReturnsCurrentState(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Get the mock router instance by creating a new one with same config
	mock := NewMockRouter()
	state := mock.GetState()

	// Verify default state is present
	assert.NotEmpty(t, state.Interfaces)
	assert.NotEmpty(t, state.Routes)
	assert.NotEmpty(t, state.DNSRecords)
	assert.NotEmpty(t, state.HotspotDevices)
	assert.Equal(t, "router", state.SystemMode.Active)
}

// TestResetState_RestoresInitialState verifies ResetState works correctly
func TestResetState_RestoresInitialState(t *testing.T) {
	mock := NewMockRouter()

	// Get initial state
	initialState := mock.GetState()
	initialRouteCount := len(initialState.Routes)

	// Modify state
	mock.mu.Lock()
	mock.routes = append(mock.routes, MockRoute{
		Network:   "10.0.0.0",
		Mask:      "255.255.255.0",
		Interface: "Wireguard0",
	})
	mock.mu.Unlock()

	// Verify state was modified
	modifiedState := mock.GetState()
	assert.Equal(t, initialRouteCount+1, len(modifiedState.Routes))

	// Reset state
	mock.ResetState()

	// Verify state was restored
	restoredState := mock.GetState()
	assert.Equal(t, initialRouteCount, len(restoredState.Routes))
}

// TestNewMockRouterServer_WithCustomOptions verifies functional options work
func TestNewMockRouterServer_WithCustomOptions(t *testing.T) {
	customInterfaces := []MockInterface{
		{
			ID:        "CustomWG",
			Type:      InterfaceTypeWireguard,
			Address:   "192.168.100.1/24",
			Connected: StateConnected,
		},
	}

	server := NewMockRouterServer(WithInterfaces(customInterfaces))
	defer server.Close()

	// Query interfaces
	resp, err := http.Get(server.URL + "/rci/show/interface")
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	var interfaces map[string]gokeenrestapimodels.RciShowInterface
	err = json.NewDecoder(resp.Body).Decode(&interfaces)
	require.NoError(t, err)

	// Verify custom interface is present
	customIface, exists := interfaces["CustomWG"]
	assert.True(t, exists)
	assert.Equal(t, "192.168.100.1/24", customIface.Address)
}

// TestHandleInterface_NotFound verifies 404 for non-existent interface
func TestHandleInterface_NotFound(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/rci/show/interface/NonExistent")
	require.NoError(t, err)
	defer func() {
		_ = resp.Body.Close()
	}()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestHandleParse_RequestResponseLengthMatching verifies that parse endpoint returns matching length responses
func TestHandleParse_RequestResponseLengthMatching(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	tests := []struct {
		name         string
		requestCount int
	}{
		{"Single request", 1},
		{"Multiple requests", 3},
		{"Empty array", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create requests
			requests := make([]gokeenrestapimodels.ParseRequest, tt.requestCount)
			for i := 0; i < tt.requestCount; i++ {
				requests[i] = gokeenrestapimodels.ParseRequest{
					Parse: "unknown command",
				}
			}

			body, err := json.Marshal(requests)
			require.NoError(t, err)

			// Send request
			resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			// Decode response
			var responses []gokeenrestapimodels.ParseResponse
			err = json.NewDecoder(resp.Body).Decode(&responses)
			require.NoError(t, err)

			// Verify length matches
			assert.Equal(t, tt.requestCount, len(responses))
		})
	}
}

// TestHandleParse_InvalidCommands verifies that invalid commands return error responses
func TestHandleParse_InvalidCommands(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	tests := []struct {
		name          string
		command       string
		expectSuccess bool // Empty command returns success to match old mock behavior
	}{
		{"Empty command", "", true}, // Old mock accepted empty commands
		{"Unknown command", "unknown command", false},
		{"Unknown ip subcommand", "ip unknown", false},
		{"Incomplete no ip command", "no ip", false},
		{"Unknown no subcommand", "no unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := []gokeenrestapimodels.ParseRequest{
				{Parse: tt.command},
			}

			body, err := json.Marshal(requests)
			require.NoError(t, err)

			resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			var responses []gokeenrestapimodels.ParseResponse
			err = json.NewDecoder(resp.Body).Decode(&responses)
			require.NoError(t, err)

			require.Len(t, responses, 1)
			require.Len(t, responses[0].Parse.Status, 1)

			if tt.expectSuccess {
				assert.Equal(t, "ok", responses[0].Parse.Status[0].Status)
			} else {
				assert.Equal(t, "error", responses[0].Parse.Status[0].Status)
			}
			assert.NotEmpty(t, responses[0].Parse.Status[0].Message)
		})
	}
}

// TestHandleParse_MethodNotAllowed verifies that non-POST requests return 405
func TestHandleParse_MethodNotAllowed(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	methods := []string{"GET", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, err := http.NewRequest(method, server.URL+"/rci/", nil)
			require.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
		})
	}
}

// TestHandleParse_MalformedJSON verifies that malformed JSON returns 400
func TestHandleParse_MalformedJSON(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader([]byte("not json")))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestHandleParse_AddRoute verifies that adding routes works correctly
func TestHandleParse_AddRoute(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Get initial route count
	resp, err := http.Get(server.URL + "/rci/ip/route")
	require.NoError(t, err)
	var initialRoutes []gokeenrestapimodels.RciIpRoute
	err = json.NewDecoder(resp.Body).Decode(&initialRoutes)
	_ = resp.Body.Close()
	require.NoError(t, err)
	initialCount := len(initialRoutes)

	// Add a route
	requests := []gokeenrestapimodels.ParseRequest{
		{Parse: "ip route 10.0.0.0 255.255.255.0 Wireguard0"},
	}

	body, err := json.Marshal(requests)
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

	// Verify route was added
	resp, err = http.Get(server.URL + "/rci/ip/route")
	require.NoError(t, err)
	var routes []gokeenrestapimodels.RciIpRoute
	err = json.NewDecoder(resp.Body).Decode(&routes)
	_ = resp.Body.Close()
	require.NoError(t, err)

	assert.Equal(t, initialCount+1, len(routes))

	// Find the added route
	found := false
	for _, route := range routes {
		if route.Network == "10.0.0.0" && route.Mask == "255.255.255.0" && route.Interface == "Wireguard0" {
			found = true
			break
		}
	}
	assert.True(t, found, "Added route should be present in route list")
}

// TestHandleParse_AddRouteWithAuto verifies that adding routes with auto flag works
func TestHandleParse_AddRouteWithAuto(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Add a route with auto flag
	requests := []gokeenrestapimodels.ParseRequest{
		{Parse: "ip route 172.16.0.0 255.255.0.0 Wireguard0 auto"},
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

	// Verify route was added
	resp, err = http.Get(server.URL + "/rci/ip/route")
	require.NoError(t, err)
	var routes []gokeenrestapimodels.RciIpRoute
	err = json.NewDecoder(resp.Body).Decode(&routes)
	_ = resp.Body.Close()
	require.NoError(t, err)

	// Find the added route and verify auto flag
	found := false
	for _, route := range routes {
		if route.Network == "172.16.0.0" && route.Mask == "255.255.0.0" && route.Interface == "Wireguard0" {
			found = true
			assert.True(t, route.Auto, "Route should have auto flag set")
			break
		}
	}
	assert.True(t, found, "Added route should be present in route list")
}

// TestHandleParse_AddRouteInvalidInterface verifies that adding routes to non-existent interface fails
func TestHandleParse_AddRouteInvalidInterface(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Try to add a route to non-existent interface
	requests := []gokeenrestapimodels.ParseRequest{
		{Parse: "ip route 10.0.0.0 255.255.255.0 NonExistentInterface"},
	}

	body, err := json.Marshal(requests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	require.NoError(t, err)

	// Verify error response
	require.Len(t, responses, 1)
	require.Len(t, responses[0].Parse.Status, 1)
	assert.Equal(t, "error", responses[0].Parse.Status[0].Status)
	assert.Contains(t, responses[0].Parse.Status[0].Message, "does not exist")
}

// TestHandleParse_AddRouteInvalidFormat verifies that malformed route commands fail
func TestHandleParse_AddRouteInvalidFormat(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	tests := []struct {
		name    string
		command string
	}{
		{"Missing interface", "ip route 10.0.0.0 255.255.255.0"},
		{"Missing mask", "ip route 10.0.0.0"},
		{"Missing network", "ip route"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := []gokeenrestapimodels.ParseRequest{
				{Parse: tt.command},
			}

			body, err := json.Marshal(requests)
			require.NoError(t, err)

			resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			var responses []gokeenrestapimodels.ParseResponse
			err = json.NewDecoder(resp.Body).Decode(&responses)
			require.NoError(t, err)

			// Verify error response
			require.Len(t, responses, 1)
			require.Len(t, responses[0].Parse.Status, 1)
			assert.Equal(t, "error", responses[0].Parse.Status[0].Status)
		})
	}
}

// TestHandleParse_DeleteRoute verifies that deleting routes works correctly
func TestHandleParse_DeleteRoute(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// First, add a route
	addRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "ip route 10.0.0.0 255.255.255.0 Wireguard0"},
	}

	body, err := json.Marshal(addRequests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	_ = resp.Body.Close()

	// Get route count after adding
	resp, err = http.Get(server.URL + "/rci/ip/route")
	require.NoError(t, err)
	var routesAfterAdd []gokeenrestapimodels.RciIpRoute
	err = json.NewDecoder(resp.Body).Decode(&routesAfterAdd)
	_ = resp.Body.Close()
	require.NoError(t, err)
	countAfterAdd := len(routesAfterAdd)

	// Now delete the route
	deleteRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "no ip route 10.0.0.0 255.255.255.0 Wireguard0"},
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

	// Verify route was deleted
	resp, err = http.Get(server.URL + "/rci/ip/route")
	require.NoError(t, err)
	var routesAfterDelete []gokeenrestapimodels.RciIpRoute
	err = json.NewDecoder(resp.Body).Decode(&routesAfterDelete)
	_ = resp.Body.Close()
	require.NoError(t, err)

	assert.Equal(t, countAfterAdd-1, len(routesAfterDelete))

	// Verify the specific route is gone
	for _, route := range routesAfterDelete {
		if route.Network == "10.0.0.0" && route.Mask == "255.255.255.0" && route.Interface == "Wireguard0" {
			t.Fatal("Deleted route should not be present in route list")
		}
	}
}

// TestHandleParse_DeleteNonExistentRoute verifies that deleting non-existent routes succeeds (matches old mock behavior)
func TestHandleParse_DeleteNonExistentRoute(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Try to delete a route that doesn't exist
	requests := []gokeenrestapimodels.ParseRequest{
		{Parse: "no ip route 99.99.99.0 255.255.255.0 Wireguard0"},
	}

	body, err := json.Marshal(requests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	require.NoError(t, err)

	// Verify success response (matches old mock behavior - doesn't error for non-existent items)
	require.Len(t, responses, 1)
	require.Len(t, responses[0].Parse.Status, 1)
	assert.Equal(t, "ok", responses[0].Parse.Status[0].Status)
}

// TestHandleParse_DeleteRouteInvalidFormat verifies that malformed delete route commands fail
func TestHandleParse_DeleteRouteInvalidFormat(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	tests := []struct {
		name    string
		command string
	}{
		{"Missing interface", "no ip route 10.0.0.0 255.255.255.0"},
		{"Missing mask", "no ip route 10.0.0.0"},
		{"Missing network", "no ip route"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := []gokeenrestapimodels.ParseRequest{
				{Parse: tt.command},
			}

			body, err := json.Marshal(requests)
			require.NoError(t, err)

			resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			var responses []gokeenrestapimodels.ParseResponse
			err = json.NewDecoder(resp.Body).Decode(&responses)
			require.NoError(t, err)

			// Verify error response
			require.Len(t, responses, 1)
			require.Len(t, responses[0].Parse.Status, 1)
			assert.Equal(t, "error", responses[0].Parse.Status[0].Status)
		})
	}
}

// TestHandleParse_RouteRoundTrip verifies that adding and then deleting a route works correctly
func TestHandleParse_RouteRoundTrip(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Get initial route count
	resp, err := http.Get(server.URL + "/rci/ip/route")
	require.NoError(t, err)
	var initialRoutes []gokeenrestapimodels.RciIpRoute
	err = json.NewDecoder(resp.Body).Decode(&initialRoutes)
	_ = resp.Body.Close()
	require.NoError(t, err)
	initialCount := len(initialRoutes)

	// Add a route
	addRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "ip route 192.168.100.0 255.255.255.0 ISP auto"},
	}

	body, err := json.Marshal(addRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	_ = resp.Body.Close()

	// Verify route was added
	resp, err = http.Get(server.URL + "/rci/ip/route")
	require.NoError(t, err)
	var routesAfterAdd []gokeenrestapimodels.RciIpRoute
	err = json.NewDecoder(resp.Body).Decode(&routesAfterAdd)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, initialCount+1, len(routesAfterAdd))

	// Delete the route
	deleteRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "no ip route 192.168.100.0 255.255.255.0 ISP"},
	}

	body, err = json.Marshal(deleteRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	_ = resp.Body.Close()

	// Verify route was deleted and we're back to initial count
	resp, err = http.Get(server.URL + "/rci/ip/route")
	require.NoError(t, err)
	var routesAfterDelete []gokeenrestapimodels.RciIpRoute
	err = json.NewDecoder(resp.Body).Decode(&routesAfterDelete)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, initialCount, len(routesAfterDelete))
}

// TestHandleParse_InterfaceState verifies that interface state changes work correctly
func TestHandleParse_InterfaceState(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Get initial interface state
	resp, err := http.Get(server.URL + "/rci/show/interface/Wireguard0")
	require.NoError(t, err)
	var initialInterface gokeenrestapimodels.RciShowInterface
	err = json.NewDecoder(resp.Body).Decode(&initialInterface)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, StateConnected, initialInterface.Connected)
	assert.Equal(t, StateUp, initialInterface.Link)
	assert.Equal(t, StateUp, initialInterface.State)

	// Bring interface down
	downRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "interface Wireguard0 down"},
	}

	body, err := json.Marshal(downRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var downResponses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&downResponses)
	_ = resp.Body.Close()
	require.NoError(t, err)
	require.Len(t, downResponses, 1)
	assert.Equal(t, StatusOK, downResponses[0].Parse.Status[0].Status)

	// Verify interface is down
	resp, err = http.Get(server.URL + "/rci/show/interface/Wireguard0")
	require.NoError(t, err)
	var downInterface gokeenrestapimodels.RciShowInterface
	err = json.NewDecoder(resp.Body).Decode(&downInterface)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, StateDisconnected, downInterface.Connected)
	assert.Equal(t, StateDown, downInterface.Link)
	assert.Equal(t, StateDown, downInterface.State)

	// Bring interface up
	upRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "interface Wireguard0 up"},
	}

	body, err = json.Marshal(upRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var upResponses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&upResponses)
	_ = resp.Body.Close()
	require.NoError(t, err)
	require.Len(t, upResponses, 1)
	assert.Equal(t, StatusOK, upResponses[0].Parse.Status[0].Status)

	// Verify interface is up
	resp, err = http.Get(server.URL + "/rci/show/interface/Wireguard0")
	require.NoError(t, err)
	var upInterface gokeenrestapimodels.RciShowInterface
	err = json.NewDecoder(resp.Body).Decode(&upInterface)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, StateConnected, upInterface.Connected)
	assert.Equal(t, StateUp, upInterface.Link)
	assert.Equal(t, StateUp, upInterface.State)
}

// TestHandleParse_InterfaceStateInvalidInterface verifies error handling for non-existent interfaces
func TestHandleParse_InterfaceStateInvalidInterface(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	requests := []gokeenrestapimodels.ParseRequest{
		{Parse: "interface NonExistent up"},
	}

	body, err := json.Marshal(requests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	_ = resp.Body.Close()
	require.NoError(t, err)

	require.Len(t, responses, 1)
	require.Len(t, responses[0].Parse.Status, 1)
	assert.Equal(t, StatusError, responses[0].Parse.Status[0].Status)
	assert.Contains(t, responses[0].Parse.Status[0].Message, "does not exist")
}

// TestHandleParse_AwgConfig verifies that AWG parameter updates work correctly
func TestHandleParse_AwgConfig(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Get initial AWG parameters
	resp, err := http.Get(server.URL + "/rci/show/sc/interface/Wireguard0")
	require.NoError(t, err)
	var initialInterface gokeenrestapimodels.RciShowScInterface
	err = json.NewDecoder(resp.Body).Decode(&initialInterface)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "3", initialInterface.Wireguard.Asc.Jc)
	assert.Equal(t, "50", initialInterface.Wireguard.Asc.Jmin)

	// Update AWG parameters
	updateRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "interface Wireguard0 wireguard asc jc 5 jmin 100 jmax 2000 s1 99 s2 10 h1 10 h2 20 h3 30 h4 40"},
	}

	body, err := json.Marshal(updateRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var updateResponses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&updateResponses)
	_ = resp.Body.Close()
	require.NoError(t, err)
	require.Len(t, updateResponses, 1)
	assert.Equal(t, StatusOK, updateResponses[0].Parse.Status[0].Status)

	// Verify AWG parameters were updated
	resp, err = http.Get(server.URL + "/rci/show/sc/interface/Wireguard0")
	require.NoError(t, err)
	var updatedInterface gokeenrestapimodels.RciShowScInterface
	err = json.NewDecoder(resp.Body).Decode(&updatedInterface)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "5", updatedInterface.Wireguard.Asc.Jc)
	assert.Equal(t, "100", updatedInterface.Wireguard.Asc.Jmin)
	assert.Equal(t, "2000", updatedInterface.Wireguard.Asc.Jmax)
	assert.Equal(t, "99", updatedInterface.Wireguard.Asc.S1)
	assert.Equal(t, "10", updatedInterface.Wireguard.Asc.S2)
	assert.Equal(t, "10", updatedInterface.Wireguard.Asc.H1)
	assert.Equal(t, "20", updatedInterface.Wireguard.Asc.H2)
	assert.Equal(t, "30", updatedInterface.Wireguard.Asc.H3)
	assert.Equal(t, "40", updatedInterface.Wireguard.Asc.H4)
}

// TestHandleParse_AwgConfigInvalidInterface verifies error handling for non-existent SC interfaces
func TestHandleParse_AwgConfigInvalidInterface(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	requests := []gokeenrestapimodels.ParseRequest{
		{Parse: "interface NonExistent wireguard asc jc 5"},
	}

	body, err := json.Marshal(requests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	_ = resp.Body.Close()
	require.NoError(t, err)

	require.Len(t, responses, 1)
	require.Len(t, responses[0].Parse.Status, 1)
	assert.Equal(t, StatusError, responses[0].Parse.Status[0].Status)
	assert.Contains(t, responses[0].Parse.Status[0].Message, "does not exist")
}

// TestHandleParse_CreateInterface verifies that interface creation works correctly
func TestHandleParse_CreateInterface(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Verify interface doesn't exist initially
	resp, err := http.Get(server.URL + "/rci/show/interface/Wireguard1")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	_ = resp.Body.Close()

	// Create new interface
	createRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "interface Wireguard1 create type Wireguard description TestInterface address 10.1.1.1/24"},
	}

	body, err := json.Marshal(createRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var createResponses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&createResponses)
	_ = resp.Body.Close()
	require.NoError(t, err)
	require.Len(t, createResponses, 1)
	assert.Equal(t, StatusOK, createResponses[0].Parse.Status[0].Status)

	// Verify interface exists in regular interface listing
	resp, err = http.Get(server.URL + "/rci/show/interface/Wireguard1")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var newInterface gokeenrestapimodels.RciShowInterface
	err = json.NewDecoder(resp.Body).Decode(&newInterface)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "Wireguard1", newInterface.Id)
	assert.Equal(t, InterfaceTypeWireguard, newInterface.Type)
	assert.Equal(t, "TestInterface", newInterface.Description)
	assert.Equal(t, "10.1.1.1/24", newInterface.Address)
	assert.Equal(t, StateDisconnected, newInterface.Connected)
	assert.Equal(t, StateDown, newInterface.Link)
	assert.Equal(t, StateDown, newInterface.State)

	// Verify interface exists in SC interface listing
	resp, err = http.Get(server.URL + "/rci/show/sc/interface/Wireguard1")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var newScInterface gokeenrestapimodels.RciShowScInterface
	err = json.NewDecoder(resp.Body).Decode(&newScInterface)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, "TestInterface", newScInterface.Description)
	assert.Equal(t, "10.1.1.1/24", newScInterface.IP.Address.Address)
}

// TestHandleParse_CreateInterfaceAlreadyExists verifies error handling for duplicate interface creation
func TestHandleParse_CreateInterfaceAlreadyExists(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	requests := []gokeenrestapimodels.ParseRequest{
		{Parse: "interface Wireguard0 create type Wireguard"},
	}

	body, err := json.Marshal(requests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var responses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&responses)
	_ = resp.Body.Close()
	require.NoError(t, err)

	require.Len(t, responses, 1)
	require.Len(t, responses[0].Parse.Status, 1)
	assert.Equal(t, StatusError, responses[0].Parse.Status[0].Status)
	assert.Contains(t, responses[0].Parse.Status[0].Message, "already exists")
}

// TestHandleParse_DeleteKnownHost verifies that deleting known hosts works correctly
func TestHandleParse_DeleteKnownHost(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Get initial hotspot devices
	resp, err := http.Get(server.URL + "/rci/show/ip/hotspot")
	require.NoError(t, err)
	var initialHotspot gokeenrestapimodels.RciShowIpHotspot
	err = json.NewDecoder(resp.Body).Decode(&initialHotspot)
	_ = resp.Body.Close()
	require.NoError(t, err)
	initialCount := len(initialHotspot.Host)
	assert.Greater(t, initialCount, 0, "Should have initial hotspot devices")

	// Find a device to delete
	deviceToDelete := initialHotspot.Host[0]

	// Delete the known host
	deleteRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: fmt.Sprintf("no known host \"%s\"", deviceToDelete.Mac)},
	}

	body, err := json.Marshal(deleteRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var deleteResponses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&deleteResponses)
	_ = resp.Body.Close()
	require.NoError(t, err)
	require.Len(t, deleteResponses, 1)
	assert.Equal(t, StatusOK, deleteResponses[0].Parse.Status[0].Status)

	// Verify device was deleted
	resp, err = http.Get(server.URL + "/rci/show/ip/hotspot")
	require.NoError(t, err)
	var updatedHotspot gokeenrestapimodels.RciShowIpHotspot
	err = json.NewDecoder(resp.Body).Decode(&updatedHotspot)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, initialCount-1, len(updatedHotspot.Host))

	// Verify the specific device is gone
	for _, host := range updatedHotspot.Host {
		if strings.EqualFold(host.Mac, deviceToDelete.Mac) {
			t.Fatalf("Deleted host with MAC %s should not be present in hotspot list", deviceToDelete.Mac)
		}
	}
}

// TestHandleParse_DeleteKnownHostWithoutQuotes verifies that MAC addresses without quotes work
func TestHandleParse_DeleteKnownHostWithoutQuotes(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Get initial hotspot devices
	resp, err := http.Get(server.URL + "/rci/show/ip/hotspot")
	require.NoError(t, err)
	var initialHotspot gokeenrestapimodels.RciShowIpHotspot
	err = json.NewDecoder(resp.Body).Decode(&initialHotspot)
	_ = resp.Body.Close()
	require.NoError(t, err)
	initialCount := len(initialHotspot.Host)

	// Find a device to delete
	deviceToDelete := initialHotspot.Host[0]

	// Delete the known host without quotes
	deleteRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: fmt.Sprintf("no known host %s", deviceToDelete.Mac)},
	}

	body, err := json.Marshal(deleteRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var deleteResponses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&deleteResponses)
	_ = resp.Body.Close()
	require.NoError(t, err)
	require.Len(t, deleteResponses, 1)
	assert.Equal(t, StatusOK, deleteResponses[0].Parse.Status[0].Status)

	// Verify device was deleted
	resp, err = http.Get(server.URL + "/rci/show/ip/hotspot")
	require.NoError(t, err)
	var updatedHotspot gokeenrestapimodels.RciShowIpHotspot
	err = json.NewDecoder(resp.Body).Decode(&updatedHotspot)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, initialCount-1, len(updatedHotspot.Host))
}

// TestHandleParse_DeleteKnownHostCaseInsensitive verifies that MAC address matching is case-insensitive
func TestHandleParse_DeleteKnownHostCaseInsensitive(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Get initial hotspot devices
	resp, err := http.Get(server.URL + "/rci/show/ip/hotspot")
	require.NoError(t, err)
	var initialHotspot gokeenrestapimodels.RciShowIpHotspot
	err = json.NewDecoder(resp.Body).Decode(&initialHotspot)
	_ = resp.Body.Close()
	require.NoError(t, err)
	initialCount := len(initialHotspot.Host)

	// Find a device to delete and convert MAC to uppercase
	deviceToDelete := initialHotspot.Host[0]
	upperMac := strings.ToUpper(deviceToDelete.Mac)

	// Delete the known host using uppercase MAC
	deleteRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: fmt.Sprintf("no known host \"%s\"", upperMac)},
	}

	body, err := json.Marshal(deleteRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var deleteResponses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&deleteResponses)
	_ = resp.Body.Close()
	require.NoError(t, err)
	require.Len(t, deleteResponses, 1)
	assert.Equal(t, StatusOK, deleteResponses[0].Parse.Status[0].Status)

	// Verify device was deleted
	resp, err = http.Get(server.URL + "/rci/show/ip/hotspot")
	require.NoError(t, err)
	var updatedHotspot gokeenrestapimodels.RciShowIpHotspot
	err = json.NewDecoder(resp.Body).Decode(&updatedHotspot)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, initialCount-1, len(updatedHotspot.Host))
}

// TestHandleParse_DeleteKnownHostNotFound verifies that deleting non-existent hosts succeeds (matches old mock behavior)
func TestHandleParse_DeleteKnownHostNotFound(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Try to delete a host that doesn't exist
	deleteRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "no known host \"ff:ff:ff:ff:ff:ff\""},
	}

	body, err := json.Marshal(deleteRequests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var deleteResponses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&deleteResponses)
	_ = resp.Body.Close()
	require.NoError(t, err)

	require.Len(t, deleteResponses, 1)
	require.Len(t, deleteResponses[0].Parse.Status, 1)
	assert.Equal(t, StatusOK, deleteResponses[0].Parse.Status[0].Status)
}

// TestHandleParse_DeleteKnownHostInvalidFormat verifies error handling for malformed commands
func TestHandleParse_DeleteKnownHostInvalidFormat(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Try to delete without MAC address
	deleteRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: "no known host"},
	}

	body, err := json.Marshal(deleteRequests)
	require.NoError(t, err)

	resp, err := http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var deleteResponses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&deleteResponses)
	_ = resp.Body.Close()
	require.NoError(t, err)

	require.Len(t, deleteResponses, 1)
	require.Len(t, deleteResponses[0].Parse.Status, 1)
	assert.Equal(t, StatusError, deleteResponses[0].Parse.Status[0].Status)
}

// TestHandleParse_DeleteMultipleKnownHosts verifies that multiple hosts can be deleted in sequence
func TestHandleParse_DeleteMultipleKnownHosts(t *testing.T) {
	server := NewMockRouterServer()
	defer server.Close()

	// Get initial hotspot devices
	resp, err := http.Get(server.URL + "/rci/show/ip/hotspot")
	require.NoError(t, err)
	var initialHotspot gokeenrestapimodels.RciShowIpHotspot
	err = json.NewDecoder(resp.Body).Decode(&initialHotspot)
	_ = resp.Body.Close()
	require.NoError(t, err)
	initialCount := len(initialHotspot.Host)
	require.GreaterOrEqual(t, initialCount, 2, "Need at least 2 hosts for this test")

	// Delete multiple hosts in one request
	deleteRequests := []gokeenrestapimodels.ParseRequest{
		{Parse: fmt.Sprintf("no known host \"%s\"", initialHotspot.Host[0].Mac)},
		{Parse: fmt.Sprintf("no known host \"%s\"", initialHotspot.Host[1].Mac)},
	}

	body, err := json.Marshal(deleteRequests)
	require.NoError(t, err)

	resp, err = http.Post(server.URL+"/rci/", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	var deleteResponses []gokeenrestapimodels.ParseResponse
	err = json.NewDecoder(resp.Body).Decode(&deleteResponses)
	_ = resp.Body.Close()
	require.NoError(t, err)
	require.Len(t, deleteResponses, 2)
	assert.Equal(t, StatusOK, deleteResponses[0].Parse.Status[0].Status)
	assert.Equal(t, StatusOK, deleteResponses[1].Parse.Status[0].Status)

	// Verify both devices were deleted
	resp, err = http.Get(server.URL + "/rci/show/ip/hotspot")
	require.NoError(t, err)
	var updatedHotspot gokeenrestapimodels.RciShowIpHotspot
	err = json.NewDecoder(resp.Body).Decode(&updatedHotspot)
	_ = resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, initialCount-2, len(updatedHotspot.Host))
}
