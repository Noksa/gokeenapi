# Design Document

## Overview

The unified Keenetic mock system provides a comprehensive, stateful simulation of the Keenetic router REST API for testing purposes. The design centers around a `MockRouter` struct that maintains internal state and exposes HTTP endpoints matching the real Keenetic router API. The mock supports all current API operations including authentication, interface management, routing, DNS, and device tracking.

The key innovation is consolidating multiple scattered mock implementations into a single, maintainable system with clear state management, making it easy to test complex sequences of operations and add new functionality.

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Test Code                             │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ HTTP Requests
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   httptest.Server                            │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              HTTP Handler (ServeMux)                   │  │
│  │  ┌─────────────────────────────────────────────────┐  │  │
│  │  │           MockRouter                             │  │  │
│  │  │  ┌──────────────────────────────────────────┐   │  │  │
│  │  │  │        Router State                       │   │  │  │
│  │  │  │  - Interfaces                             │   │  │  │
│  │  │  │  - Routes                                 │   │  │  │
│  │  │  │  - DNS Records                            │   │  │  │
│  │  │  │  - Hotspot Devices                        │   │  │  │
│  │  │  │  - Auth State                             │   │  │  │
│  │  │  └──────────────────────────────────────────┘   │  │  │
│  │  │                                                   │  │  │
│  │  │  ┌──────────────────────────────────────────┐   │  │  │
│  │  │  │     Endpoint Handlers                     │   │  │  │
│  │  │  │  - handleAuth()                           │   │  │  │
│  │  │  │  - handleVersion()                        │   │  │  │
│  │  │  │  - handleInterfaces()                     │   │  │  │
│  │  │  │  - handleRoutes()                         │   │  │  │
│  │  │  │  - handleDNS()                            │   │  │  │
│  │  │  │  - handleHotspot()                        │   │  │  │
│  │  │  │  - handleParse()                          │   │  │  │
│  │  │  └──────────────────────────────────────────┘   │  │  │
│  │  └─────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Component Interaction Flow

```
Test → NewMockRouter(config) → MockRouter instance
                                     │
                                     ├─ Initialize default state
                                     ├─ Apply custom config
                                     └─ Register HTTP handlers
                                            │
Test → HTTP Request → Handler → Read/Modify State → JSON Response
```

## Components and Interfaces

### MockRouter

The central component that maintains router state and handles all API endpoints.

```go
type MockRouter struct {
    mu              sync.RWMutex
    interfaces      map[string]*MockInterface
    scInterfaces    map[string]*MockScInterface
    routes          []MockRoute
    dnsRecords      map[string]string
    hotspotDevices  []MockHost
    authRealm       string
    authChallenge   string
    sessionCookie   string
    systemMode      MockSystemMode
}
```

### MockInterface

Represents a network interface on the router.

```go
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
```

### MockScInterface

Represents the SC (system configuration) view of an interface, including AWG parameters.

```go
type MockScInterface struct {
    Description string
    IP          MockIP
    Wireguard   MockWireguard
}

type MockWireguard struct {
    Asc  MockAsc
    Peer []MockPeer
}

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
```

### MockRoute

Represents a static route.

```go
type MockRoute struct {
    Network   string
    Host      string
    Mask      string
    Interface string
    Auto      bool
}
```

### MockHost

Represents a device connected to the router.

```go
type MockHost struct {
    Name       string
    Mac        string
    IP         string
    Hostname   string
    Registered bool
    Link       string
    Via        string
}
```

### MockRouterOption

Functional option type for configuring the mock router.

```go
type MockRouterOption func(*MockRouter)
```

### Public API

```go
// NewMockRouter creates a new mock router with default state and optional configuration
func NewMockRouter(opts ...MockRouterOption) *MockRouter

// NewMockRouterServer creates an httptest.Server with the mock router
func NewMockRouterServer(opts ...MockRouterOption) *httptest.Server

// SetupMockRouterForTest is a convenience function for test suites with default config
func SetupMockRouterForTest(opts ...MockRouterOption) *httptest.Server

// Functional options for configuring the mock router
func WithInterfaces(interfaces []MockInterface) MockRouterOption
func WithRoutes(routes []MockRoute) MockRouterOption
func WithDNSRecords(records map[string]string) MockRouterOption
func WithHotspotDevices(devices []MockHost) MockRouterOption
func WithSystemMode(mode MockSystemMode) MockRouterOption

// GetState returns a snapshot of the current mock state (for test assertions)
func (m *MockRouter) GetState() MockRouterState

// ResetState resets the mock to its initial state
func (m *MockRouter) ResetState()
```

