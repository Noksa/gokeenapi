package gokeenrestapi

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/noksa/gokeenapi/pkg/gokeenrestapimodels"
)

// MockInterface represents a network interface in the mock router
type MockInterface struct {
	ID          string
	Type        string
	Description string
	Address     string
	Connected   string
	Link        string
	State       string
	DefaultGw   bool
}

// MockRoute represents a static route in the mock router
type MockRoute struct {
	Network   string
	Host      string
	Mask      string
	Interface string
	Auto      bool
}

// MockHost represents a device connected to the router (hotspot)
type MockHost struct {
	Name       string
	Mac        string
	IP         string
	Hostname   string
	Registered bool
	Link       string
	Via        string
}

// MockSystemMode represents the system mode configuration
type MockSystemMode struct {
	Active   string
	Selected string
}

// MockIP represents IP configuration for SC interfaces
type MockIP struct {
	Address string
}

// MockAsc represents AWG (Amnezia WireGuard) configuration parameters
type MockAsc struct {
	Jc   string
	Jmin string
	Jmax string
	S1   string
	S2   string
	H1   string
	H2   string
	H3   string
	H4   string
}

// MockPeer represents a WireGuard peer configuration
type MockPeer struct {
	Key               string
	Comment           string
	Endpoint          string
	KeepaliveInterval int
	PresharedKey      string
	AllowedIPs        []MockAllowedIP
}

// MockAllowedIP represents an allowed IP range for a WireGuard peer
type MockAllowedIP struct {
	Address string
	Mask    string
}

// MockWireguard represents WireGuard configuration for SC interfaces
type MockWireguard struct {
	Asc  MockAsc
	Peer []MockPeer
}

// MockScInterface represents the SC (system configuration) view of an interface
type MockScInterface struct {
	Description string
	IP          MockIP
	Wireguard   MockWireguard
}

// MockDnsRoutingGroup represents a DNS-routing object-group in the mock router
type MockDnsRoutingGroup struct {
	Name    string
	Domains []string
}

// MockDnsProxyRoute represents a dns-proxy route in the mock router
type MockDnsProxyRoute struct {
	GroupName   string
	InterfaceID string
	Mode        string // typically "auto"
}

// MockRouterState represents a snapshot of the mock router's state
type MockRouterState struct {
	Interfaces       map[string]*MockInterface
	ScInterfaces     map[string]*MockScInterface
	Routes           []MockRoute
	DNSRecords       map[string]string
	HotspotDevices   []MockHost
	SystemMode       MockSystemMode
	DnsRoutingGroups []MockDnsRoutingGroup
	DnsProxyRoutes   []MockDnsProxyRoute
}

// MockRouter is a comprehensive mock implementation of the Keenetic router API
// It maintains stateful behavior and supports all API endpoints for testing
type MockRouter struct {
	mu                  sync.RWMutex
	interfaces          map[string]*MockInterface
	scInterfaces        map[string]*MockScInterface
	routes              []MockRoute
	dnsRecords          map[string]string
	hotspotDevices      []MockHost
	authRealm           string
	authChallenge       string
	sessionCookie       string
	systemMode          MockSystemMode
	dnsRoutingGroups    []MockDnsRoutingGroup
	dnsProxyRoutes      []MockDnsProxyRoute
	initialState        MockRouterState // Store initial state for reset functionality
	staticRunningConfig []string        // Optional static running config (overrides generated config)
	version             string          // Router firmware version (default: "4.3.6.3")
}

// MockRouterOption is a functional option for configuring the mock router
type MockRouterOption func(*MockRouter)

// WithInterfaces sets custom initial interfaces for the mock router
func WithInterfaces(interfaces []MockInterface) MockRouterOption {
	return func(m *MockRouter) {
		m.interfaces = make(map[string]*MockInterface)
		for i := range interfaces {
			iface := interfaces[i]
			m.interfaces[iface.ID] = &iface
		}
	}
}

// WithScInterfaces sets custom initial SC interfaces for the mock router
func WithScInterfaces(scInterfaces map[string]MockScInterface) MockRouterOption {
	return func(m *MockRouter) {
		m.scInterfaces = make(map[string]*MockScInterface)
		for id, scIface := range scInterfaces {
			scIfaceCopy := scIface
			m.scInterfaces[id] = &scIfaceCopy
		}
	}
}

// WithRoutes sets custom initial routes for the mock router
func WithRoutes(routes []MockRoute) MockRouterOption {
	return func(m *MockRouter) {
		m.routes = make([]MockRoute, len(routes))
		copy(m.routes, routes)
	}
}

// WithDNSRecords sets custom initial DNS records for the mock router
func WithDNSRecords(records map[string]string) MockRouterOption {
	return func(m *MockRouter) {
		m.dnsRecords = make(map[string]string)
		maps.Copy(m.dnsRecords, records)
	}
}

// WithHotspotDevices sets custom initial hotspot devices for the mock router
func WithHotspotDevices(devices []MockHost) MockRouterOption {
	return func(m *MockRouter) {
		m.hotspotDevices = make([]MockHost, len(devices))
		copy(m.hotspotDevices, devices)
	}
}

// WithSystemMode sets custom system mode for the mock router
func WithSystemMode(mode MockSystemMode) MockRouterOption {
	return func(m *MockRouter) {
		m.systemMode = mode
	}
}

// WithStaticRunningConfig sets static running config lines (overrides generated config)
func WithStaticRunningConfig(lines []string) MockRouterOption {
	return func(m *MockRouter) {
		m.staticRunningConfig = make([]string, len(lines))
		copy(m.staticRunningConfig, lines)
	}
}

// WithDnsRoutingGroups sets custom initial DNS-routing groups for the mock router
func WithDnsRoutingGroups(groups []MockDnsRoutingGroup, routes []MockDnsProxyRoute) MockRouterOption {
	return func(m *MockRouter) {
		m.dnsRoutingGroups = make([]MockDnsRoutingGroup, len(groups))
		copy(m.dnsRoutingGroups, groups)
		m.dnsProxyRoutes = make([]MockDnsProxyRoute, len(routes))
		copy(m.dnsProxyRoutes, routes)
	}
}

// WithVersion sets custom firmware version for the mock router
func WithVersion(version string) MockRouterOption {
	return func(m *MockRouter) {
		m.version = version
	}
}

