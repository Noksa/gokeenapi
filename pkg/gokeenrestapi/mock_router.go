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

// MockInterface represents a network interface in the mock router.
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

// MockRoute represents a static route in the mock router.
type MockRoute struct {
	Network   string
	Host      string
	Mask      string
	Interface string
	Auto      bool
}

// MockHost represents a device connected to the router (hotspot).
type MockHost struct {
	Name       string
	Mac        string
	IP         string
	Hostname   string
	Registered bool
	Link       string
	Via        string
}

// MockSystemMode represents the system mode configuration.
type MockSystemMode struct {
	Active   string
	Selected string
}

// MockIP represents IP configuration for SC interfaces.
type MockIP struct {
	Address string
}

// MockAsc represents AWG (Amnezia WireGuard) configuration parameters.
// AWG 1.0: Jc, Jmin, Jmax, S1, S2, H1–H4
// AWG 2.0 (KeeneticOS 5.1+): S3, S4, I1–I5
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
	S3   string
	S4   string
	I1   string
	I2   string
	I3   string
	I4   string
	I5   string
}

// MockPeer represents a WireGuard peer configuration.
type MockPeer struct {
	Key               string
	Comment           string
	Endpoint          string
	KeepaliveInterval int
	PresharedKey      string
	AllowedIPs        []MockAllowedIP
}

// MockAllowedIP represents an allowed IP range for a WireGuard peer.
type MockAllowedIP struct {
	Address string
	Mask    string
}

// MockWireguard represents WireGuard configuration for SC interfaces.
type MockWireguard struct {
	Asc  MockAsc
	Peer []MockPeer
}

// MockScInterface represents the SC (system configuration) view of an interface.
type MockScInterface struct {
	Description string
	IP          MockIP
	Wireguard   MockWireguard
}

// MockDnsRoutingGroup represents a DNS-routing object-group in the mock router.
type MockDnsRoutingGroup struct {
	Name    string
	Domains []string
}

// MockDnsProxyRoute represents a dns-proxy route in the mock router.
type MockDnsProxyRoute struct {
	GroupName   string
	InterfaceID string
	Mode        string // typically "auto"
}

// MockRouterState represents a snapshot of the mock router's state.
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

// MockRouter is a comprehensive mock implementation of the Keenetic router API.
// It maintains stateful behavior and supports all API endpoints for testing.
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
	initialState        MockRouterState
	staticRunningConfig []string
	version             string
	rciBodyOverride     []byte
	components          map[string]gokeenrestapimodels.Component
}

// MockRouterOption is a functional option for configuring the mock router.
type MockRouterOption func(*MockRouter)

// WithInterfaces sets custom initial interfaces for the mock router.
func WithInterfaces(interfaces []MockInterface) MockRouterOption {
	return func(m *MockRouter) {
		m.interfaces = make(map[string]*MockInterface)
		for i := range interfaces {
			iface := interfaces[i]
			m.interfaces[iface.ID] = &iface
		}
	}
}

// WithScInterfaces sets custom initial SC interfaces for the mock router.
func WithScInterfaces(scInterfaces map[string]MockScInterface) MockRouterOption {
	return func(m *MockRouter) {
		m.scInterfaces = make(map[string]*MockScInterface)
		for id, scIface := range scInterfaces {
			scIfaceCopy := scIface
			m.scInterfaces[id] = &scIfaceCopy
		}
	}
}

// WithRoutes sets custom initial routes for the mock router.
func WithRoutes(routes []MockRoute) MockRouterOption {
	return func(m *MockRouter) {
		m.routes = make([]MockRoute, len(routes))
		copy(m.routes, routes)
	}
}

// WithDNSRecords sets custom initial DNS records for the mock router.
func WithDNSRecords(records map[string]string) MockRouterOption {
	return func(m *MockRouter) {
		m.dnsRecords = make(map[string]string)
		maps.Copy(m.dnsRecords, records)
	}
}

// WithHotspotDevices sets custom initial hotspot devices for the mock router.
func WithHotspotDevices(devices []MockHost) MockRouterOption {
	return func(m *MockRouter) {
		m.hotspotDevices = make([]MockHost, len(devices))
		copy(m.hotspotDevices, devices)
	}
}

