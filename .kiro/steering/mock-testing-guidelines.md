# Mock Testing Guidelines

## Unified Mock Router

This project has a **unified mock router** (`pkg/gokeenrestapi/mock_router.go`) that simulates the complete Keenetic router REST API for testing purposes.

### Using the Mock in Tests

**ALWAYS use the unified mock for any test that needs router API interaction:**

```go
func (s *YourTestSuite) SetupSuite() {
    s.server = SetupMockRouterForTest()
    SetupTestConfig(s.server.URL)
}
```

For custom initial state:
```go
s.server = SetupMockRouterForTest(
    WithInterfaces(customInterfaces),
    WithRoutes(customRoutes),
    WithDNSRecords(customDNS),
)
```

### When Mock Functionality is Missing

If you need to test functionality that the mock doesn't support yet:

1. ✅ **DO** add the missing endpoint/behavior to the unified mock
2. ✅ **DO** write your API client test that uses the enhanced mock
3. ❌ **DON'T** create a separate mock just for your test
4. ❌ **DON'T** write unit tests for the mock endpoint itself

**The mock should grow with the application's needs, but remain a single unified implementation.**

## Philosophy

Mock servers exist to support testing of real application code. The mock itself is test infrastructure, not production code. Therefore, testing the mock should be minimal and focused.

## Testing Strategy for Mocks

### What TO Test (Mock-Specific Functionality)

Write unit tests ONLY for mock-specific features that aren't exercised by real API client tests:

1. **State management functions** - `GetState()`, `ResetState()`, state isolation
2. **Configuration options** - Functional options pattern (`WithInterfaces`, `WithRoutes`, etc.)
3. **Mock infrastructure** - Endpoint registration, server setup
4. **Edge cases not covered by API tests** - 404 responses, 405 method not allowed, malformed requests

### What NOT TO Test (Endpoint Behavior)

DO NOT write unit tests for mock endpoint behavior if it's already tested by real API client tests:

- ❌ Authentication flow (if `TestAuth()` exists in API tests)
- ❌ Interface queries (if `TestGetInterfaces()` exists in API tests)
- ❌ Route operations (if route tests exist in API tests)
- ❌ DNS operations (if DNS tests exist in API tests)
- ❌ Any endpoint that has corresponding API client tests

**Rationale:** The API client tests already prove the mock works correctly in the context it will actually be used. Duplicating these tests at the mock level is redundant and adds maintenance burden.

## Example: Authentication Testing

### ❌ Bad (Redundant)
```go
// In mock_router_test.go
func TestHandleAuth_GET(t *testing.T) {
    server := NewMockRouterServer()
    resp, _ := http.Get(server.URL + "/auth")
    assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
    assert.NotEmpty(t, resp.Header.Get("x-ndm-realm"))
}
```

### ✅ Good (Already covered by API tests)
```go
// In api_test.go - this already tests the mock!
func (s *ApiTestSuite) TestAuth() {
    err := Common.Auth()  // Uses the mock server
    s.NoError(err)
}
```

## Minimal Mock Test Suite

A well-designed mock test suite should have:

1. **One smoke test** - Verifies all endpoints are registered
2. **State management tests** - Tests `GetState()`, `ResetState()`
3. **Configuration tests** - Tests functional options work
4. **Edge case tests** - Tests 404s, 405s, error conditions not covered by API tests

**Total:** ~5-10 tests maximum for a comprehensive mock server

## When Adding New Mock Endpoints

When implementing a new mock endpoint (because you need it for a test):

1. ✅ **DO** add the endpoint to `pkg/gokeenrestapi/mock_router.go`
2. ✅ **DO** register it in `NewMockRouterServer()`
3. ✅ **DO** add the endpoint to the smoke test in `mock_router_test.go` (endpoint registration check)
4. ✅ **DO** write API client tests that use the mock
5. ❌ **DON'T** write separate unit tests for the endpoint behavior
6. ✅ **DO** add edge case tests to `mock_router_test.go` if the API client doesn't cover them (e.g., 404, invalid input)

**Example workflow:**
- Need to test DNS record deletion → Add `handleDeleteDnsRecord` to mock → Write `TestDeleteDnsRecord` in API tests → Done!

## Benefits of This Approach

- **Less code to maintain** - Fewer tests means less maintenance burden
- **Faster test execution** - No redundant test runs
- **Better integration testing** - API tests prove the mock works in real usage
- **Clear separation** - Mock tests focus on mock infrastructure, API tests focus on API behavior
- **Easier refactoring** - Changes to mock implementation don't break redundant tests

## Summary

- **One unified mock** - `pkg/gokeenrestapi/mock_router.go` for all router API testing
- **Use it everywhere** - All tests requiring router API should use `SetupMockRouterForTest()`
- **Extend when needed** - Add missing functionality to the unified mock, don't create separate mocks
- **Test through API clients** - Write tests that use the mock via real API client code
- **Minimal mock tests** - Only test mock infrastructure, not endpoint behavior

## Reference Implementation

See `pkg/gokeenrestapi/mock_router_test.go` for an example of minimal mock testing:
- Endpoint registration smoke test
- State management tests
- Configuration option tests
- Edge case tests (404 for non-existent resources)

All endpoint behavior is tested via `pkg/gokeenrestapi/api_test.go` which uses the mock server.