// NewMockRouter creates a new mock router with default state and optional configuration
func NewMockRouter(opts ...MockRouterOption) *MockRouter {
	m := &MockRouter{
		interfaces:       make(map[string]*MockInterface),
		scInterfaces:     make(map[string]*MockScInterface),
		routes:           []MockRoute{},
		dnsRecords:       make(map[string]string),
		hotspotDevices:   []MockHost{},
		dnsRoutingGroups: []MockDnsRoutingGroup{},
		dnsProxyRoutes:   []MockDnsProxyRoute{},
		authRealm:        "test-realm",
		authChallenge:    "test-challenge",
		sessionCookie:    "session=test-session",
		version:          "4.3.6.3", // Default version
		systemMode: MockSystemMode{
			Active:   "router",
			Selected: "router",
		},
	}

	// Initialize default interfaces
	m.interfaces["Wireguard0"] = &MockInterface{
		ID:          "Wireguard0",
		Type:        InterfaceTypeWireguard,
		Description: "Test WireGuard interface",
		Address:     "10.0.0.1/24",
		Connected:   StateConnected,
		Link:        StateUp,
		State:       StateUp,
		DefaultGw:   false,
	}
	m.interfaces["ISP"] = &MockInterface{
		ID:        "ISP",
		Type:      InterfaceTypePPPoE,
		Connected: StateConnected,
		Link:      StateUp,
		State:     StateUp,
		DefaultGw: false,
	}

	// Initialize default SC interfaces
	m.scInterfaces["Wireguard0"] = &MockScInterface{
		Description: "Test WireGuard interface",
		IP: MockIP{
			Address: "10.0.0.1/24",
		},
		Wireguard: MockWireguard{
			Asc: MockAsc{
				Jc:   "3",
				Jmin: "50",
				Jmax: "1000",
				S1:   "86",
				S2:   "3",
				H1:   "1",
				H2:   "2",
				H3:   "3",
				H4:   "4",
			},
			Peer: []MockPeer{},
		},
	}

	// Initialize default routes
	m.routes = []MockRoute{
		{
			Network:   "192.168.1.0",
			Host:      "192.168.1.0",
			Mask:      "255.255.255.0",
			Interface: "Wireguard0",
			Auto:      false,
		},
	}

	// Initialize default DNS records
	m.dnsRecords["example.com"] = "1.2.3.4"
	m.dnsRecords["test.local"] = "192.168.1.50"

	// Initialize default hotspot devices
	m.hotspotDevices = []MockHost{
		{
			Name:       "test-device-1",
			Mac:        "aa:bb:cc:dd:ee:ff",
			IP:         "192.168.1.100",
			Hostname:   "device1",
			Registered: false,
			Link:       "up",
			Via:        "ISP",
		},
		{
			Name:       "test-device-2",
			Mac:        "11:22:33:44:55:66",
			IP:         "192.168.1.101",
			Hostname:   "device2",
			Registered: false,
			Link:       "up",
			Via:        "ISP",
		},
	}

	// Apply functional options to override defaults
	for _, opt := range opts {
		opt(m)
	}

	// Store initial state for reset functionality
	m.initialState = m.captureState()

	return m
}

// captureState creates a deep copy of the current state
func (m *MockRouter) captureState() MockRouterState {
	state := MockRouterState{
		Interfaces:       make(map[string]*MockInterface),
		ScInterfaces:     make(map[string]*MockScInterface),
		Routes:           make([]MockRoute, len(m.routes)),
		DNSRecords:       make(map[string]string),
		HotspotDevices:   make([]MockHost, len(m.hotspotDevices)),
		SystemMode:       m.systemMode,
		DnsRoutingGroups: make([]MockDnsRoutingGroup, len(m.dnsRoutingGroups)),
		DnsProxyRoutes:   make([]MockDnsProxyRoute, len(m.dnsProxyRoutes)),
	}

	// Deep copy interfaces
	for id, iface := range m.interfaces {
		ifaceCopy := *iface
		state.Interfaces[id] = &ifaceCopy
	}

	// Deep copy SC interfaces
	for id, scIface := range m.scInterfaces {
		scIfaceCopy := *scIface
		state.ScInterfaces[id] = &scIfaceCopy
	}

	// Copy routes
	copy(state.Routes, m.routes)

	// Copy DNS records
	maps.Copy(state.DNSRecords, m.dnsRecords)

	// Copy hotspot devices
	copy(state.HotspotDevices, m.hotspotDevices)

	// Copy DNS-routing groups
	copy(state.DnsRoutingGroups, m.dnsRoutingGroups)

	// Copy DNS proxy routes
	copy(state.DnsProxyRoutes, m.dnsProxyRoutes)

	return state
}

// GetState returns a snapshot of the current mock state (for test assertions)
func (m *MockRouter) GetState() MockRouterState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.captureState()
}

// ResetState resets the mock to its initial state
func (m *MockRouter) ResetState() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Restore interfaces
	m.interfaces = make(map[string]*MockInterface)
	for id, iface := range m.initialState.Interfaces {
		ifaceCopy := *iface
		m.interfaces[id] = &ifaceCopy
	}

	// Restore SC interfaces
	m.scInterfaces = make(map[string]*MockScInterface)
	for id, scIface := range m.initialState.ScInterfaces {
		scIfaceCopy := *scIface
		m.scInterfaces[id] = &scIfaceCopy
	}

	// Restore routes
	m.routes = make([]MockRoute, len(m.initialState.Routes))
	copy(m.routes, m.initialState.Routes)

	// Restore DNS records
	m.dnsRecords = make(map[string]string)
	maps.Copy(m.dnsRecords, m.initialState.DNSRecords)

	// Restore hotspot devices
	m.hotspotDevices = make([]MockHost, len(m.initialState.HotspotDevices))
	copy(m.hotspotDevices, m.initialState.HotspotDevices)

	// Restore system mode
	m.systemMode = m.initialState.SystemMode

	// Restore DNS-routing groups
	m.dnsRoutingGroups = make([]MockDnsRoutingGroup, len(m.initialState.DnsRoutingGroups))
	copy(m.dnsRoutingGroups, m.initialState.DnsRoutingGroups)

	// Restore DNS proxy routes
	m.dnsProxyRoutes = make([]MockDnsProxyRoute, len(m.initialState.DnsProxyRoutes))
	copy(m.dnsProxyRoutes, m.initialState.DnsProxyRoutes)
}