## Data Models

The mock uses internal data structures that map to the existing `gokeenrestapimodels` types. The internal types (Mock*) provide a simplified representation for state management, while endpoint handlers convert them to the appropriate API model types for responses.

### State Conversion

```
Internal State (Mock*)  →  API Response (gokeenrestapimodels.*)
MockInterface          →  RciShowInterface
MockScInterface        →  RciShowScInterface
MockRoute              →  RciIpRoute
MockHost               →  Host (in RciShowIpHotspot)
```

### Default State

When no custom configuration is provided, the mock initializes with:

- **Interfaces**: 
  - Wireguard0 (WireGuard, 10.0.0.1/24, connected/up)
  - ISP (PPPoE, connected/up)
- **Routes**: 
  - 192.168.1.0/24 via Wireguard0
- **DNS Records**: 
  - example.com → 1.2.3.4
  - test.local → 192.168.1.50
- **Hotspot Devices**: 
  - test-device-1 (aa:bb:cc:dd:ee:ff, 192.168.1.100)
  - test-device-2 (11:22:33:44:55:66, 192.168.1.101)
- **System Mode**: router/router (active/selected)


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: All endpoints are accessible

*For any* newly created mock router instance, making HTTP requests to all required endpoints (auth, version, interfaces, routes, DNS, hotspot, parse, running-config, system-mode) should return valid responses (not HTTP 404)
**Validates: Requirements 1.1**

### Property 2: Default state initialization

*For any* mock router created without custom configuration, the initial state should contain non-empty default values for interfaces, routes, DNS records, and hotspot devices
**Validates: Requirements 1.3, 3.5**

### Property 3: Mock instance isolation

*For any* two independently created mock router instances, modifying the state of one instance should not affect the state of the other instance
**Validates: Requirements 1.4**

### Property 4: HTTP method support

*For any* endpoint that accepts GET requests, making a GET request should return HTTP 200 (or appropriate success code), and for any endpoint that accepts POST requests, making a POST request with valid data should return HTTP 200
**Validates: Requirements 1.5**

### Property 5: Route addition round-trip

*For any* valid route, adding it via the parse endpoint then querying the routes list should include that route with matching network, mask, and interface fields
**Validates: Requirements 2.1**

### Property 6: DNS record addition round-trip

*For any* valid DNS record (domain and IP pair), adding it via the parse endpoint then querying DNS records should include that record
**Validates: Requirements 2.2, 7.2**

### Property 7: Interface modification round-trip

*For any* existing interface and valid configuration change, modifying the interface via the parse endpoint then querying that interface should reflect the modification
**Validates: Requirements 2.3**

### Property 8: Host deletion removes from hotspot

*For any* host in the hotspot list, deleting it via the parse endpoint then querying the hotspot should not include that host
**Validates: Requirements 2.4, 8.2**

### Property 9: Invalid interface returns 404

*For any* interface ID that does not exist in the mock's state, querying that interface should return HTTP 404
**Validates: Requirements 4.1**

### Property 10: Invalid auth returns 401

*For any* authentication request with invalid credentials, the response should be HTTP 401 with authentication challenge headers
**Validates: Requirements 4.2, 9.3**

### Property 11: Malformed parse command returns error

*For any* parse command that is syntactically invalid or references non-existent resources, the parse response should contain an error status
**Validates: Requirements 4.3, 12.4**

### Property 12: Wrong HTTP method returns 405

*For any* GET-only endpoint, making a POST request should return HTTP 405 Method Not Allowed
**Validates: Requirements 4.4**

### Property 13: WireGuard interface response completeness

*For any* WireGuard interface in the mock state, querying interfaces should return that interface with all required fields (id, type, description, address, connected, link, state) populated
**Validates: Requirements 5.1**

