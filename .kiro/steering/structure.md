---
inclusion: always
---

# Project Structure & Architecture

Command list, config schema, and behavioral contracts: `product.md`.
Stack, make targets, and anti-patterns: `tech.md`.
Mock router details: `mock-testing-guidelines.md`.

## Directory Layout

```
gokeenapi/
├── main.go                      # Entry point — cmd.NewRootCmd().Execute()
├── cmd/                         # CLI commands (Cobra)
│   ├── root.go                  # Root command, PersistentPreRunE (config load + auth)
│   ├── constants.go             # CmdXxx constants and AliasesXxx slices
│   ├── common.go                # checkRequiredFields, confirmAction, RestoreCursor
│   ├── helpers_test.go          # setupMockRouter, cleanupMockRouter, writeTempFile, captureOutput
│   ├── configure_awg.go         # implements update-awg (historical name — do not add update_awg.go)
│   ├── <command>.go             # One file per command (snake_case)
│   └── *_test.go
├── pkg/
│   ├── config/                  # YAML load + expansion; access via config.Cfg
│   ├── gokeenrestapi/           # API singletons + unified mock router
│   └── gokeenrestapimodels/     # Request/response structs — always use these types
├── internal/
│   ├── gokeenlog/               # The only allowed logging package
│   ├── gokeencache/             # In-memory API / URL-content cache
│   ├── gokeenspinner/           # CLI progress spinners
│   └── gokeenversion/           # Build-time version info
├── docs/                        # Human config reference (EN + RU) — not agent instructions
├── scripts/                     # check.sh (lint), build.sh, create_binaries.sh
├── batfiles/                    # Example Windows-style route bat-files
├── custom/                      # Local example router configs (gitignored patterns apply)
├── .github/workflows/           # ci.yml, release.yml
├── docker_*_test.go             # Docker integration tests (repo root)
├── config_example.yaml          # Annotated single-router config
├── scheduler_example.yaml       # Annotated scheduler config
├── Dockerfile                   # Multi-arch image (linux/amd64, linux/arm64)
└── Makefile                     # lint, test, build, coverage targets
```

`custom/` holds operator-specific configs and domain lists used with this checkout. Do not treat it as public product docs.

## File & Package Naming

- Source files: `snake_case` — `add_routes.go`, `add_routes_test.go`
- Command constructors: `newXxxCmd()` (lowercase `new`)
- Exported symbols: `PascalCase`; unexported: `camelCase`
- Package names: single lowercase word, no underscores
- Shared helpers go in `cmd/common.go`; command-specific logic in `cmd/<command>.go`
- Externally importable code in `pkg/`; internal-only utilities in `internal/`
- CLI layer in `cmd/` must not grow business logic — that belongs in `pkg/` or `internal/`

## API Singletons (`pkg/gokeenrestapi`)

Never instantiate — use package-level values only. Method names: `product.md`.

Authentication is automatic via `PersistentPreRunE` in `root.go`. Commands must **never** call `Auth()` directly.

`PersistentPreRunE` skips config load and auth for: `completion`, `help`, `scheduler`, `version`.

## Global Config

`config.Cfg` is loaded once in `root.go` via `config.LoadConfig()`. Rules:

- Access `config.Cfg` only inside `RunE` — never in command constructors
- Never call `config.LoadConfig()` in commands
- Env vars override YAML: `GOKEENAPI_KEENETIC_LOGIN`, `GOKEENAPI_KEENETIC_PASSWORD`, `GOKEENAPI_CONFIG`

## Adding a Command (3 steps)

**Step 1 — `cmd/constants.go`**: define name and aliases

```go
const CmdMyCommand = "my-command"                    // kebab-case
var AliasesMyCommand = []string{"mycommand", "mc"}   // compact, typically no hyphens
```

**Step 2 — `cmd/my_command.go`**: implement