// NewMockRouterServer creates an httptest.Server with the mock router
// It registers all endpoint handlers and returns a test server ready for use
func NewMockRouterServer(opts ...MockRouterOption) *httptest.Server {
	m := NewMockRouter(opts...)
	mux := http.NewServeMux()

	// Auth endpoint
	mux.HandleFunc("/auth", m.handleAuth)

	// Version endpoint
	mux.HandleFunc("/rci/show/version", m.handleVersion)

	// Interface endpoints
	mux.HandleFunc("/rci/show/interface", m.handleInterfaces)
	mux.HandleFunc("/rci/show/interface/", m.handleInterface)

	// SC interface endpoints
	mux.HandleFunc("/rci/show/sc/interface", m.handleScInterfaces)
	mux.HandleFunc("/rci/show/sc/interface/", m.handleScInterface)

	// Route endpoints
	mux.HandleFunc("/rci/ip/route", m.handleRoutes)

	// DNS endpoints
	mux.HandleFunc("/rci/show/ip/name-server", m.handleDnsRecords)

	// DNS-routing endpoints
	mux.HandleFunc("/rci/object-group/fqdn", m.handleObjectGroupFqdn)
	mux.HandleFunc("/rci/dns-proxy/route", m.handleDnsProxyRoute)

	// Hotspot endpoint
	mux.HandleFunc("/rci/show/ip/hotspot", m.handleHotspot)

	// Parse endpoint
	mux.HandleFunc("/rci/", m.handleParse)

	// Running config endpoint
	mux.HandleFunc("/rci/show/running-config", m.handleRunningConfig)

	// System mode endpoint
	mux.HandleFunc("/rci/show/system/mode", m.handleSystemMode)

	return httptest.NewServer(mux)
}

// SetupMockRouterForTest is a convenience function for test suites with default config.
// It creates a mock server and automatically configures the global API client to use it.
// This is the recommended way to set up mock servers in tests.
func SetupMockRouterForTest(opts ...MockRouterOption) *httptest.Server {
	server := NewMockRouterServer(opts...)
	SetupTestConfig(server.URL)
	return server
}

// Placeholder endpoint handlers - these will be implemented in subsequent tasks
// For now, they return basic responses to satisfy the endpoint registration requirement

// handleAuth implements the authentication endpoint
// GET requests return 401 with authentication challenge headers
// POST requests validate credentials and return 200 on success
func (m *MockRouter) handleAuth(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	realm := m.authRealm
	challenge := m.authChallenge
	cookie := m.sessionCookie
	m.mu.RUnlock()

	switch r.Method {
	case http.MethodGet:
		// Return 401 with authentication challenge headers and session cookie
		// This initiates the challenge-response authentication flow
		w.Header().Set("x-ndm-realm", realm)
		w.Header().Set("x-ndm-challenge", challenge)
		w.Header().Set("set-cookie", cookie+"; Path=/")
		w.WriteHeader(http.StatusUnauthorized)
		return

	case http.MethodPost:
		// For the mock, we accept any credentials
		// In a real router, this would validate the MD5/SHA256 challenge-response
		var authRequest struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&authRequest); err != nil {
			// Invalid JSON body
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Mock accepts any non-empty credentials
		if authRequest.Login == "" || authRequest.Password == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Authentication successful
		w.WriteHeader(http.StatusOK)
		return

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *MockRouter) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	versionStr := m.version
	m.mu.RUnlock()

	// Return version information for the mock router
	version := gokeenrestapimodels.Version{
		Release:      versionStr,
		Title:        versionStr, // Just the version number (e.g., "5.0.1")
		Arch:         "mips",
		Manufacturer: "Keenetic",
		Vendor:       "Keenetic Ltd.",
		Series:       "KN",
		Model:        "KN-1010",
		HwVersion:    "1.0",
		Device:       "Keenetic",
		Description:  "Mock Keenetic Router",
		Ndm: gokeenrestapimodels.Ndm{
			Exact: versionStr,
			Cdate: "2024-01-01",
		},
		Bsp: gokeenrestapimodels.Bsp{
			Exact: versionStr,
			Cdate: "2024-01-01",
		},
		Ndw: gokeenrestapimodels.Ndw{
			Features:   "mock-features",
			Components: "mock-components",
		},
		Ndw4: gokeenrestapimodels.Ndw4{
			Version: "4.0",
		},
	}
	m.encodeJSON(w, version)
}

func (m *MockRouter) handleInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Support optional type filtering via query parameter
	typeFilter := r.URL.Query().Get("type")

	interfaces := make(map[string]gokeenrestapimodels.RciShowInterface)
	for id, iface := range m.interfaces {
		// Apply type filter if specified
		if typeFilter != "" && iface.Type != typeFilter {
			continue
		}

		interfaces[id] = gokeenrestapimodels.RciShowInterface{
			Id:          iface.ID,
			Type:        iface.Type,
			Description: iface.Description,
			Address:     iface.Address,
			Connected:   iface.Connected,
			Link:        iface.Link,
			State:       iface.State,
			DefaultGw:   iface.DefaultGw,
		}
	}
	m.encodeJSON(w, interfaces)
}

func (m *MockRouter) handleInterface(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	interfaceID := strings.TrimPrefix(r.URL.Path, "/rci/show/interface/")

	m.mu.RLock()
	defer m.mu.RUnlock()

	iface, exists := m.interfaces[interfaceID]
	if !exists {
		http.Error(w, "Interface not found", http.StatusNotFound)
		return
	}

	response := gokeenrestapimodels.RciShowInterface{
		Id:          iface.ID,
		Type:        iface.Type,
		Description: iface.Description,
		Address:     iface.Address,
		Connected:   iface.Connected,
		Link:        iface.Link,
		State:       iface.State,
		DefaultGw:   iface.DefaultGw,
	}
	m.encodeJSON(w, response)
}

func (m *MockRouter) handleScInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	interfaces := make(map[string]gokeenrestapimodels.RciShowScInterface)
	for id, scIface := range m.scInterfaces {
		// Convert MockScInterface to RciShowScInterface
		peers := make([]gokeenrestapimodels.Peer, len(scIface.Wireguard.Peer))
		for i, mockPeer := range scIface.Wireguard.Peer {
			allowIps := make([]gokeenrestapimodels.AllowIps, len(mockPeer.AllowedIPs))
			for j, mockAllowedIP := range mockPeer.AllowedIPs {
				allowIps[j] = gokeenrestapimodels.AllowIps{
					Address: mockAllowedIP.Address,
					Mask:    mockAllowedIP.Mask,
				}
			}

			peers[i] = gokeenrestapimodels.Peer{
				Key:     mockPeer.Key,
				Comment: mockPeer.Comment,
				Endpoint: gokeenrestapimodels.Endpoint{
					Address: mockPeer.Endpoint,
				},
				KeepaliveInterval: gokeenrestapimodels.KeepaliveInterval{
					Interval: mockPeer.KeepaliveInterval,
				},
				PresharedKey: mockPeer.PresharedKey,
				AllowIps:     allowIps,
			}
		}

		interfaces[id] = gokeenrestapimodels.RciShowScInterface{
			Description: scIface.Description,
			IP: gokeenrestapimodels.IP{
				Address: gokeenrestapimodels.Address{
					Address: scIface.IP.Address,
				},
			},
			Wireguard: gokeenrestapimodels.Wireguard{
				Asc: gokeenrestapimodels.Asc{
					Jc:   scIface.Wireguard.Asc.Jc,
					Jmin: scIface.Wireguard.Asc.Jmin,
					Jmax: scIface.Wireguard.Asc.Jmax,
					S1:   scIface.Wireguard.Asc.S1,
					S2:   scIface.Wireguard.Asc.S2,
					H1:   scIface.Wireguard.Asc.H1,
					H2:   scIface.Wireguard.Asc.H2,
					H3:   scIface.Wireguard.Asc.H3,
					H4:   scIface.Wireguard.Asc.H4,
				},
				Peer: peers,
			},
		}
	}
	m.encodeJSON(w, interfaces)
}