### Property 14: SC interface AWG parameters completeness

*For any* WireGuard interface with AWG configuration, querying SC interfaces should return that interface with all AWG parameters (Jc, Jmin, Jmax, S1, S2, H1, H2, H3, H4) populated
**Validates: Requirements 5.2**

### Property 15: AWG parameter update round-trip

*For any* WireGuard interface with AWG configuration and any valid AWG parameter change, updating the parameters via parse endpoint then querying SC interfaces should reflect the new parameter values
**Validates: Requirements 5.3**

### Property 16: Interface creation appears in both listings

*For any* new WireGuard interface created via parse endpoint, both the interfaces endpoint and SC interfaces endpoint should include that interface
**Validates: Requirements 5.4**

### Property 17: Interface state change propagation

*For any* interface and any valid state change (connected, link, or state field), changing the state via parse endpoint then querying that interface should show the updated state values
**Validates: Requirements 5.5**

### Property 18: Route filtering by interface

*For any* interface ID, querying routes for that interface should return only routes where the interface field matches that ID
**Validates: Requirements 6.1**

### Property 19: Route addition validates interface existence

*For any* route with an interface ID that does not exist in the mock state, attempting to add that route via parse endpoint should return an error response
**Validates: Requirements 6.2**

### Property 20: Route deletion removes from list

*For any* route in the mock state, deleting it via parse endpoint then querying routes should not include that route
**Validates: Requirements 6.3**

### Property 21: Route response field completeness

*For any* route in the mock state, querying routes should return that route with all required fields (network, host, mask, interface, auto) populated
**Validates: Requirements 6.5**

### Property 22: DNS query returns all records

*For any* set of DNS records in the mock state, querying DNS records should return all of them
**Validates: Requirements 7.1**

### Property 23: DNS deletion removes from list

*For any* DNS record in the mock state, deleting it via parse endpoint then querying DNS records should not include that record
**Validates: Requirements 7.3**

### Property 24: DNS response format correctness

*For any* DNS query, the response should be a JSON object with a "static" key containing a map of domain names to IP addresses
**Validates: Requirements 7.5**

### Property 25: Hotspot query returns all devices

*For any* set of devices in the hotspot state, querying hotspot should return all devices with all required fields (name, mac, ip, hostname) populated
**Validates: Requirements 8.1, 8.4**

### Property 26: Hotspot response format correctness

*For any* hotspot query, the response should be a JSON object matching the RciShowIpHotspot structure with a "host" array
**Validates: Requirements 8.3**

### Property 27: Parse request/response length matching

*For any* array of N parse requests submitted to the parse endpoint, the response should contain exactly N parse response objects
**Validates: Requirements 12.2**

### Property 28: Authenticated requests succeed

*For any* mock router instance, after successfully authenticating and receiving a session cookie, subsequent requests with that session cookie should succeed (not return 401)
**Validates: Requirements 9.4**


## Error Handling

### HTTP Error Responses

The mock implements proper HTTP error codes for various failure scenarios:

- **404 Not Found**: When querying non-existent interfaces or resources
- **401 Unauthorized**: When authentication fails or is missing
- **405 Method Not Allowed**: When using wrong HTTP method for an endpoint
- **400 Bad Request**: When submitting malformed JSON or invalid request data
- **500 Internal Server Error**: For unexpected errors during request processing

### Parse Command Errors

Parse commands that fail return a ParseResponse with error status:

```go
ParseResponse{
    Parse: Parse{
        Status: []Status{
            {
                Status:  "error",
                Code:    "1",
                Message: "descriptive error message",
            },
        },
    },
}
```

### Validation Errors

The mock validates:
- Interface existence before adding routes
- Valid IP address formats for DNS records
- Valid MAC address formats for hotspot devices
- Valid network/mask combinations for routes
- Valid AWG parameter ranges

### Thread Safety

All state modifications are protected by a `sync.RWMutex` to ensure thread-safe access. Read operations use `RLock()` and write operations use `Lock()`.

## Testing Strategy

### Unit Testing