```go
func newMyCommandCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:     CmdMyCommand,
        Aliases: AliasesMyCommand,
        Short:   "Brief description",
        Long:    `Detailed description`,
    }
    cmd.RunE = func(cmd *cobra.Command, args []string) error {
        // 1. Parse flags
        // 2. Validate via gokeenrestapi.Checks
        // 3. Call API singletons
        // 4. Log with gokeenlog
        return nil
    }
    return cmd
}
```

**Step 3 — `cmd/root.go`**: register inside `NewRootCmd()`:

```go
rootCmd.AddCommand(newMyCommandCmd())
```

Then: unit tests in `cmd/my_command_test.go`; update `README.md` / `docs/` / `custom/` examples if the change is user-visible. Property tests (`*_property_test.go`) belong with non-trivial invariants (parsing, expansion, idempotency), not every command.

### Enforced Command Rules

- Always `RunE`, never `Run`
- Never access `config.Cfg` or API singletons in the constructor — only inside `RunE`
- Never use `fmt.Println` / `log.Println` / `print` — always `gokeenlog` (exception: `confirmAction` in `common.go` writes an interactive prompt)
- Never call `os.Exit()` — return errors from `RunE`
- Always validate inputs via `gokeenrestapi.Checks` before any API call

## API Usage Patterns

```go
gokeenrestapi.Checks.CheckInterfaceId(id)
gokeenrestapi.Checks.CheckInterfaceExists(id)

gokeenrestapi.Ip.AddRoutesFromBatFile(absPath, interfaceID)
gokeenrestapi.Ip.AddRoutesFromBatUrl(url, interfaceID)
gokeenrestapi.Ip.DeleteRoutes(routes, interfaceName)
gokeenrestapi.Ip.DeleteAllRoutes()

gokeenrestapi.DnsRouting.AddDnsRoutingGroups(config.Cfg.DNS.Routes.Groups)
gokeenrestapi.DnsRouting.DeleteDnsRoutingGroups(config.Cfg.DNS.Routes.Groups)
gokeenrestapi.DnsRouting.DeleteAllDnsRoutingGroups()

output, err := gokeenrestapi.Common.ExecutePostParse(command)
```

Multi-operation error aggregation — collect all errors, never just the first:

```go
var errs error
for _, item := range items {
    if err := doSomething(item); err != nil {
        errs = multierr.Append(errs, err)
    }
}
return errs
```

## Logging

`internal/gokeenlog` is the only allowed logging package. Function table: `tech.md`.

## Testing

Mock-router rules: `mock-testing-guidelines.md`.

| Test type | File pattern | Location |
|---|---|---|
| Unit | `*_test.go` | Same dir as source |
| Property-based | `*_property_test.go` | Same dir as source |
| Integration | `docker_*_test.go` | Repository root |

Key rules:

- All tests use Ginkgo v2 + Gomega — never testify
- Every test package has `suite_test.go` with `RegisterFailHandler(Fail)` + `RunSpecs()`
- Property tests use `rapid.Check(GinkgoT(), ...)` inside Ginkgo `It` blocks
- Any test touching the API must call `gokeenrestapi.SetupMockRouterForTest()` — never create custom mocks
- Use `Eventually(...).WithTimeout(...).WithPolling(...).Should(...)` fluent API — never positional args

### Standard Test Setup (`cmd` package)

Helpers in `cmd/helpers_test.go`:

```go
var _ = Describe("MyCommand", func() {
    var server *httptest.Server

    BeforeEach(func() {
        server = setupMockRouter() // SetupMockRouterForTest + Auth
    })

    AfterEach(func() {
        cleanupMockRouter(server) // Close + CleanupTestConfig
    })

    It("should do something", func() {
        cmd := newMyCommandCmd()
        Expect(cmd.RunE(cmd, []string{})).To(Succeed())
    })
})
```

- `setupMockRouter(opts ...MockRouterOption) *httptest.Server`
- `cleanupMockRouter(server)`
- `captureOutput(cmd, args) (string, error)`
- `writeTempFile(dir, name, content string) string`

`setupMockRouter` accepts `gokeenrestapi.WithInterfaces(...)`, `WithRoutes(...)`, and other options listed in `mock-testing-guidelines.md`.