func (m *MockRouter) handleScInterface(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	interfaceID := strings.TrimPrefix(r.URL.Path, "/rci/show/sc/interface/")

	m.mu.RLock()
	defer m.mu.RUnlock()

	scIface, exists := m.scInterfaces[interfaceID]
	if !exists {
		http.Error(w, "Interface not found", http.StatusNotFound)
		return
	}

	// Convert MockScInterface to RciShowScInterface
	peers := make([]gokeenrestapimodels.Peer, len(scIface.Wireguard.Peer))
	for i, mockPeer := range scIface.Wireguard.Peer {
		allowIps := make([]gokeenrestapimodels.AllowIps, len(mockPeer.AllowedIPs))
		for j, mockAllowedIP := range mockPeer.AllowedIPs {
			allowIps[j] = gokeenrestapimodels.AllowIps{
				Address: mockAllowedIP.Address,
				Mask:    mockAllowedIP.Mask,
			}
		}

		peers[i] = gokeenrestapimodels.Peer{
			Key:     mockPeer.Key,
			Comment: mockPeer.Comment,
			Endpoint: gokeenrestapimodels.Endpoint{
				Address: mockPeer.Endpoint,
			},
			KeepaliveInterval: gokeenrestapimodels.KeepaliveInterval{
				Interval: mockPeer.KeepaliveInterval,
			},
			PresharedKey: mockPeer.PresharedKey,
			AllowIps:     allowIps,
		}
	}

	response := gokeenrestapimodels.RciShowScInterface{
		Description: scIface.Description,
		IP: gokeenrestapimodels.IP{
			Address: gokeenrestapimodels.Address{
				Address: scIface.IP.Address,
			},
		},
		Wireguard: gokeenrestapimodels.Wireguard{
			Asc: gokeenrestapimodels.Asc{
				Jc:   scIface.Wireguard.Asc.Jc,
				Jmin: scIface.Wireguard.Asc.Jmin,
				Jmax: scIface.Wireguard.Asc.Jmax,
				S1:   scIface.Wireguard.Asc.S1,
				S2:   scIface.Wireguard.Asc.S2,
				H1:   scIface.Wireguard.Asc.H1,
				H2:   scIface.Wireguard.Asc.H2,
				H3:   scIface.Wireguard.Asc.H3,
				H4:   scIface.Wireguard.Asc.H4,
			},
			Peer: peers,
		},
	}
	m.encodeJSON(w, response)
}

func (m *MockRouter) handleRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Support optional interface filtering via query parameter
	interfaceFilter := r.URL.Query().Get("interface")

	routes := make([]gokeenrestapimodels.RciIpRoute, 0, len(m.routes))
	for _, route := range m.routes {
		// Apply interface filter if specified
		if interfaceFilter != "" && route.Interface != interfaceFilter {
			continue
		}

		routes = append(routes, gokeenrestapimodels.RciIpRoute{
			Network:   route.Network,
			Host:      route.Host,
			Mask:      route.Mask,
			Interface: route.Interface,
			Auto:      route.Auto,
		})
	}
	m.encodeJSON(w, routes)
}

func (m *MockRouter) handleDnsRecords(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// DNS records are returned in a specific format with a "static" key
	response := map[string]map[string]string{
		"static": m.dnsRecords,
	}
	m.encodeJSON(w, response)
}

func (m *MockRouter) handleObjectGroupFqdn(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Convert internal DNS-routing groups to API response format
	response := make(gokeenrestapimodels.ObjectGroupFqdnResponse)
	for _, group := range m.dnsRoutingGroups {
		entries := make([]gokeenrestapimodels.ObjectGroupFqdnEntry, 0, len(group.Domains))
		for _, domain := range group.Domains {
			entries = append(entries, gokeenrestapimodels.ObjectGroupFqdnEntry{
				Address: domain,
			})
		}

		response[group.Name] = gokeenrestapimodels.ObjectGroupFqdn{
			Include: entries,
		}
	}

	m.encodeJSON(w, response)
}

func (m *MockRouter) handleDnsProxyRoute(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Convert internal dns-proxy routes to API response format
	routes := make(gokeenrestapimodels.DnsProxyRouteResponse, 0, len(m.dnsProxyRoutes))
	for _, route := range m.dnsProxyRoutes {
		routes = append(routes, gokeenrestapimodels.DnsProxyRoute{
			Group:     route.GroupName,
			Interface: route.InterfaceID,
		})
	}

	m.encodeJSON(w, routes)
}

func (m *MockRouter) handleHotspot(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	hosts := make([]gokeenrestapimodels.Host, 0, len(m.hotspotDevices))
	for _, device := range m.hotspotDevices {
		hosts = append(hosts, gokeenrestapimodels.Host{
			Name:       device.Name,
			Mac:        device.Mac,
			IP:         device.IP,
			Hostname:   device.Hostname,
			Registered: device.Registered,
			Link:       device.Link,
			Via:        device.Via,
		})
	}

	hotspot := gokeenrestapimodels.RciShowIpHotspot{
		Host: hosts,
	}
	m.encodeJSON(w, hotspot)
}

func (m *MockRouter) handleParse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var requests []gokeenrestapimodels.ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Process each request and generate matching response
	responses := make([]gokeenrestapimodels.ParseResponse, len(requests))
	for i, req := range requests {
		responses[i] = m.routeParseCommand(req.Parse)
	}
	m.encodeJSON(w, responses)
}

