---
inclusion: always
---

# Tech Stack & Development Rules

## Language & Runtime

- Go 1.25+ with standard project layout (`cmd/`, `pkg/`, `internal/`)
- `go.mod` is the source of truth for module version and dependencies
- Never modify `go.mod` manually — use `go get` or `go mod tidy`

## Key Dependencies

| Package | Import Path | Purpose |
|---------|-------------|---------|
| cobra | `github.com/spf13/cobra` | CLI commands — always use `RunE`, never `Run` |
| resty | `github.com/go-resty/resty/v2` | All HTTP calls — never use `net/http` directly |
| yaml.v3 | `gopkg.in/yaml.v3` | YAML parsing via `yaml.Unmarshal()` |
| testify | `github.com/stretchr/testify` | Test assertions: `assert.NoError()`, `assert.Equal()` |
| rapid | `pgregory.net/rapid` | Property-based tests in `*_property_test.go` files |
| multierr | `go.uber.org/multierr` | Error aggregation: `multierr.Append(errs, err)` |
| go-version | `github.com/hashicorp/go-version` | Version comparison via `version.NewVersion()` |

Request/response structs live in `pkg/gokeenrestapimodels/` — use those types when calling API singletons.

## Build & Test Commands

```bash
make lint           # REQUIRED before committing (tidy → fmt → vet → modernize → golangci-lint)
make test           # REQUIRED — all tests must pass
make test-coverage  # Use when adding new functionality
```

## Code Style

### Error Handling
- Use `multierr.Append()` when iterating lists so all errors are collected, not just the first
- Return errors from `RunE` — never call `os.Exit()` in command functions
- Validate inputs (IPs, domains, interface IDs) before any API call — fail fast

### Logging
- Always use `internal/gokeenlog` for all output — never `fmt.Println()`, `log.Println()`, or `print()`
- Available functions: `Info(msg)`, `Infof(msg, args...)`, `InfoSubStepf(msg, args...)`, `InfoSubStep(msg)`, `HorizontalLine()`, `PrintParseResponse(resp)`

### API Singletons (`pkg/gokeenrestapi/`)
Four singletons cover all router operations — never instantiate them yourself:
- `gokeenrestapi.Common` — auth, RCI execution, config save
- `gokeenrestapi.Ip` — IP route management
- `gokeenrestapi.DnsRouting` — DNS-routing management
- `gokeenrestapi.Checks` — input validation (`CheckInterfaceId`, `CheckInterfaceExists`, `CheckComponentInstalled`)
- `gokeenrestapi.Interface` — interface listing

Authentication is handled automatically in `root.go` `PersistentPreRunE`. Commands never call `Auth()` directly.

## Command Implementation Pattern

### 1. `cmd/constants.go` — register name and aliases
```go
const CmdMyCommand = "my-command"
var AliasesMyCommand = []string{"mycommand", "mc"}
```
- Command names: kebab-case
- Constants: `Cmd` prefix, PascalCase
- Alias vars: `Aliases` prefix; aliases are compact (no hyphens)

### 2. `cmd/my_command.go` — implement the command
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
- Never use `fmt.Println` — always `gokeenlog`

### 3. `cmd/root.go` — register the command
```go
rootCmd.AddCommand(newMyCommandCmd())
```

`PersistentPreRunE` skips initialization for `completion`, `help`, `scheduler`, and `version` commands.

## Configuration

- Config is loaded once in `root.go` `PersistentPreRunE` via `config.LoadConfig()`
- Access globally via `config.Cfg` — never reload it in commands
- Environment variables override YAML values:
  - `GOKEENAPI_KEENETIC_LOGIN` — router username
  - `GOKEENAPI_KEENETIC_PASSWORD` — router password
  - `GOKEENAPI_CONFIG` — config file path
- YAML expansion: config files can reference other YAML files; paths resolve relative to the referencing file

## Testing

### File placement
- Unit tests: `*_test.go` — same directory as source
- Property tests: `*_property_test.go` — same directory as source
- Integration tests: `docker_*_test.go` — repository root

### Any test touching the API must set up the mock router
```go
func TestMyFunction(t *testing.T) {
    cleanup := gokeenrestapi.SetupMockRouterForTest()
    defer cleanup()
    // ...
}
```

### Property test naming and structure
```go
func TestProperty_MyInvariant(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        input := rapid.String().Draw(t, "input")
        result := MyFunction(input)
        if !invariantHolds(result) {
            t.Fatalf("invariant violated for input: %v", input)
        }
    })
}
```

### Test suite pattern
```go
type MyTestSuite struct{ suite.Suite }

func (s *MyTestSuite) TestSomething() { s.NoError(err) }

func TestMyTestSuite(t *testing.T) { suite.Run(t, new(MyTestSuite)) }
```

## Docker

- Multi-arch: `linux/amd64`, `linux/arm64`
- Image: `noksa/gokeenapi:stable`
- Build via BuildKit: `docker buildx build`
- Dockerfile at repository root

## Anti-Patterns (never do these)

- `fmt.Println()` / `log.Println()` / `print()` — use `gokeenlog`
- `net/http` directly — use `resty`
- `os.Exit()` in command functions — return errors
- `Run` instead of `RunE` in cobra commands
- Skipping `multierr` when iterating lists
- Editing `go.mod` manually — use `go get` / `go mod tidy`
- Omitting `SetupMockRouterForTest()` in API tests
- Accessing `config.Cfg` or singletons in command constructors
