---
inclusion: fileMatch
fileMatchPattern: ['**/*_test.go', '**/mock_*.go']
---

# Mock Testing Guidelines

## Core Rule: One Unified Mock

Location: `pkg/gokeenrestapi/mock_router.go`

**NEVER create separate mocks.** Always extend the unified mock router.

## Setup Pattern

```go
func (s *YourTestSuite) SetupSuite() {
    s.server = SetupMockRouterForTest()
    SetupTestConfig(s.server.URL)
}

// With custom state:
s.server = SetupMockRouterForTest(
    WithInterfaces(customInterfaces),
    WithRoutes(customRoutes),
    WithDNSRecords(customDNS),
)
```

## Adding New Mock Functionality

| Action | Required |
|--------|----------|
| Add endpoint to `mock_router.go` | ✅ Yes |
| Register in `NewMockRouterServer()` | ✅ Yes |
| Add to smoke test (endpoint registration) | ✅ Yes |
| Write API client tests using the mock | ✅ Yes |
| Write unit tests for endpoint behavior | ❌ No |

## What TO Test in Mock Tests

Only test mock infrastructure not covered by API client tests:

- State management: `GetState()`, `ResetState()`
- Configuration options: `WithInterfaces`, `WithRoutes`, etc.
- Edge cases: 404s, 405s, malformed requests

## What NOT TO Test

Skip if API client tests already cover it:

- Authentication flow → covered by `api_test.go`
- Interface queries → covered by API tests
- Route/DNS operations → covered by API tests

## Quick Reference

| File | Purpose |
|------|---------|
| `mock_router.go` | Unified mock implementation |
| `mock_router_test.go` | Infrastructure tests only (~5-10 tests max) |
| `api_test.go` | Endpoint behavior tests via API client |

## Workflow Example

Need DNS record deletion test:
1. Add `handleDeleteDnsRecord` to `mock_router.go`
2. Register endpoint in `NewMockRouterServer()`
3. Write `TestDeleteDnsRecord` in `api_test.go`
4. Done — no mock unit test needed