// routeParseCommand parses a command string and routes it to the appropriate handler
// It returns a ParseResponse with either success or error status
func (m *MockRouter) routeParseCommand(command string) gokeenrestapimodels.ParseResponse {
	// Trim whitespace
	command = strings.TrimSpace(command)

	// Empty command - return success to match old mock behavior
	if command == "" {
		return m.successResponse("Empty command accepted")
	}

	// Split command into tokens
	tokens := strings.Fields(command)
	if len(tokens) == 0 {
		return m.errorResponse("Empty command")
	}

	// Route based on command pattern
	switch {
	case tokens[0] == "system" && len(tokens) >= 2:
		// system configuration save - just return success
		if tokens[1] == "configuration" && len(tokens) >= 3 && tokens[2] == "save" {
			return m.successResponse("Configuration saved")
		}
		return m.errorResponse(fmt.Sprintf("Unknown system command: %s", strings.Join(tokens[1:], " ")))

	case tokens[0] == "interface" && len(tokens) >= 3:
		// interface <id> up|down
		// interface <id> wireguard asc ...
		interfaceID := tokens[1]

		// Check for interface state change (up/down)
		if tokens[2] == "up" || tokens[2] == "down" {
			return m.parseInterfaceState(interfaceID, tokens[2])
		}

		// Check for wireguard configuration
		if tokens[2] == "wireguard" && len(tokens) >= 4 {
			return m.parseAwgConfig(interfaceID, tokens[3:])
		}

		// Check for interface creation
		if tokens[2] == "create" {
			return m.parseCreateInterface(interfaceID, tokens[3:])
		}

		// Check for ip global auto command
		if tokens[2] == "ip" && len(tokens) >= 5 && tokens[3] == "global" && tokens[4] == "auto" {
			return m.successResponse(fmt.Sprintf("IP global auto enabled for interface %s", interfaceID))
		}

		// Check for no ip global command
		if tokens[2] == "no" && len(tokens) >= 5 && tokens[3] == "ip" && tokens[4] == "global" {
			return m.successResponse(fmt.Sprintf("IP global disabled for interface %s", interfaceID))
		}

		return m.errorResponse(fmt.Sprintf("Unknown interface command: %s", strings.Join(tokens[2:], " ")))

	case tokens[0] == "ip" && len(tokens) >= 2:
		switch tokens[1] {
		case "route":
			// ip route <network> <mask> <interface> [auto]
			return m.parseAddRoute(tokens[2:])
		case "host":
			// ip host <domain> <ip>
			return m.parseAddDnsRecord(tokens[2:])
		default:
			return m.errorResponse(fmt.Sprintf("Unknown ip subcommand: %s", tokens[1]))
		}

	case tokens[0] == "object-group" && len(tokens) >= 3:
		// object-group fqdn <group-name> [include <domain>]
		if tokens[1] == "fqdn" {
			groupName := tokens[2]
			if len(tokens) >= 5 && tokens[3] == "include" {
				// object-group fqdn <group-name> include <domain>
				domain := tokens[4]
				return m.parseAddDomainToGroup(groupName, domain)
			}
			// object-group fqdn <group-name>
			return m.parseCreateObjectGroup(groupName)
		}
		return m.errorResponse(fmt.Sprintf("Unknown object-group type: %s", tokens[1]))

	case tokens[0] == "dns-proxy" && len(tokens) >= 5:
		// dns-proxy route object-group <group-name> <interface-id> auto
		if tokens[1] == "route" && tokens[2] == "object-group" {
			groupName := tokens[3]
			interfaceID := tokens[4]
			mode := "auto"
			if len(tokens) >= 6 {
				mode = tokens[5]
			}
			return m.parseCreateDnsProxyRoute(groupName, interfaceID, mode)
		}
		return m.errorResponse("Invalid dns-proxy command: expected 'dns-proxy route object-group <group-name> <interface-id> auto'")

	case tokens[0] == "no" && len(tokens) >= 2:
		switch tokens[1] {
		case "object-group":
			// no object-group fqdn <group-name> [include <domain>]
			if len(tokens) >= 4 && tokens[2] == "fqdn" {
				groupName := tokens[3]
				// Check if this is removing a domain from the group
				if len(tokens) >= 6 && tokens[4] == "include" {
					domain := tokens[5]
					return m.parseRemoveDomainFromGroup(groupName, domain)
				}
				// Otherwise, delete the entire group
				return m.parseDeleteObjectGroup(groupName)
			}
			return m.errorResponse("Invalid object-group deletion: expected 'no object-group fqdn <group-name> [include <domain>]'")
		case "dns-proxy":
			// no dns-proxy route object-group <group-name> <interface-id>
			if len(tokens) >= 6 && tokens[2] == "route" && tokens[3] == "object-group" {
				groupName := tokens[4]
				interfaceID := tokens[5]
				return m.parseDeleteDnsProxyRoute(groupName, interfaceID)
			}
			return m.errorResponse("Invalid dns-proxy deletion: expected 'no dns-proxy route object-group <group-name> <interface-id>'")
		case "ip":
			if len(tokens) >= 3 {
				switch tokens[2] {
				case "route":
					// no ip route <network> <mask> <interface>
					return m.parseDeleteRoute(tokens[3:])
				case "host":
					// no ip host <domain>
					return m.parseDeleteDnsRecord(tokens[3:])
				default:
					return m.errorResponse(fmt.Sprintf("Unknown no ip subcommand: %s", tokens[2]))
				}
			}
			return m.errorResponse("Incomplete no ip command")
		case "known":
			if len(tokens) >= 3 && tokens[2] == "host" {
				// no known host "<mac>"
				return m.parseDeleteKnownHost(tokens[3:])
			}
			return m.errorResponse("Invalid known command: expected 'no known host <mac>'")
		default:
			return m.errorResponse(fmt.Sprintf("Unknown no subcommand: %s", tokens[1]))
		}

	default:
		return m.errorResponse(fmt.Sprintf("Unknown command: %s", tokens[0]))
	}
}

// errorResponse creates a ParseResponse with error status
func (m *MockRouter) errorResponse(message string) gokeenrestapimodels.ParseResponse {
	return gokeenrestapimodels.ParseResponse{
		Parse: gokeenrestapimodels.Parse{
			Status: []gokeenrestapimodels.Status{
				{
					Status:  StatusError,
					Code:    "1",
					Message: message,
				},
			},
		},
	}
}

// successResponse creates a ParseResponse with success status
func (m *MockRouter) successResponse(message string) gokeenrestapimodels.ParseResponse {
	return gokeenrestapimodels.ParseResponse{
		Parse: gokeenrestapimodels.Parse{
			Status: []gokeenrestapimodels.Status{
				{
					Status:  StatusOK,
					Code:    "0",
					Message: message,
				},
			},
		},
	}
}

