---
inclusion: always
---

# Project Structure & Architecture

## Directory Layout

```
gokeenapi/
├── main.go                      # Entry point — calls cmd.NewRootCmd().Execute()
├── cmd/                         # CLI commands (Cobra)
│   ├── root.go                  # Root command + PersistentPreRunE (auth, config load)
│   ├── constants.go             # CmdXxx constants and AliasesXxx slices
│   ├── common.go                # Shared validation, prompts, error helpers
│   ├── <command>.go             # One file per command
│   └── *_test.go / *_property_test.go
├── pkg/
│   ├── config/                  # YAML config loading + expansion (main_config.go, scheduler_config.go)
│   ├── gokeenrestapi/           # Keenetic REST API singletons + mock router
│   └── gokeenrestapimodels/     # API request/response structs
├── internal/
│   ├── gokeenlog/               # Structured logging — ONLY logging package to use
│   ├── gokeencache/             # In-memory API response cache
│   ├── gokeenspinner/           # CLI progress indicators
│   └── gokeenversion/           # Build-time version info
├── custom/                      # Example configs and domain lists
├── batfiles/                    # Example IP route bat-files
└── Makefile                     # lint, test, build, coverage targets
```

## Core Architecture Patterns

### Singleton API Clients (`pkg/gokeenrestapi`)
Four singletons cover all router operations — never instantiate them yourself:
- `gokeenrestapi.Common` — auth, RCI execution
- `gokeenrestapi.Ip` — IP route management
- `gokeenrestapi.DnsRouting` — DNS-routing management
- `gokeenrestapi.Checks` — input validation (interface existence, etc.)

Authentication is automatic via `root.go` PersistentPreRunE. Commands never call `Auth()`.

### Global Config (`config.Cfg`)
Loaded once in `root.go` PersistentPreRunE via `config.LoadConfig()`. Commands read it directly — never reload or validate it themselves.

### Command Registration (3-step)
1. Add `CmdXxx` constant + `AliasesXxx` slice in `cmd/constants.go`
2. Implement `newXxxCmd() *cobra.Command` in `cmd/<command>.go`
3. Register via `rootCmd.AddCommand(newXxxCmd())` in `cmd/root.go`

## Adding a New Command

### `cmd/constants.go`
```go
const CmdMyCommand = "my-command"
var AliasesMyCommand = []string{"mycommand", "mc"}
```
Naming: command names are kebab-case; constants use `Cmd` prefix (PascalCase); alias vars use `Aliases` prefix; aliases are compact/no-hyphens.

### `cmd/my_command.go`
```go
func newMyCommandCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:     CmdMyCommand,
        Aliases: AliasesMyCommand,
        Short:   "Brief description",
        Long:    `Detailed description`,
    }
    cmd.Flags().StringP("interface", "i", "", "Interface name")
    cmd.RunE = func(cmd *cobra.Command, args []string) error {
        // 1. Parse flags
        // 2. Validate inputs (use cmd/common.go helpers)
        // 3. Call API singletons
        // 4. Log with gokeenlog
        return nil
    }
    return cmd
}
```

Critical rules:
- Always `RunE`, never `Run`
- Never access `config.Cfg` or API singletons in the constructor — only inside `RunE`
- Never use `fmt.Println` — always `gokeenlog`
- Never call `os.Exit()` — return errors

### `cmd/root.go`
```go
rootCmd.AddCommand(newMyCommandCmd())
```

## API Usage

```go
// Routes
gokeenrestapi.Ip.AddRoutes(routes, interfaceName)
gokeenrestapi.Ip.DeleteRoutes(routes, interfaceName)

// DNS routing
gokeenrestapi.DnsRouting.AddDomains(domains, interfaceName)
gokeenrestapi.DnsRouting.DeleteDomains(domains, interfaceName)

// Validation
exists, err := gokeenrestapi.Checks.InterfaceExists(interfaceName)

// Raw RCI
output, err := gokeenrestapi.Common.ExecutePostParse(command)
```

Multi-operation error aggregation — always use `multierr`:
```go
var errs error
for _, item := range items {
    if err := doSomething(item); err != nil {
        errs = multierr.Append(errs, err)
    }
}
return errs
```

## Configuration

Config fields accessed via `config.Cfg`:
```go
config.Cfg.KeeneticURL
config.Cfg.KeeneticLogin   // prefer env vars
config.Cfg.KeeneticPassword
config.Cfg.BatFiles
config.Cfg.DomainFiles
```

YAML expansion: config files can reference other YAML files by path. `LoadConfig()` resolves them relative to the referencing file's directory.

## Testing

### Unit tests — mock router required for any API test
```go
func TestMyCommand(t *testing.T) {
    cleanup := gokeenrestapi.SetupMockRouterForTest()
    defer cleanup()
    // ...
}
```

### Property-based tests (`*_property_test.go`)
```go
func TestProperty_MyInvariant(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        input := rapid.String().Draw(t, "input")
        result := MyFunction(input)
        if !invariantHolds(result) {
            t.Fatalf("violated for: %v", input)
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

Test file placement:
- `*_test.go` — same directory as source
- `*_property_test.go` — same directory as source
- `docker_*_test.go` — repository root (integration tests)

## Logging

Always use `gokeenlog`. Never `fmt.Println`, `log.Println`, or `print`.

```go
gokeenlog.Info("done")
gokeenlog.Infof("processed %d items", n)
gokeenlog.InfoSubStepf("  - item %s", name)
gokeenlog.Debug("detail")
gokeenlog.Error("failed: %v", err)
gokeenlog.HorizontalLine()
```

## File & Package Conventions

| Concern | Location |
|---|---|
| Shared validation / prompt helpers | `cmd/common.go` |
| Command-specific logic | `cmd/<command>.go` |
| Externally importable code | `pkg/` |
| Internal-only utilities | `internal/` |
| Integration tests | repo root `docker_*_test.go` |

Naming:
- Files: `add_routes.go`, `add_routes_test.go`, `add_routes_property_test.go`
- Command constructors: `newXxxCmd()` (lowercase `new`)
- Exported: PascalCase; unexported: camelCase
- Packages: single lowercase word, no underscores
