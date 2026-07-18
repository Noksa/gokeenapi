---
inclusion: always
---

# Tech Stack & Development Rules

## Language & Runtime

- Go 1.25+ with standard project layout (`cmd/`, `pkg/`, `internal/`)
- Never modify `go.mod` manually — use `go get` or `go mod tidy`

## Key Dependencies

| Package | Import Path | Purpose |
|---------|-------------|---------|
| cobra | `github.com/spf13/cobra` | CLI — always `RunE`, never `Run` |
| resty | `github.com/go-resty/resty/v2` | HTTP — never use `net/http` directly |
| yaml.v3 | `gopkg.in/yaml.v3` | YAML parsing via `yaml.Unmarshal()` |
| ginkgo/v2 | `github.com/onsi/ginkgo/v2` | Test framework — all tests use Ginkgo `Describe`/`It` |
| gomega | `github.com/onsi/gomega` | Matchers — `Expect(x).To(Equal(y))`, never testify |
| rapid | `pgregory.net/rapid` | Property-based tests in `*_property_test.go` |
| multierr | `go.uber.org/multierr` | Error aggregation: `multierr.Append(errs, err)` |
| go-version | `github.com/hashicorp/go-version` | Version comparison |

Request/response structs live in `pkg/gokeenrestapimodels/` — always use those types with API singletons.

## Build & Test Commands

```bash
make lint           # REQUIRED before committing (tidy → fmt → vet → modernize → golangci-lint)
make test           # REQUIRED — all tests must pass (RACE=1 for race detection)
make test-short     # Run short tests only, skip slow tests
make test-focus     # Run focused tests (FOCUS="pattern")
make test-coverage  # Run when adding new functionality
make test-ci        # CI: race + randomized + json report (uses go run for version pinning)
```

## Code Style

### Error Handling

- Use `multierr.Append()` when iterating lists — collect all errors, not just the first
- Return errors from `RunE` — never `os.Exit()` in command functions
- Validate inputs (IPs, domains, interface IDs) via `gokeenrestapi.Checks` before any API call — fail fast

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

Use only `internal/gokeenlog` — never `fmt.Println()`, `log.Println()`, or `print()`.

| Function | Use for |
|----------|---------|
| `gokeenlog.Info(msg)` | Top-level step messages |
| `gokeenlog.Infof(msg, args...)` | Formatted top-level messages |
| `gokeenlog.InfoSubStep(msg)` | Bullet-point detail under a step |
| `gokeenlog.InfoSubStepf(msg, args...)` | Formatted bullet-point detail |
| `gokeenlog.HorizontalLine()` | Visual separator between sections |
| `gokeenlog.PrintParseResponse(resp)` | Debug-only API response output |

### API Singletons (`pkg/gokeenrestapi/`)

Never instantiate — use only the package-level singletons:

| Singleton | Responsibility |
|-----------|---------------|
| `gokeenrestapi.Common` | Auth, raw RCI execution (`ExecutePostParse`, `ExecuteGetSubPath`), config save |
| `gokeenrestapi.Ip` | IP route management (`AddRoutesFromBatFile`, `AddRoutesFromBatUrl`, `DeleteRoutes`, `DeleteAllRoutes`) |
| `gokeenrestapi.DnsRouting` | DNS-routing management (`AddDomains`, `DeleteDomains`) |
| `gokeenrestapi.AwgConf` | AWG/WireGuard configuration and diff-update |
| `gokeenrestapi.Checks` | Input validation (`CheckInterfaceId`, `CheckInterfaceExists`, `CheckComponentInstalled`) |
| `gokeenrestapi.Interface` | Interface listing |

Authentication is automatic via `PersistentPreRunE` in `root.go`. Commands must never call `Auth()` directly.

## Adding a Command (3 Steps)

**Step 1 — `cmd/constants.go`**: define name and aliases

```go
const CmdMyCommand = "my-command"                        // kebab-case
var AliasesMyCommand = []string{"mycommand", "mc"}       // compact, no hyphens
```

**Step 2 — `cmd/my_command.go`**: implement