func (m *MockRouter) handleRunningConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var configLines []string

	// Use static config if provided, otherwise generate from state
	if len(m.staticRunningConfig) > 0 {
		configLines = m.staticRunningConfig
	} else {
		// Generate config lines from current state
		// Add system mode configuration
		configLines = append(configLines, fmt.Sprintf("system mode %s", m.systemMode.Selected))

		// Add interface configurations
		for id, iface := range m.interfaces {
			configLines = append(configLines, fmt.Sprintf("interface %s", id))
			if iface.Description != "" {
				configLines = append(configLines, fmt.Sprintf("  description \"%s\"", iface.Description))
			}
			if iface.Address != "" {
				configLines = append(configLines, fmt.Sprintf("  ip address %s", iface.Address))
			}
			// Add interface state
			if iface.State == StateUp {
				configLines = append(configLines, "  no shutdown")
			} else {
				configLines = append(configLines, "  shutdown")
			}
		}

		// Add route configurations
		for _, route := range m.routes {
			if route.Auto {
				configLines = append(configLines, fmt.Sprintf("ip route %s %s %s auto", route.Network, route.Mask, route.Interface))
			} else {
				configLines = append(configLines, fmt.Sprintf("ip route %s %s %s", route.Network, route.Mask, route.Interface))
			}
		}

		// Add DNS record configurations
		for domain, ip := range m.dnsRecords {
			configLines = append(configLines, fmt.Sprintf("ip host %s %s", domain, ip))
		}

		// Add DNS-routing object-group configurations
		for _, group := range m.dnsRoutingGroups {
			configLines = append(configLines, fmt.Sprintf("object-group fqdn %s", group.Name))
			for _, domain := range group.Domains {
				configLines = append(configLines, fmt.Sprintf("  include %s", domain))
			}
		}

		// Add dns-proxy route configurations
		for _, route := range m.dnsProxyRoutes {
			configLines = append(configLines, fmt.Sprintf("dns-proxy route object-group %s %s %s", route.GroupName, route.InterfaceID, route.Mode))
		}
	}

	runningConfig := gokeenrestapimodels.RunningConfig{
		Message: configLines,
	}
	m.encodeJSON(w, runningConfig)
}

func (m *MockRouter) handleSystemMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	systemMode := gokeenrestapimodels.SystemMode{
		Active:   m.systemMode.Active,
		Selected: m.systemMode.Selected,
	}
	m.encodeJSON(w, systemMode)
}

// parseAddRoute handles the "ip route <network> <mask> <interface> [auto]" command
// It validates the interface exists and adds the route to the mock state
func (m *MockRouter) parseAddRoute(tokens []string) gokeenrestapimodels.ParseResponse {
	// Expected format: <network> <mask> <interface> [auto]
	// Minimum 3 tokens required: network, mask, interface
	if len(tokens) < 3 {
		return m.errorResponse("Invalid route command: expected 'ip route <network> <mask> <interface> [auto]'")
	}

	network := tokens[0]
	mask := tokens[1]
	interfaceID := tokens[2]
	auto := false
	if len(tokens) >= 4 && tokens[3] == "auto" {
		auto = true
	}

	// Validate interface exists
	m.mu.RLock()
	_, exists := m.interfaces[interfaceID]
	m.mu.RUnlock()

	if !exists {
		return m.errorResponse(fmt.Sprintf("Interface '%s' does not exist", interfaceID))
	}

	// Add route to state
	m.mu.Lock()
	defer m.mu.Unlock()

	newRoute := MockRoute{
		Network:   network,
		Host:      network, // Host is typically the same as network for static routes
		Mask:      mask,
		Interface: interfaceID,
		Auto:      auto,
	}

	m.routes = append(m.routes, newRoute)

	return m.successResponse(fmt.Sprintf("Route %s %s added to interface %s", network, mask, interfaceID))
}

// parseDeleteRoute handles the "no ip route <network> <mask> <interface>" command
// It removes the matching route from the mock state
func (m *MockRouter) parseDeleteRoute(tokens []string) gokeenrestapimodels.ParseResponse {
	// Expected format: <network> <mask> <interface>
	// Minimum 3 tokens required: network, mask, interface
	if len(tokens) < 3 {
		return m.errorResponse("Invalid route deletion command: expected 'no ip route <network> <mask> <interface>'")
	}

	network := tokens[0]
	mask := tokens[1]
	interfaceID := tokens[2]

	// Remove route from state
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find and remove the matching route
	found := false
	newRoutes := make([]MockRoute, 0, len(m.routes))
	for _, route := range m.routes {
		// Match by network, mask, and interface
		if route.Network == network && route.Mask == mask && route.Interface == interfaceID {
			found = true
			// Skip this route (don't add to newRoutes)
			continue
		}
		newRoutes = append(newRoutes, route)
	}

	// If route not found, still return success (matches old mock behavior)
	// The old mock didn't validate whether routes existed
	m.routes = newRoutes

	if found {
		return m.successResponse(fmt.Sprintf("Route %s %s removed from interface %s", network, mask, interfaceID))
	}
	return m.successResponse(fmt.Sprintf("Route %s %s on interface %s (not found, but command accepted)", network, mask, interfaceID))
}

// parseAddDnsRecord handles the "ip host <domain> <ip>" command
// It adds a DNS record to the mock state
func (m *MockRouter) parseAddDnsRecord(tokens []string) gokeenrestapimodels.ParseResponse {
	// Expected format: <domain> <ip>
	// Minimum 2 tokens required: domain, ip
	if len(tokens) < 2 {
		return m.errorResponse("Invalid DNS command: expected 'ip host <domain> <ip>'")
	}

	domain := tokens[0]
	ip := tokens[1]

	// Add DNS record to state
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dnsRecords[domain] = ip

	return m.successResponse(fmt.Sprintf("DNS record %s -> %s added", domain, ip))
}

// parseDeleteDnsRecord handles the "no ip host <domain>" command
// It removes the DNS record from the mock state
func (m *MockRouter) parseDeleteDnsRecord(tokens []string) gokeenrestapimodels.ParseResponse {
	// Expected format: <domain>
	// Minimum 1 token required: domain
	if len(tokens) < 1 {
		return m.errorResponse("Invalid DNS deletion command: expected 'no ip host <domain>'")
	}

	domain := tokens[0]

	// Remove DNS record from state
	m.mu.Lock()
	defer m.mu.Unlock()

	// Delete DNS record (matches old mock behavior - doesn't error if not found)
	delete(m.dnsRecords, domain)

	return m.successResponse(fmt.Sprintf("DNS record for domain '%s' removed", domain))
}

