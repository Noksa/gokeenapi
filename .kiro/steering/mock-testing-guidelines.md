---
inclusion: fileMatch
fileMatchPattern: ['**/*_test.go', '**/mock_*.go']
---

# Mock Testing Guidelines

Ginkgo/Gomega layout, suites, and Makefile targets: `structure.md` and `tech.md`.
This file covers the unified Keenetic mock only.

## Critical Rule: Single Unified Mock Router

All tests use one mock at `pkg/gokeenrestapi/mock_router.go` (`MockRouter`).

**Never create a separate mock implementation.** Extend this mock.

## File Split

| File | Role |
|------|------|
| `mock_router.go` | `MockRouter` state, `MockRouterOption`s, `NewMockRouter`, `NewMockRouterServer`, `SetupMockRouterForTest` |
| `mock_router_handlers_core.go` | Auth, version, parse dispatch, system mode, running-config |
| `mock_router_handlers_dns.go` | DNS records, object-groups, dns-proxy routes, components |
| `mock_router_handlers_interface.go` | Interfaces, SC interfaces, hotspot, known hosts |
| `mock_router_handlers_routes.go` | IP routes, AWG/WireGuard parse |
| `test_helpers.go` | `SetupTestConfig`, `CleanupTestConfig` |

## Standard Setup

### `cmd` package

Use `setupMockRouter` / `cleanupMockRouter` from `cmd/helpers_test.go` (they already call `Auth()` and `CleanupTestConfig()`).

### Other packages

```go
var _ = Describe("MyFeature", func() {
    var server *httptest.Server

    BeforeEach(func() {
        server = gokeenrestapi.SetupMockRouterForTest()
    })

    AfterEach(func() {
        if server != nil {
            server.Close()
        }
        gokeenrestapi.CleanupTestConfig()
    })
})
```

### Custom initial state

Options are `MockRouterOption` funcs on `*MockRouter`:

```go
server = gokeenrestapi.SetupMockRouterForTest(
    gokeenrestapi.WithInterfaces([]gokeenrestapi.MockInterface{{ID: "Wireguard0", Type: gokeenrestapi.InterfaceTypeWireguard}}),
    gokeenrestapi.WithRoutes([]gokeenrestapi.MockRoute{{Host: "192.168.1.0", Mask: "255.255.255.0", Interface: "Wireguard0"}}),
    gokeenrestapi.WithDNSRecords(map[string]string{"example.com": "1.2.3.4"}),
    gokeenrestapi.WithVersion("5.0.1"),
)
```

Other options: `WithScInterfaces`, `WithHotspotDevices`, `WithSystemMode`, `WithStaticRunningConfig`, `WithDnsRoutingGroups`, `WithRciBody`, `WithComponents`.

## Adding a New Mock Endpoint

| Step | Action |
|------|--------|
| 1 | Add the handler on `*MockRouter` in the matching `mock_router_handlers_*.go` file |
| 2 | Register the path in `NewMockRouterServer()` in `mock_router.go` |
| 3 | Smoke-cover the path from an API client test (or a small mock infrastructure test if the endpoint has no client yet) |
| 4 | Do **not** add a parallel mock-only unit test that duplicates the client test |

Handlers take `(w http.ResponseWriter, r *http.Request)`, lock `m.mu` when mutating state, and encode JSON via `m.encodeJSON`.

## What To Test Where

**Mock infrastructure tests** (`mock_router_test.go` and `mock_router_*_test.go`): state helpers, options, lifecycle, mock-only edge cases (bad method, 404). Keep these focused.

**API client tests**: real behavior through singletons (`api_test.go`, `ip_test.go`, `dns_routing_*.go` tests, `awg_test.go`, `interface_wait_test.go`, `checks_test.go`).

Do not re-test route add/delete inside mock tests if `ip_test.go` already covers it.

## State

```go
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
```

`GetState()` / `ResetState()` operate on this snapshot. Types are `MockRoute`, `MockInterface`, … — not unprefixed `Route`.

## Example: command path through the real API

```go
It("should add routes from a bat file", func() {
    path := writeTempFile(tmpDir, "routes.bat", "route add 10.0.0.0 MASK 255.255.255.0 0.0.0.0\n")
    Expect(gokeenrestapi.Ip.AddRoutesFromBatFile(path, "Wireguard0")).To(Succeed())
})
```

There is no `Ip.AddRoute` helper.

## Property tests

The mock works with `rapid.Check(GinkgoT(), ...)` inside Ginkgo `It` blocks. Reset or isolate state inside the property if the generator mutates the router.

## Anti-Patterns

- Separate mock types (`type MyCustomMock struct {}`)
- Testing business logic only in `mock_router_test.go`
- Skipping `AfterEach` cleanup
- Hand-rolling `httptest.NewServer` instead of `SetupMockRouterForTest`
- Calling `Auth()` in `cmd` tests after `setupMockRouter` (already authenticated)
