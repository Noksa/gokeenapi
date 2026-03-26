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
│   ├── config/                  # YAML config loading + expansion
│   ├── gokeenrestapi/           # Keenetic REST API singletons + mock router
│   └── gokeenrestapimodels/     # API request/response structs — always use these types
├── internal/
│   ├── gokeenlog/               # Structured logging — the ONLY logging package allowed
│   ├── gokeencache/             # In-memory API response cache
│   ├── gokeenspinner/           # CLI progress indicators
│   └── gokeenversion/           # Build-time version info
├── custom/                      # Example configs and domain lists
├── batfiles/                    # Example IP route bat-files
└── Makefile                     # lint, test, build, coverage targets
```

## API Singletons (`pkg/gokeenrestapi`)

Never instantiate these — use the package-level singletons only:

| Singleton | Responsibility |
|---|---|
| `gokeenrestapi.Common` | Auth, raw RCI execution |
| `gokeenrestapi.Ip` | IP route management |
| `gokeenrestapi.DnsRouting` | DNS-routing management |
| `gokeenrestapi.Checks` | Input validation (`CheckInterfaceId`, `CheckInterfaceExists`, `CheckComponentInstalled`) |
| `gokeenrestapi.Interface` | Interface listing |

Authentication is handled automatically by `PersistentPreRunE` in `root.go`. Commands must never call `Auth()` directly.

`PersistentPreRunE` skips init for `completion`, `help`, `scheduler`, and `version` commands.

## Global Config

`config.Cfg` is loaded once in `root.go` via `config.LoadConfig()`. Access it directly inside `RunE` — never reload it in commands, and never access it in command constructors.

## Adding a New Command (3 steps)

**Step 1 — `cmd/constants.go`**: define name and aliases
```go
const CmdMyCommand = "my-command"           // kebab-case
var AliasesMyCommand = []string{"mycommand", "mc"} // compact, no hyphens
```

**Step 2 — `cmd/my_command.go`**: implement the command
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
        // 2. Validate inputs via gokeenrestapi.Checks
        // 3. Call API singletons
        // 4. Log with gokeenlog
        return nil
    }
    return cmd
}
```

**Step 3 — `cmd/root.go`**: register
```go
rootCmd.AddCommand(newMyCommandCmd())
```

### Command Rules (enforced)
- Always `RunE`, never `Run`
- Never access `config.Cfg` or API singletons in the constructor — only inside `RunE`
- Never use `fmt.Println` / `log.Println` / `print` — always `gokeenlog`
- Never call `os.Exit()` — return errors

## API Usage Patterns

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

Multi-operation error aggregation — always collect all errors, not just the first:
```go
var errs error
for _, item := range items {
    if err := doSomething(item); err != nil {
        errs = multierr.Append(errs, err)
    }
}
return errs
```

## Testing

> **IMPORTANT:** Activate the `golang-testing` skill before creating or modifying any test file.
> The skill is the single source of truth for test patterns, structure, and conventions.

| Test type | File pattern | Location |
|---|---|---|
| Unit | `*_test.go` | Same dir as source |
| Property-based | `*_property_test.go` | Same dir as source |
| Integration | `docker_*_test.go` | Repository root |

Key rules (see `golang-testing` skill for full details):
- All tests use Ginkgo v2 + Gomega — never testify
- Every test package has `suite_test.go` with `RegisterFailHandler(Fail)` + `RunSpecs()`
- Property tests use `rapid.Check(GinkgoT(), ...)` inside Ginkgo `It` blocks
- Any test touching the API must call `gokeenrestapi.SetupMockRouterForTest()`
- Mock testing follows `mock-testing-guidelines.md` steering

## Logging

Always use `gokeenlog`. Available functions:

```go
gokeenlog.Info("done")
gokeenlog.Infof("processed %d items", n)
gokeenlog.InfoSubStepf("  - item %s", name)
gokeenlog.InfoSubStep("  - item")
gokeenlog.HorizontalLine()
gokeenlog.PrintParseResponse(resp)
```

## File & Package Naming Conventions

| Concern | Location |
|---|---|
| Shared validation / prompt helpers | `cmd/common.go` |
| Command-specific logic | `cmd/<command>.go` |
| Externally importable code | `pkg/` |
| Internal-only utilities | `internal/` |
| Integration tests | repo root `docker_*_test.go` |

- Files: `snake_case` — e.g. `add_routes.go`, `add_routes_test.go`, `add_routes_property_test.go`
- Command constructors: `newXxxCmd()` (lowercase `new`)
- Exported symbols: PascalCase; unexported: camelCase
- Package names: single lowercase word, no underscores
