---
inclusion: fileMatch
fileMatchPattern: ['**/*_test.go', '**/mock_*.go']
---

# Mock Testing Guidelines

## Critical Rule: Single Unified Mock Router

This project uses ONE centralized mock router at `pkg/gokeenrestapi/mock_router.go` that simulates the Keenetic REST API for all tests.

**NEVER create separate mock implementations.** Always extend the unified mock router.

## Why This Matters

- Consistency: All tests use the same mock behavior
- Maintainability: API changes require updates in one place
- Realism: Mock closely mirrors actual Keenetic router responses
- Simplicity: No need to maintain multiple mock implementations

## Standard Setup Pattern

### Basic Setup (Default State)
```go
func TestMyFunction(t *testing.T) {
    cleanup := gokeenrestapi.SetupMockRouterForTest()
    defer cleanup()
    
    // Test code here - mock is ready with default state
}
```

### Test Suite Setup
```go
type MyTestSuite struct {
    suite.Suite
    server *httptest.Server
}

func (s *MyTestSuite) SetupSuite() {
    s.server = gokeenrestapi.SetupMockRouterForTest()
}

func (s *MyTestSuite) TearDownSuite() {
    s.server.Close()
}
```

### Custom Initial State
```go
func (s *MyTestSuite) SetupSuite() {
    s.server = gokeenrestapi.SetupMockRouterForTest(
        gokeenrestapi.WithInterfaces(customInterfaces),
        gokeenrestapi.WithRoutes(customRoutes),
        gokeenrestapi.WithDNSRecords(customDNS),
    )
}
```

## Adding New Mock Endpoints

When you need to test a new API endpoint, follow this checklist:

| Step | Action | Required |
|------|--------|----------|
| 1 | Add handler function to `mock_router.go` (e.g., `handleDeleteDnsRecord`) | ✅ Yes |
| 2 | Register endpoint in `NewMockRouterServer()` mux | ✅ Yes |
| 3 | Add endpoint to smoke test in `mock_router_test.go` | ✅ Yes |
| 4 | Write API client tests in appropriate `*_test.go` file | ✅ Yes |
| 5 | Write unit tests for mock handler behavior | ❌ No - avoid duplication |

### Handler Implementation Pattern
```go
func (m *MockRouterServer) handleMyEndpoint(w http.ResponseWriter, r *http.Request) {
    // 1. Validate HTTP method
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // 2. Parse request body
    var req MyRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // 3. Update mock state
    m.mu.Lock()
    defer m.mu.Unlock()
    // ... modify m.state ...
    
    // 4. Return response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(MyResponse{Success: true})
}
```

### Endpoint Registration Pattern
```go
func NewMockRouterServer(opts ...MockOption) *MockRouterServer {
    // ... initialization ...
    
    mux.HandleFunc("/api/my-endpoint", server.handleMyEndpoint)
    
    return server
}
```

## Test Coverage Strategy

### DO Test in Mock Infrastructure Tests (`mock_router_test.go`)

Only test mock-specific functionality not covered by API client tests:

- State management methods: `GetState()`, `ResetState()`, `SetState()`
- Configuration options: `WithInterfaces()`, `WithRoutes()`, `WithDNSRecords()`
- Mock server lifecycle: startup, shutdown, cleanup
- Edge cases unique to mock: malformed requests, invalid methods, 404s

Keep `mock_router_test.go` minimal (~5-10 tests maximum).

### DO Test in API Client Tests (`*_test.go`)

Test actual API functionality through the client interface:

- Authentication flow (`api_test.go`)
- Interface queries (`interface_test.go`)
- Route operations (`ip_test.go`)
- DNS-routing operations (`dns_routing_test.go`)
- AWG configuration (`awg_test.go`)
- Error handling and validation

### DO NOT Test in Mock Tests

Avoid duplicating coverage:

- ❌ Authentication flow → already covered by `api_test.go`
- ❌ Interface existence checks → already covered by API tests
- ❌ Route add/delete operations → already covered by `ip_test.go`
- ❌ DNS record operations → already covered by `dns_routing_test.go`
- ❌ Business logic validation → belongs in API client tests

## Mock State Management

The mock router maintains stateful behavior to simulate a real router:

```go
type MockRouterState struct {
    Interfaces  []Interface
    Routes      []Route
    DNSRecords  []DNSRecord
    DNSRouting  []DNSRoutingRule
    AWGConfig   *AWGConfig
}
```

### Accessing State in Tests
```go
// Get current state
state := server.GetState()
assert.Len(t, state.Routes, expectedCount)

// Reset to default state
server.ResetState()

// Set custom state
server.SetState(customState)
```

## Common Patterns

### Testing Add Operations
```go
func TestAddRoute(t *testing.T) {
    cleanup := gokeenrestapi.SetupMockRouterForTest()
    defer cleanup()
    
    // Perform operation
    err := gokeenrestapi.Ip.AddRoute("192.168.1.0/24", "Wireguard0")
    assert.NoError(t, err)
    
    // Verify state changed (if needed for test)
    // Usually not necessary - trust the API client
}
```