// WithSystemMode sets custom system mode for the mock router.
func WithSystemMode(mode MockSystemMode) MockRouterOption {
	return func(m *MockRouter) {
		m.systemMode = mode
	}
}

// WithStaticRunningConfig sets static running config lines (overrides generated config).
func WithStaticRunningConfig(lines []string) MockRouterOption {
	return func(m *MockRouter) {
		m.staticRunningConfig = make([]string, len(lines))
		copy(m.staticRunningConfig, lines)
	}
}

// WithDnsRoutingGroups sets custom initial DNS-routing groups for the mock router.
func WithDnsRoutingGroups(groups []MockDnsRoutingGroup, routes []MockDnsProxyRoute) MockRouterOption {
	return func(m *MockRouter) {
		m.dnsRoutingGroups = make([]MockDnsRoutingGroup, len(groups))
		copy(m.dnsRoutingGroups, groups)
		m.dnsProxyRoutes = make([]MockDnsProxyRoute, len(routes))
		copy(m.dnsProxyRoutes, routes)
	}
}

// WithVersion sets custom firmware version for the mock router.
func WithVersion(version string) MockRouterOption {
	return func(m *MockRouter) {
		m.version = version
	}
}

// WithRciBody overrides the /rci/ POST endpoint to return a fixed raw body.
// Useful for testing how callers handle malformed or unexpected responses.
func WithRciBody(body []byte) MockRouterOption {
	return func(m *MockRouter) {
		m.rciBodyOverride = body
	}
}

// WithComponents sets custom components for the mock router.
// The wireguard component is pre-installed by default.
func WithComponents(components map[string]gokeenrestapimodels.Component) MockRouterOption {
	return func(m *MockRouter) {
		m.components = make(map[string]gokeenrestapimodels.Component)
		maps.Copy(m.components, components)
	}
}

// NewMockRouter creates a new mock router with default state and optional configuration.
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
		version:          "4.3.6.3",
		systemMode:       MockSystemMode{Active: "router", Selected: "router"},
		components: map[string]gokeenrestapimodels.Component{
			"wireguard": {Group: "vpn", Installed: "yes", Version: "1.0"},
		},
	}

	m.interfaces["Wireguard0"] = &MockInterface{
		ID: "Wireguard0", Type: InterfaceTypeWireguard,
		Description: "Test WireGuard interface", Address: "10.0.0.1/24",
		Connected: StateConnected, Link: StateUp, State: StateUp,
	}
	m.interfaces["ISP"] = &MockInterface{
		ID: "ISP", Type: InterfaceTypePPPoE,
		Connected: StateConnected, Link: StateUp, State: StateUp,
	}

	m.scInterfaces["Wireguard0"] = &MockScInterface{
		Description: "Test WireGuard interface",
		IP:          MockIP{Address: "10.0.0.1/24"},
		Wireguard: MockWireguard{
			Asc: MockAsc{
				Jc: "3", Jmin: "50", Jmax: "1000",
				S1: "86", S2: "3",
				H1: "1", H2: "2", H3: "3", H4: "4",
			},
			Peer: []MockPeer{},
		},
	}

	m.routes = []MockRoute{
		{Network: "192.168.1.0", Host: "192.168.1.0", Mask: "255.255.255.0", Interface: "Wireguard0"},
	}

	m.dnsRecords["example.com"] = "1.2.3.4"
	m.dnsRecords["test.local"] = "192.168.1.50"

	m.hotspotDevices = []MockHost{
		{Name: "test-device-1", Mac: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.100", Hostname: "device1", Link: "up", Via: "ISP"},
		{Name: "test-device-2", Mac: "11:22:33:44:55:66", IP: "192.168.1.101", Hostname: "device2", Link: "up", Via: "ISP"},
	}

	for _, opt := range opts {
		opt(m)
	}

	m.initialState = m.captureState()
	return m
}

// captureState creates a deep copy of the current state.
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

	for id, iface := range m.interfaces {
		c := *iface
		state.Interfaces[id] = &c
	}
	for id, scIface := range m.scInterfaces {
		c := *scIface
		state.ScInterfaces[id] = &c
	}
	copy(state.Routes, m.routes)
	maps.Copy(state.DNSRecords, m.dnsRecords)
	copy(state.HotspotDevices, m.hotspotDevices)
	copy(state.DnsRoutingGroups, m.dnsRoutingGroups)
	copy(state.DnsProxyRoutes, m.dnsProxyRoutes)

	return state
}