```go
func newMyCommandCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:     CmdMyCommand,
        Aliases: AliasesMyCommand,
        Short:   "Brief description",
    }
    cmd.RunE = func(cmd *cobra.Command, args []string) error {
        // 1. Parse flags
        // 2. Validate inputs via gokeenrestapi.Checks
        // 3. Call API singletons
        // 4. Log with gokeenlog
        return nil
    }
    return cmd
}
```

- Never access `config.Cfg` or API singletons in the constructor — only inside `RunE`

**Step 3 — `cmd/root.go`**: register

```go
rootCmd.AddCommand(newMyCommandCmd())
```

`PersistentPreRunE` skips auth/config init for `completion`, `help`, `scheduler`, and `version` commands.

## Configuration

- Loaded once in `root.go` via `config.LoadConfig()`; access globally as `config.Cfg` — never reload in commands
- Env vars override YAML: `GOKEENAPI_KEENETIC_LOGIN`, `GOKEENAPI_KEENETIC_PASSWORD`, `GOKEENAPI_CONFIG`
- YAML expansion: configs can reference other YAML files; paths resolve relative to the referencing file
- Only required field: `keenetic-url`; credentials should use env vars, not plain YAML

## Testing

> **IMPORTANT:** Activate the `golang-testing` skill before creating or modifying any test file.

| Test type | File pattern | Location |
|-----------|--------------|----------|
| Unit | `*_test.go` | Same dir as source |
| Property-based | `*_property_test.go` | Same dir as source |
| Integration | `docker_*_test.go` | Repository root |

Key rules:
- All tests use Ginkgo v2 (`Describe`/`Context`/`It`) + Gomega (`Expect`) — never testify
- Every test package has a `suite_test.go` with `RegisterFailHandler(Fail)` + `RunSpecs()`
- Property tests use `rapid.Check(GinkgoT(), ...)` inside Ginkgo `It` blocks
- Any test touching the API must call `gokeenrestapi.SetupMockRouterForTest()` — never create custom mocks
- Use `Eventually(...).WithTimeout(...).WithPolling(...).Should(...)` fluent API — never positional args
- Use `cmd/helpers_test.go` helpers (`setupMockRouter`, `cleanupMockRouter`, `writeTempFile`) in `cmd` package tests
- Mock testing patterns are governed by the `mock-testing-guidelines` steering file

### Standard Test Setup Pattern (cmd package)

```go
var _ = Describe("MyCommand", func() {
    var server *httptest.Server

    BeforeEach(func() {
        server = setupMockRouter()  // from cmd/helpers_test.go
    })

    AfterEach(func() {
        cleanupMockRouter(server)
    })

    It("should do something", func() {
        cmd := newMyCommandCmd()
        Expect(cmd.RunE(cmd, []string{})).To(Succeed())
    })
})
```

## Docker

- Multi-arch: `linux/amd64`, `linux/arm64`; image: `noksa/gokeenapi:stable`
- Build via BuildKit: `docker buildx build`; Dockerfile at repository root

## Anti-Patterns

| Never do this | Do this instead |
|---------------|----------------|
| `fmt.Println()` / `log.Println()` / `print()` | `gokeenlog` functions |
| `net/http` directly | `resty` via `gokeenrestapi.Common.GetApiClient()` |
| `os.Exit()` in command functions | return errors from `RunE` |
| `Run` in cobra commands | `RunE` |
| Collect only first error when iterating | `multierr.Append()` |
| Edit `go.mod` manually | `go get` / `go mod tidy` |
| Omit `SetupMockRouterForTest()` in API tests | always call it |
| Create custom mock implementations | extend unified mock at `pkg/gokeenrestapi/mock_router.go` |
| Access `config.Cfg` or singletons in constructors | access only inside `RunE` |
| `testify` assertions (`assert.Equal`, `suite.Suite`) | Ginkgo/Gomega (`Expect(x).To(Equal(y))`) |
| `go test` in Makefile or CI | `ginkgo` CLI or `go run ginkgo` |
| `Eventually(fn, timeout, polling)` positional args | `.WithTimeout().WithPolling()` fluent API |
| Call `Auth()` in commands | authentication is automatic via `PersistentPreRunE` |