### Testing Delete Operations
```go
func TestDeleteRoute(t *testing.T) {
    // Setup with existing route
    cleanup := gokeenrestapi.SetupMockRouterForTest(
        gokeenrestapi.WithRoutes([]Route{{IP: "192.168.1.0/24"}}),
    )
    defer cleanup()
    
    err := gokeenrestapi.Ip.DeleteRoute("192.168.1.0/24", "Wireguard0")
    assert.NoError(t, err)
}
```

### Testing Error Conditions
```go
func TestInvalidInterface(t *testing.T) {
    cleanup := gokeenrestapi.SetupMockRouterForTest()
    defer cleanup()
    
    err := gokeenrestapi.Ip.AddRoute("192.168.1.0/24", "NonExistent")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "interface not found")
}
```

## File Organization

| File | Purpose | Test Count |
|------|---------|------------|
| `mock_router.go` | Unified mock implementation with all handlers | N/A |
| `mock_router_test.go` | Mock infrastructure tests only | ~5-10 |
| `api_test.go` | Authentication and general API tests | ~10-20 |
| `ip_test.go` | IP route operation tests | ~15-30 |
| `dns_routing_test.go` | DNS-routing operation tests | ~15-30 |
| `awg_test.go` | AWG configuration tests | ~10-20 |
| `interface_test.go` | Interface query tests | ~5-10 |

## Workflow: Adding New Endpoint Support

Example: Adding DNS record deletion support

1. **Add handler to `mock_router.go`**:
```go
func (m *MockRouterServer) handleDeleteDnsRecord(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var req DeleteDNSRecordRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Remove from state
    filtered := []DNSRecord{}
    for _, record := range m.state.DNSRecords {
        if record.Domain != req.Domain {
            filtered = append(filtered, record)
        }
    }
    m.state.DNSRecords = filtered
    
    w.WriteHeader(http.StatusOK)
}
```

2. **Register in `NewMockRouterServer()`**:
```go
mux.HandleFunc("/api/dns/record", server.handleDeleteDnsRecord)
```

3. **Add to smoke test in `mock_router_test.go`**:
```go
func TestMockRouterEndpoints(t *testing.T) {
    // ... existing tests ...
    
    // Test DNS record deletion endpoint exists
    resp, err := http.Delete(server.URL + "/api/dns/record")
    assert.NoError(t, err)
    assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
}
```

4. **Write API client test in `dns_routing_test.go`**:
```go
func TestDeleteDnsRecord(t *testing.T) {
    cleanup := SetupMockRouterForTest(
        WithDNSRecords([]DNSRecord{{Domain: "test.local", IP: "192.168.1.1"}}),
    )
    defer cleanup()
    
    err := DnsRouting.DeleteRecord("test.local")
    assert.NoError(t, err)
}
```

5. **Done** - No need for mock unit test since API client test covers the behavior

## Anti-Patterns to Avoid

❌ **Creating separate mock implementations**
```go
// DON'T DO THIS
type MyCustomMock struct {}
func (m *MyCustomMock) GetRoutes() []Route { ... }
```

❌ **Testing mock behavior in mock tests when API tests cover it**
```go
// DON'T DO THIS in mock_router_test.go
func TestMockHandlesRouteAddition(t *testing.T) {
    // This belongs in ip_test.go, not mock_router_test.go
}
```

❌ **Not using the cleanup pattern**
```go
// DON'T DO THIS
func TestSomething(t *testing.T) {
    SetupMockRouterForTest() // Missing defer cleanup()
    // Test code
}
```

❌ **Manually managing mock server lifecycle**
```go
// DON'T DO THIS
server := httptest.NewServer(...)
defer server.Close()
// Use SetupMockRouterForTest() instead
```

## Integration with Property-Based Tests

The mock router works seamlessly with property-based tests using `rapid`:

```go
func TestProperty_RouteOperationsIdempotent(t *testing.T) {
    cleanup := SetupMockRouterForTest()
    defer cleanup()
    
    rapid.Check(t, func(t *rapid.T) {
        route := rapid.String().Draw(t, "route")
        iface := rapid.String().Draw(t, "interface")
        
        // Add twice - should be idempotent
        err1 := gokeenrestapi.Ip.AddRoute(route, iface)
        err2 := gokeenrestapi.Ip.AddRoute(route, iface)
        
        // Both should succeed or both should fail
        assert.Equal(t, err1 == nil, err2 == nil)
    })
}
```

## Summary

- Use the unified mock at `pkg/gokeenrestapi/mock_router.go` for all tests
- Extend the mock by adding handlers and registering endpoints
- Test API behavior through client tests, not mock unit tests
- Keep mock infrastructure tests minimal and focused
- Follow the cleanup pattern with `defer cleanup()`
- Use state management methods when you need custom initial state