// GetState returns a snapshot of the current mock state (for test assertions).
func (m *MockRouter) GetState() MockRouterState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.captureState()
}

// ResetState resets the mock to its initial state.
func (m *MockRouter) ResetState() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.interfaces = make(map[string]*MockInterface)
	for id, iface := range m.initialState.Interfaces {
		c := *iface
		m.interfaces[id] = &c
	}
	m.scInterfaces = make(map[string]*MockScInterface)
	for id, scIface := range m.initialState.ScInterfaces {
		c := *scIface
		m.scInterfaces[id] = &c
	}
	m.routes = make([]MockRoute, len(m.initialState.Routes))
	copy(m.routes, m.initialState.Routes)
	m.dnsRecords = make(map[string]string)
	maps.Copy(m.dnsRecords, m.initialState.DNSRecords)
	m.hotspotDevices = make([]MockHost, len(m.initialState.HotspotDevices))
	copy(m.hotspotDevices, m.initialState.HotspotDevices)
	m.systemMode = m.initialState.SystemMode
	m.dnsRoutingGroups = make([]MockDnsRoutingGroup, len(m.initialState.DnsRoutingGroups))
	copy(m.dnsRoutingGroups, m.initialState.DnsRoutingGroups)
	m.dnsProxyRoutes = make([]MockDnsProxyRoute, len(m.initialState.DnsProxyRoutes))
	copy(m.dnsProxyRoutes, m.initialState.DnsProxyRoutes)
}

// NewMockRouterServer creates an httptest.Server with the mock router.
// It registers all endpoint handlers and returns a test server ready for use.
func NewMockRouterServer(opts ...MockRouterOption) *httptest.Server {
	m := NewMockRouter(opts...)
	mux := http.NewServeMux()

	mux.HandleFunc("/auth", m.handleAuth)
	mux.HandleFunc("/rci/show/version", m.handleVersion)
	mux.HandleFunc("/rci/show/interface", m.handleInterfaces)
	mux.HandleFunc("/rci/show/interface/", m.handleInterface)
	mux.HandleFunc("/rci/show/sc/interface", m.handleScInterfaces)
	mux.HandleFunc("/rci/show/sc/interface/", m.handleScInterface)
	mux.HandleFunc("/rci/ip/route", m.handleRoutes)
	mux.HandleFunc("/rci/show/ip/route", m.handleShowIpRoute)
	mux.HandleFunc("/rci/show/ip/name-server", m.handleDnsRecords)
	mux.HandleFunc("/rci/object-group/fqdn", m.handleObjectGroupFqdn)
	mux.HandleFunc("/rci/dns-proxy/route", m.handleDnsProxyRoute)
	mux.HandleFunc("/rci/components/list", m.handleComponentsList)
	mux.HandleFunc("/rci/show/ip/hotspot", m.handleHotspot)
	mux.HandleFunc("/rci/show/running-config", m.handleRunningConfig)
	mux.HandleFunc("/rci/show/system/mode", m.handleSystemMode)
	mux.HandleFunc("/rci/", m.handleParse)

	return httptest.NewServer(mux)
}

// SetupMockRouterForTest is a convenience function for test suites.
// It creates a mock server and configures the global API client to use it.
func SetupMockRouterForTest(opts ...MockRouterOption) *httptest.Server {
	server := NewMockRouterServer(opts...)
	SetupTestConfig(server.URL)
	return server
}

// encodeJSON safely encodes data as JSON and writes it to the response.
func (m *MockRouter) encodeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode JSON: %v", err), http.StatusInternalServerError)
	}
}

// maskToCIDRString converts a dotted subnet mask to a CIDR prefix length string.
// e.g. "255.255.255.0" → "24", "255.255.0.0" → "16"
func maskToCIDRString(mask string) string {
	parts := strings.Split(mask, ".")
	if len(parts) != 4 {
		return "32"
	}
	bits := 0
	for _, p := range parts {
		var b int
		fmt.Sscanf(p, "%d", &b) //nolint:errcheck // best-effort parse
		for b > 0 {
			bits += b & 1
			b >>= 1
		}
	}
	return fmt.Sprintf("%d", bits)
}