Unit tests will verify:
- Individual endpoint handlers return correct response formats
- State initialization with default and custom configurations
- Error responses for invalid inputs
- JSON encoding/decoding of all data structures
- Helper functions for state manipulation
- Thread safety of concurrent access

Example unit tests:
- `TestNewMockRouter_DefaultState`: Verify default initialization
- `TestNewMockRouter_CustomConfig`: Verify custom config is applied
- `TestHandleAuth_Success`: Verify successful authentication flow
- `TestHandleAuth_Failure`: Verify failed authentication returns 401
- `TestHandleInterfaces_ReturnsAllInterfaces`: Verify interface listing
- `TestHandleInterface_NotFound`: Verify 404 for non-existent interface
- `TestHandleParse_AddRoute`: Verify route addition updates state
- `TestHandleParse_InvalidCommand`: Verify error response for invalid command

### Property-Based Testing

Property-based tests will use the `rapid` framework to verify universal properties across many randomly generated inputs. Each property test will run a minimum of 100 iterations.

Property tests will verify:
- **Property 1-28**: All correctness properties defined above
- State consistency after sequences of operations
- Idempotent operations (e.g., adding same route twice)
- Commutative operations (e.g., order of adding routes doesn't matter)
- Round-trip properties (add then query, modify then query, delete then query)

Example property test structure:

```go
func TestProperty5_RouteAdditionRoundTrip(t *testing.T) {
    // Feature: unified-keenetic-mock, Property 5: Route addition round-trip
    rapid.Check(t, func(t *rapid.T) {
        // Generate random route
        route := generateRandomRoute(t)
        
        // Create mock and add route
        mock := NewMockRouter(nil)
        server := httptest.NewServer(mock.Handler())
        defer server.Close()
        
        // Add route via parse endpoint
        addRouteCommand := fmt.Sprintf("ip route %s/%s %s", 
            route.Network, route.Mask, route.Interface)
        // ... execute parse command ...
        
        // Query routes
        routes := queryRoutes(server.URL, route.Interface)
        
        // Verify route is present
        found := false
        for _, r := range routes {
            if r.Network == route.Network && 
               r.Mask == route.Mask && 
               r.Interface == route.Interface {
                found = true
                break
            }
        }
        
        if !found {
            t.Fatalf("route not found after addition: %+v", route)
        }
    })
}
```

### Integration Testing

Integration tests will verify:
- Compatibility with existing test infrastructure (testify suites)
- Compatibility with existing `SetupTestConfig` function
- Proper cleanup in `TearDownSuite`
- Migration from old mocks to unified mock

### Test Organization

Tests will be organized in:
- `pkg/gokeenrestapi/mock_router.go`: Mock implementation
- `pkg/gokeenrestapi/mock_router_test.go`: Unit tests
- `pkg/gokeenrestapi/mock_router_property_test.go`: Property-based tests
- `pkg/gokeenrestapi/mock_router_integration_test.go`: Integration tests with existing code

## Implementation Details

### Endpoint Handlers

Each endpoint handler follows this pattern:

```go
func (m *MockRouter) handleEndpoint(w http.ResponseWriter, r *http.Request) {
    // 1. Validate HTTP method
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // 2. Acquire appropriate lock
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    // 3. Process request and build response
    response := m.buildResponse()
    
    // 4. Encode and send response
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(response); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}
```

### Parse Command Processing

The parse endpoint uses a command parser that:
1. Parses the command string to identify operation type
2. Extracts parameters from the command
3. Validates parameters against current state
4. Executes state modification
5. Returns appropriate success/error response

Command patterns supported:
- `interface <id> up|down`: Change interface state
- `ip route <network>/<mask> <gateway> <interface>`: Add route
- `no ip route <network>/<mask> <interface>`: Delete route
- `ip name-server <domain> <ip>`: Add DNS record
- `no ip name-server <domain>`: Delete DNS record
- `interface <id> wireguard asc jc <value> ...`: Update AWG parameters

### State Management

State modifications follow these principles:
- All modifications acquire write lock
- All queries acquire read lock
- State changes are atomic (all-or-nothing)
- Invalid operations leave state unchanged
- State is deep-copied when returned via GetState()

### Migration Support

To support migration from existing mocks:

1. **Backward-compatible constructor**: `SetupMockRouterForTest()` matches signature of existing `SetupMockServer()`
2. **Drop-in replacement**: Returns `*httptest.Server` like existing mocks
3. **State inspection**: `GetState()` method for test assertions
4. **Reset capability**: `ResetState()` for test isolation

Migration steps for each test file:
1. Replace `setupMockServerForX()` with `SetupMockRouterForTest(config)`
2. Remove custom mock server code
3. Update test assertions to use unified mock's state
4. Verify all tests pass

## Dependencies

- `net/http`: HTTP server and client
- `net/http/httptest`: Test server infrastructure
- `encoding/json`: JSON encoding/decoding
- `sync`: Mutex for thread safety
- `github.com/stretchr/testify/suite`: Test suite framework
- `pgregory.net/rapid`: Property-based testing framework
- `github.com/noksa/gokeenapi/pkg/gokeenrestapimodels`: API data models
- `github.com/noksa/gokeenapi/pkg/config`: Configuration types

## Migration Guide

### Identifying Existing Mocks

Current mock implementations to be replaced:
1. `pkg/gokeenrestapi/testing.go`: `SetupMockServer()` - General mock
2. `pkg/gokeenrestapi/awg_test.go`: `setupMockServerForAWG()` - AWG-specific mock
3. `pkg/gokeenrestapi/ip_test.go`: `setupMockServerForIP()` - IP operations mock

### Migration Examples

#### Before: Using separate mock

```go
func (s *AwgTestSuite) SetupSuite() {
    s.server = s.setupMockServerForAWG()
    SetupTestConfig(s.server.URL)
}

func (s *AwgTestSuite) setupMockServerForAWG() *httptest.Server {
    mux := http.NewServeMux()
    // ... 50+ lines of endpoint setup ...
    return httptest.NewServer(mux)
}
```

#### After: Using unified mock with default state

```go
func (s *AwgTestSuite) SetupSuite() {
    // Uses default state - no options needed
    s.server = SetupMockRouterForTest()
    SetupTestConfig(s.server.URL)
}
```

#### After: Using unified mock with custom state

```go
func (s *AwgTestSuite) SetupSuite() {
    s.server = SetupMockRouterForTest(
        WithInterfaces([]MockInterface{
            {
                ID:        "Wireguard0",
                Type:      InterfaceTypeWireguard,
                Connected: StateConnected,
                Address:   "10.0.0.1/24",
            },
        }),
        WithRoutes([]MockRoute{
            {Network: "192.168.1.0", Mask: "255.255.255.0", Interface: "Wireguard0"},
        }),
    )
    SetupTestConfig(s.server.URL)
}
```

### Migration Checklist

For each test file:
- [ ] Identify custom mock server setup
- [ ] Determine required initial state
- [ ] Create `MockRouterConfig` with initial state
- [ ] Replace mock setup with `SetupMockRouterForTest(config)`
- [ ] Remove custom mock server code
- [ ] Run tests and verify they pass
- [ ] Check for any test-specific mock behavior that needs to be added to unified mock

### Backward Compatibility

The unified mock maintains backward compatibility by:
- Providing `SetupMockServer()` function that creates a mock with default state (alias for `SetupMockRouterForTest()`)
- Returning `*httptest.Server` from all setup functions
- Working with existing `SetupTestConfig()` function
- Supporting existing cleanup patterns in `TearDownSuite`
- Using variadic options pattern - no options means default behavior

This allows gradual migration - tests can be migrated one file at a time without breaking other tests.

### Usage Examples

```go
// Simple case - use defaults
server := SetupMockRouterForTest()

// Custom interfaces only
server := SetupMockRouterForTest(
    WithInterfaces([]MockInterface{
        {ID: "Wireguard0", Type: "Wireguard", Address: "10.0.0.1/24"},
    }),
)

// Multiple options
server := SetupMockRouterForTest(
    WithInterfaces(customInterfaces),
    WithRoutes(customRoutes),
    WithDNSRecords(map[string]string{"example.com": "1.2.3.4"}),
)

// Backward compatible - existing code still works
server := SetupMockServer() // Equivalent to SetupMockRouterForTest()
```
