---
inclusion: always
---

# Tech Stack & Development Rules

Product behavior: `product.md`. Layout and adding a command: `structure.md`. Mocks: `mock-testing-guidelines.md`. CI YAML: `github-actions-workflow.md`.

## Language & Runtime

- Go 1.26+ (`go` directive in `go.mod`; Dockerfile uses `golang:1.26-alpine`)
- Never modify `go.mod` manually — use `go get` or `go mod tidy`

## Key Dependencies

| Package | Import Path | Purpose |
|---------|-------------|---------|
| cobra | `github.com/spf13/cobra` | CLI — always `RunE`, never `Run` |
| resty | `github.com/go-resty/resty/v2` | HTTP — never use `net/http` directly in production code |
| yaml.v3 | `gopkg.in/yaml.v3` | YAML parsing via `yaml.Unmarshal()` |
| ginkgo/v2 | `github.com/onsi/ginkgo/v2` | Test framework — all tests use Ginkgo `Describe`/`It` |
| gomega | `github.com/onsi/gomega` | Matchers — `Expect(x).To(Equal(y))`, never testify |
| rapid | `pgregory.net/rapid` | Property-based tests in `*_property_test.go` |
| multierr | `go.uber.org/multierr` | Error aggregation: `multierr.Append(errs, err)` |
| go-version | `github.com/hashicorp/go-version` | Firmware version comparison |
| color | `github.com/fatih/color` | Colored CLI output |
| spinner | `github.com/briandowns/spinner` | Progress indicators (`internal/gokeenspinner`) |
| go-cache | `github.com/patrickmn/go-cache` | In-memory cache (`internal/gokeencache`) |
| ini | `gopkg.in/ini.v1` | WireGuard `.conf` parsing |

Request/response structs live in `pkg/gokeenrestapimodels/` — always use those types with API singletons.

## Build & Test Commands

```bash
make lint           # REQUIRED before committing (tidy → fmt → vet → modernize → golangci-lint)
make test           # REQUIRED — all tests must pass (RACE=1 for race detection)
make test-short     # Run short tests only, skip slow tests
make test-focus     # Run focused tests (FOCUS="pattern")
make test-coverage  # Run when adding new functionality
make test-ci        # CI: race + randomized + json report (uses go run for version pinning)
make docker-build-test  # Local image gokeenapi-test:local, no push
```

CI runs `make test-ci`, not `make test`. Ginkgo is invoked via `go run github.com/onsi/ginkgo/v2/ginkgo` so the version tracks `go.mod`.

## Code Style

### Error Handling

- Use `multierr.Append()` when iterating lists — collect all errors, never just the first
- Return errors from `RunE` — never `os.Exit()` in command functions
- Validate inputs via `gokeenrestapi.Checks` before any API call — fail fast

```go
var errs error
for _, item := range items {
    if err := doSomething(item); err != nil {
        errs = multierr.Append(errs, err)
    }
}
return errs
```

### Logging

Use only `internal/gokeenlog` — never `fmt.Println()`, `log.Println()`, or `print()`. Interactive confirmation in `cmd/common.go` (`confirmAction`) is the exception.

| Function | Use for |
|----------|---------|
| `gokeenlog.Info(msg)` | Top-level step messages |
| `gokeenlog.Infof(msg, args...)` | Formatted top-level messages |
| `gokeenlog.InfoSubStep(msg)` | Bullet-point detail under a step |
| `gokeenlog.InfoSubStepf(msg, args...)` | Formatted bullet-point detail |
| `gokeenlog.HorizontalLine()` | Visual separator between sections |
| `gokeenlog.PrintParseResponse(resp)` | Debug-only API response output |

### API Singletons

Never instantiate. Full method list: `product.md`. Adding a command: `structure.md`.

Auth is automatic via `PersistentPreRunE`. Commands must never call `Auth()` directly. Skip list: `completion`, `help`, `scheduler`, `version`.

## Configuration

- Loaded once in `root.go` via `config.LoadConfig()`; access globally as `config.Cfg` — never reload in commands
- Env vars override YAML: `GOKEENAPI_KEENETIC_LOGIN`, `GOKEENAPI_KEENETIC_PASSWORD`, `GOKEENAPI_CONFIG`
- YAML expansion and required fields: `product.md`
- Schema types: `pkg/config/main_config.go` and `pkg/config/scheduler_config.go`

## Testing

| Test type | File pattern | Location |
|-----------|--------------|----------|
| Unit | `*_test.go` | Same dir as source |
| Property-based | `*_property_test.go` | Same dir as source |
| Integration | `docker_*_test.go` | Repository root |

- All tests use Ginkgo v2 + Gomega — never testify
- Every test package has `suite_test.go`
- Property tests: `rapid.Check(GinkgoT(), ...)` inside Ginkgo `It` blocks
- API tests must call `gokeenrestapi.SetupMockRouterForTest()` — never custom mocks
- `Eventually(...).WithTimeout(...).WithPolling(...).Should(...)` — never positional args
- `cmd` tests use `cmd/helpers_test.go` (`setupMockRouter`, `cleanupMockRouter`, `captureOutput`, `writeTempFile`)
- Mock patterns: `mock-testing-guidelines.md`

`setupMockRouter` also calls `Auth()`. Do not call `Auth()` again in `BeforeEach`.

## Docker

- Multi-arch: `linux/amd64`, `linux/arm64`; image: `noksa/gokeenapi:stable`
- Build via BuildKit: `docker buildx build`; Dockerfile at repository root
- Image sets `GOKEENAPI_INSIDE_DOCKER=true` and data dir `/etc/gokeenapi`

## Anti-Patterns

| Never do this | Do this instead |
|---------------|----------------|
| `fmt.Println()` / `log.Println()` / `print()` | `gokeenlog` functions |
| `net/http` directly in production code | `resty` via `gokeenrestapi.Common.GetApiClient()` |
| `os.Exit()` in command functions | return errors from `RunE` |
| `Run` in cobra commands | `RunE` |
| Collect only first error when iterating | `multierr.Append()` |
| Edit `go.mod` manually | `go get` / `go mod tidy` |
| Omit `SetupMockRouterForTest()` in API tests | always call it |
| Create custom mock implementations | extend `pkg/gokeenrestapi/mock_router.go` (and handler files) |
| Access `config.Cfg` or singletons in constructors | access only inside `RunE` |
| `testify` assertions (`assert.Equal`, `suite.Suite`) | Ginkgo/Gomega (`Expect(x).To(Equal(y))`) |
| `go test` in Makefile or CI | `ginkgo` CLI or `go run ginkgo` |
| `Eventually(fn, timeout, polling)` positional args | `.WithTimeout().WithPolling()` fluent API |
| Call `Auth()` in commands | authentication is automatic via `PersistentPreRunE` |
| Call `Auth()` directly in test `BeforeEach` | use `setupMockRouter()` — it calls `Auth()` internally |
| Store only YAML credentials and commit them | env vars; `chmod 600` on local YAML |
| CIDR-only lines in bat-files | `route add <ip> MASK <mask> <gateway>` |
