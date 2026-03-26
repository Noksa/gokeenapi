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

## Build & Test

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
- Validate inputs (IPs, domains, interface IDs) before any API call — fail fast

### Logging
- Always use `internal/gokeenlog` — never `fmt.Println()`, `log.Println()`, or `print()`
- Available: `Info(msg)`, `Infof(msg, args...)`, `InfoSubStepf(msg, args...)`, `InfoSubStep(msg)`, `HorizontalLine()`, `PrintParseResponse(resp)`

### API Singletons (`pkg/gokeenrestapi/`)
Never instantiate these — use the package-level singletons:
- `gokeenrestapi.Common` — auth, RCI execution, config save
- `gokeenrestapi.Ip` — IP route management
- `gokeenrestapi.DnsRouting` — DNS-routing management
- `gokeenrestapi.Checks` — input validation (`CheckInterfaceId`, `CheckInterfaceExists`, `CheckComponentInstalled`)
- `gokeenrestapi.Interface` — interface listing

Authentication is automatic via `root.go` `PersistentPreRunE`. Commands never call `Auth()` directly.

## Adding a Command (3 steps)

**1. `cmd/constants.go`** — name and aliases
```go
const CmdMyCommand = "my-command"           // kebab-case
var AliasesMyCommand = []string{"mycommand", "mc"} // compact, no hyphens
```

**2. `cmd/my_command.go`** — implementation
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

**3. `cmd/root.go`** — register
```go
rootCmd.AddCommand(newMyCommandCmd())
```

`PersistentPreRunE` skips init for `completion`, `help`, `scheduler`, and `version` commands.

## Configuration

- Loaded once in `root.go` via `config.LoadConfig()`; access globally as `config.Cfg` — never reload in commands
- Env vars override YAML: `GOKEENAPI_KEENETIC_LOGIN`, `GOKEENAPI_KEENETIC_PASSWORD`, `GOKEENAPI_CONFIG`
- YAML expansion: configs can reference other YAML files; paths resolve relative to the referencing file

## Testing

> **IMPORTANT:** Activate the `golang-testing` skill before creating or modifying any test file.
> The skill contains the authoritative patterns for Ginkgo/Gomega, property-based tests, suite bootstrapping, and Makefile targets.

| Test type | File pattern | Location |
|-----------|-------------|----------|
| Unit | `*_test.go` | Same dir as source |
| Property-based | `*_property_test.go` | Same dir as source |
| Integration | `docker_*_test.go` | Repository root |

Key rules (see `golang-testing` skill for full details):
- All tests use Ginkgo v2 (`Describe`/`Context`/`It`) + Gomega (`Expect`) — never testify
- Property tests use `rapid.Check(GinkgoT(), ...)` inside Ginkgo `It` blocks
- Every test package has a `suite_test.go` with `RegisterFailHandler(Fail)` + `RunSpecs()`
- Any test touching the API must call `gokeenrestapi.SetupMockRouterForTest()`
- Use `Eventually(...).WithTimeout(...).WithPolling(...).Should(...)` fluent API — never positional args
- Mock testing follows `mock-testing-guidelines.md` steering

## Docker

- Multi-arch: `linux/amd64`, `linux/arm64`; image: `noksa/gokeenapi:stable`
- Build via BuildKit: `docker buildx build`; Dockerfile at repository root

## Anti-Patterns

| Never do this | Do this instead |
|---------------|----------------|
| `fmt.Println()` / `log.Println()` / `print()` | `gokeenlog` |
| `net/http` directly | `resty` |
| `os.Exit()` in command functions | return errors |
| `Run` in cobra commands | `RunE` |
| Skip `multierr` when iterating lists | `multierr.Append()` |
| Edit `go.mod` manually | `go get` / `go mod tidy` |
| Omit `SetupMockRouterForTest()` in API tests | always call it |
| Access `config.Cfg` or singletons in constructors | access only inside `RunE` |
| `testify` assertions (`assert.Equal`, `suite.Suite`) | Ginkgo/Gomega (`Expect(x).To(Equal(y))`) |
| `go test` in Makefile or CI | `ginkgo` CLI or `go run ginkgo` |
| `Eventually(fn, timeout, polling)` positional args | `.WithTimeout().WithPolling()` fluent API |