// parseInterfaceState handles the "interface <id> up|down" command
// It updates the interface state fields (connected, link, state)
func (m *MockRouter) parseInterfaceState(interfaceID string, state string) gokeenrestapimodels.ParseResponse {
	// Validate state parameter
	if state != "up" && state != "down" {
		return m.errorResponse(fmt.Sprintf("Invalid interface state: %s (expected 'up' or 'down')", state))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if interface exists
	iface, exists := m.interfaces[interfaceID]
	if !exists {
		return m.errorResponse(fmt.Sprintf("Interface '%s' does not exist", interfaceID))
	}

	// Update interface state fields based on up/down
	if state == "up" {
		iface.Connected = StateConnected
		iface.Link = StateUp
		iface.State = StateUp
	} else {
		iface.Connected = StateDisconnected
		iface.Link = StateDown
		iface.State = StateDown
	}

	return m.successResponse(fmt.Sprintf("Interface %s set to %s", interfaceID, state))
}

// parseAwgConfig handles the "interface <id> wireguard asc jc <value> ..." command
// It updates AWG parameters in SC interface state
func (m *MockRouter) parseAwgConfig(interfaceID string, tokens []string) gokeenrestapimodels.ParseResponse {
	// Expected format: asc jc <value> jmin <value> jmax <value> s1 <value> s2 <value> h1 <value> h2 <value> h3 <value> h4 <value>
	// Minimum: asc jc <value>
	if len(tokens) < 3 || tokens[0] != "asc" {
		return m.errorResponse("Invalid AWG config command: expected 'interface <id> wireguard asc jc <value> ...'")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if SC interface exists
	scIface, exists := m.scInterfaces[interfaceID]
	if !exists {
		return m.errorResponse(fmt.Sprintf("SC interface '%s' does not exist", interfaceID))
	}

	// Parse AWG parameters from tokens
	// Format: jc <value> jmin <value> jmax <value> s1 <value> s2 <value> h1 <value> h2 <value> h3 <value> h4 <value>
	params := make(map[string]string)
	for i := 1; i < len(tokens)-1; i += 2 {
		paramName := strings.ToLower(tokens[i])
		paramValue := tokens[i+1]
		params[paramName] = paramValue
	}

	// Update AWG parameters if provided
	if val, ok := params["jc"]; ok {
		scIface.Wireguard.Asc.Jc = val
	}
	if val, ok := params["jmin"]; ok {
		scIface.Wireguard.Asc.Jmin = val
	}
	if val, ok := params["jmax"]; ok {
		scIface.Wireguard.Asc.Jmax = val
	}
	if val, ok := params["s1"]; ok {
		scIface.Wireguard.Asc.S1 = val
	}
	if val, ok := params["s2"]; ok {
		scIface.Wireguard.Asc.S2 = val
	}
	if val, ok := params["h1"]; ok {
		scIface.Wireguard.Asc.H1 = val
	}
	if val, ok := params["h2"]; ok {
		scIface.Wireguard.Asc.H2 = val
	}
	if val, ok := params["h3"]; ok {
		scIface.Wireguard.Asc.H3 = val
	}
	if val, ok := params["h4"]; ok {
		scIface.Wireguard.Asc.H4 = val
	}

	return m.successResponse(fmt.Sprintf("AWG parameters updated for interface %s", interfaceID))
}

// parseCreateInterface handles interface creation commands
// It adds the interface to both interfaces and SC interfaces state
func (m *MockRouter) parseCreateInterface(interfaceID string, tokens []string) gokeenrestapimodels.ParseResponse {
	// Expected format: type <type> [description <desc>] [address <addr>]
	// Minimum: type <type>
	if len(tokens) < 2 || tokens[0] != "type" {
		return m.errorResponse("Invalid interface creation command: expected 'interface <id> create type <type> ...'")
	}

	interfaceType := tokens[1]

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if interface already exists
	if _, exists := m.interfaces[interfaceID]; exists {
		return m.errorResponse(fmt.Sprintf("Interface '%s' already exists", interfaceID))
	}

	// Parse optional parameters
	description := ""
	address := ""

	for i := 2; i < len(tokens)-1; i += 2 {
		paramName := strings.ToLower(tokens[i])
		paramValue := tokens[i+1]

		switch paramName {
		case "description":
			description = paramValue
		case "address":
			address = paramValue
		}
	}

	// Create new interface
	newInterface := &MockInterface{
		ID:          interfaceID,
		Type:        interfaceType,
		Description: description,
		Address:     address,
		Connected:   StateDisconnected,
		Link:        StateDown,
		State:       StateDown,
		DefaultGw:   false,
	}

	m.interfaces[interfaceID] = newInterface

	// If it's a WireGuard interface, also create SC interface entry
	if interfaceType == InterfaceTypeWireguard {
		newScInterface := &MockScInterface{
			Description: description,
			IP: MockIP{
				Address: address,
			},
			Wireguard: MockWireguard{
				Asc: MockAsc{
					Jc:   "0",
					Jmin: "0",
					Jmax: "0",
					S1:   "0",
					S2:   "0",
					H1:   "0",
					H2:   "0",
					H3:   "0",
					H4:   "0",
				},
				Peer: []MockPeer{},
			},
		}
		m.scInterfaces[interfaceID] = newScInterface
	}

	return m.successResponse(fmt.Sprintf("Interface %s created with type %s", interfaceID, interfaceType))
}

// parseDeleteKnownHost handles the "no known host <mac>" command
// It removes the host from the hotspot state by MAC address
func (m *MockRouter) parseDeleteKnownHost(tokens []string) gokeenrestapimodels.ParseResponse {
	// Expected format: "<mac>" (MAC address may be quoted)
	// Minimum 1 token required: mac address
	if len(tokens) < 1 {
		return m.errorResponse("Invalid known host deletion command: expected 'no known host \"<mac>\"'")
	}

	// Extract MAC address - remove quotes if present
	mac := tokens[0]
	mac = strings.Trim(mac, "\"")

	// Remove host from hotspot state
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find and remove the matching host by MAC address
	found := false
	newHosts := make([]MockHost, 0, len(m.hotspotDevices))
	for _, host := range m.hotspotDevices {
		// Match by MAC address (case-insensitive)
		if strings.EqualFold(host.Mac, mac) {
			found = true
			// Skip this host (don't add to newHosts)
			continue
		}
		newHosts = append(newHosts, host)
	}

	// Update hotspot devices (matches old mock behavior - doesn't error if not found)
	m.hotspotDevices = newHosts

	if found {
		return m.successResponse(fmt.Sprintf("Known host with MAC '%s' removed", mac))
	}
	return m.successResponse(fmt.Sprintf("Known host with MAC '%s' (not found, but command accepted)", mac))
}

// parseCreateObjectGroup handles the "object-group fqdn <group-name>" command
// It creates a new DNS-routing object-group in the mock state
func (m *MockRouter) parseCreateObjectGroup(groupName string) gokeenrestapimodels.ParseResponse {
	if groupName == "" {
		return m.errorResponse("Invalid object-group command: group name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if group already exists
	for _, group := range m.dnsRoutingGroups {
		if group.Name == groupName {
			// Group already exists - just return success (idempotent)
			return m.successResponse(fmt.Sprintf("Object-group '%s' already exists", groupName))
		}
	}

	// Create new group
	newGroup := MockDnsRoutingGroup{
		Name:    groupName,
		Domains: []string{},
	}
	m.dnsRoutingGroups = append(m.dnsRoutingGroups, newGroup)

	return m.successResponse(fmt.Sprintf("Object-group '%s' created", groupName))
}

// parseAddDomainToGroup handles the "object-group fqdn <group-name> include <domain>" command
// It adds a domain to an existing object-group
func (m *MockRouter) parseAddDomainToGroup(groupName, domain string) gokeenrestapimodels.ParseResponse {
	if groupName == "" {
		return m.errorResponse("Invalid object-group command: group name cannot be empty")
	}
	if domain == "" {
		return m.errorResponse("Invalid object-group command: domain cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the group
	for i := range m.dnsRoutingGroups {
		if m.dnsRoutingGroups[i].Name == groupName {
			// Add domain to group (allow duplicates for simplicity)
			m.dnsRoutingGroups[i].Domains = append(m.dnsRoutingGroups[i].Domains, domain)
			return m.successResponse(fmt.Sprintf("Domain '%s' added to object-group '%s'", domain, groupName))
		}
	}

	return m.errorResponse(fmt.Sprintf("Object-group '%s' does not exist", groupName))
}

// parseCreateDnsProxyRoute handles the "dns-proxy route object-group <group-name> <interface-id> auto" command
// It creates a dns-proxy route linking a group to an interface
func (m *MockRouter) parseCreateDnsProxyRoute(groupName, interfaceID, mode string) gokeenrestapimodels.ParseResponse {
	if groupName == "" {
		return m.errorResponse("Invalid dns-proxy route command: group name cannot be empty")
	}
	if interfaceID == "" {
		return m.errorResponse("Invalid dns-proxy route command: interface ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate interface exists
	if _, exists := m.interfaces[interfaceID]; !exists {
		return m.errorResponse(fmt.Sprintf("Interface '%s' does not exist", interfaceID))
	}

	// Validate group exists
	groupExists := false
	for _, group := range m.dnsRoutingGroups {
		if group.Name == groupName {
			groupExists = true
			break
		}
	}
	if !groupExists {
		return m.errorResponse(fmt.Sprintf("Object-group '%s' does not exist", groupName))
	}

	// Check if route already exists
	for _, route := range m.dnsProxyRoutes {
		if route.GroupName == groupName && route.InterfaceID == interfaceID {
			// Route already exists - just return success (idempotent)
			return m.successResponse(fmt.Sprintf("Dns-proxy route for group '%s' to interface '%s' already exists", groupName, interfaceID))
		}
	}

	// Create new dns-proxy route
	newRoute := MockDnsProxyRoute{
		GroupName:   groupName,
		InterfaceID: interfaceID,
		Mode:        mode,
	}
	m.dnsProxyRoutes = append(m.dnsProxyRoutes, newRoute)

	return m.successResponse(fmt.Sprintf("Dns-proxy route created for group '%s' to interface '%s'", groupName, interfaceID))
}

// parseDeleteDnsProxyRoute handles the "no dns-proxy route object-group <group-name> <interface-id>" command
// It removes a dns-proxy route from the mock state
func (m *MockRouter) parseDeleteDnsProxyRoute(groupName, interfaceID string) gokeenrestapimodels.ParseResponse {
	if groupName == "" {
		return m.errorResponse("Invalid dns-proxy route deletion: group name cannot be empty")
	}
	if interfaceID == "" {
		return m.errorResponse("Invalid dns-proxy route deletion: interface ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find and remove the matching route
	found := false
	newRoutes := make([]MockDnsProxyRoute, 0, len(m.dnsProxyRoutes))
	for _, route := range m.dnsProxyRoutes {
		if route.GroupName == groupName && route.InterfaceID == interfaceID {
			found = true
			// Skip this route (don't add to newRoutes)
			continue
		}
		newRoutes = append(newRoutes, route)
	}

	m.dnsProxyRoutes = newRoutes

	if found {
		return m.successResponse(fmt.Sprintf("Dns-proxy route for group '%s' to interface '%s' removed", groupName, interfaceID))
	}
	return m.successResponse(fmt.Sprintf("Dns-proxy route for group '%s' to interface '%s' (not found, but command accepted)", groupName, interfaceID))
}

// parseRemoveDomainFromGroup handles the "no object-group fqdn <group-name> include <domain>" command
// It removes a specific domain from an object-group
func (m *MockRouter) parseRemoveDomainFromGroup(groupName, domain string) gokeenrestapimodels.ParseResponse {
	if groupName == "" {
		return m.errorResponse("Invalid domain removal: group name cannot be empty")
	}
	if domain == "" {
		return m.errorResponse("Invalid domain removal: domain cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the group and remove the domain
	for i := range m.dnsRoutingGroups {
		if m.dnsRoutingGroups[i].Name == groupName {
			// Find and remove the domain
			newDomains := make([]string, 0, len(m.dnsRoutingGroups[i].Domains))
			found := false
			for _, existingDomain := range m.dnsRoutingGroups[i].Domains {
				if existingDomain == domain {
					found = true
					// Skip this domain (don't add to newDomains)
					continue
				}
				newDomains = append(newDomains, existingDomain)
			}
			m.dnsRoutingGroups[i].Domains = newDomains

			if found {
				return m.successResponse(fmt.Sprintf("Domain '%s' removed from object-group '%s'", domain, groupName))
			}
			return m.successResponse(fmt.Sprintf("Domain '%s' not found in object-group '%s' (command accepted)", domain, groupName))
		}
	}

	return m.errorResponse(fmt.Sprintf("Object-group '%s' does not exist", groupName))
}

// parseDeleteObjectGroup handles the "no object-group fqdn <group-name>" command
// It removes an object-group from the mock state
func (m *MockRouter) parseDeleteObjectGroup(groupName string) gokeenrestapimodels.ParseResponse {
	if groupName == "" {
		return m.errorResponse("Invalid object-group deletion: group name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find and remove the matching group
	found := false
	newGroups := make([]MockDnsRoutingGroup, 0, len(m.dnsRoutingGroups))
	for _, group := range m.dnsRoutingGroups {
		if group.Name == groupName {
			found = true
			// Skip this group (don't add to newGroups)
			continue
		}
		newGroups = append(newGroups, group)
	}

	m.dnsRoutingGroups = newGroups

	if found {
		return m.successResponse(fmt.Sprintf("Object-group '%s' removed", groupName))
	}
	return m.successResponse(fmt.Sprintf("Object-group '%s' (not found, but command accepted)", groupName))
}

// encodeJSON is a helper function to safely encode JSON responses
func (m *MockRouter) encodeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode JSON: %v", err), http.StatusInternalServerError)
	}
}
