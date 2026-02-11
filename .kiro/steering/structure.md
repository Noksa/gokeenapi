---
inclusion: always
---

# Project Structure & Architecture

## Directory Layout

```
gokeenapi/
├── main.go                      # Entry point - calls cmd.NewRootCmd().Execute()
├── cmd/                         # CLI commands (Cobra framework)
│   ├── root.go                  # Root command + PersistentPreRunE (auth, config load)
│   ├── constants.go             # Command names (CmdXxx) and aliases (AliasesXxx)
│   ├── common.go                # Shared utilities (validation, prompts, error handling)
│   ├── <command>.go             # Individual command implementations
│   └── *_test.go                # Command unit tests
├── pkg/                         # Public packages (importable by external projects)
│   ├── config/                  # YAML config loading, validation, expansion
│   │   ├── main_config.go       # GokeenapiConfig struct + LoadConfig()
│   │   └── scheduler_config.go  # Scheduler-specific config
│   ├── gokeenrestapi/           # Keenetic REST API client (singleton pattern)
│   │   ├── common.go            # Auth, RCI execution, base HTTP client
│   │   ├── ip.go                # IP route operations
│   │   ├── dns_routing.go       # DNS-routing operations
│   │   ├── awgconf.go           # WireGuard configuration
│   │   ├── checks.go            # Validation helpers
│   │   └── mock_router.go       # Unified mock for testing
│   └── gokeenrestapimodels/     # API request/response models
├── internal/                    # Private packages (not importable externally)
│   ├── gokeenlog/               # Structured logging (Info, Debug, Error, etc.)
│   ├── gokeencache/             # In-memory caching for API responses
│   ├── gokeenspinner/           # CLI progress indicators
│   └── gokeenversion/           # Version info embedded at build time
├── custom/                      # Example configs and domain lists
│   ├── config_*.yaml            # Per-router configuration examples
│   ├── common_dns_groups.yaml   # Shared DNS group definitions
│   └── domains/                 # Domain list files for DNS-routing
├── batfiles/                    # Example bat files for IP routes
├── scripts/                     # Build, lint, and CI scripts
└── Makefile                     # Build targets (lint, test, build, coverage)
```

## Architecture Patterns

### Singleton API Client Pattern
The `pkg/gokeenrestapi` package uses singleton instances for API operations:
- `gokeenrestapi.Common` - Authentication and RCI execution
- `gokeenrestapi.Ip` - IP route management
- `gokeenrestapi.DnsRouting` - DNS-routing management
- `gokeenrestapi.Checks` - Validation helpers

These singletons are initialized once and reused across commands. Authentication happens automatically in `root.go` PersistentPreRunE.

### Global Configuration Pattern
- `config.Cfg` is a global variable holding the loaded configuration
- Loaded in `root.go` PersistentPreRunE before any command runs
- Commands access config directly via `config.Cfg`
- Config validation happens at load time, not during command execution

### Command Registration Pattern
All commands follow a three-step pattern:
1. Define constants in `cmd/constants.go`
2. Implement `newXxxCmd()` constructor in `cmd/xxx.go`
3. Register in `cmd/root.go` via `rootCmd.AddCommand()`

### Error Handling Pattern
- Use `multierr.Append()` to aggregate errors when processing lists
- Return errors from `RunE` functions, don't call `os.Exit()` directly
- Cobra handles error display and exit codes
- Validation errors should be clear and actionable

## Command Implementation Rules

### Step 1: Define Constants in `cmd/constants.go`
```go
const CmdMyCommand = "my-command"
var AliasesMyCommand = []string{"mycommand", "mc"}
```

Naming conventions:
- Command names: kebab-case (`add-dns-routing`)
- Constant names: PascalCase with `Cmd` prefix (`CmdAddDnsRouting`)
- Alias variables: PascalCase with `Aliases` prefix (`AliasesAddDnsRouting`)
- Aliases: compact forms without hyphens (`adddnsrouting`, `adnsr`)

### Step 2: Create Command File `cmd/my_command.go`
```go
func newMyCommandCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:     CmdMyCommand,
        Aliases: AliasesMyCommand,
        Short:   "Brief one-line description",
        Long:    `Detailed description with examples and usage patterns`,
    }
    
    // Define flags if needed
    cmd.Flags().StringP("interface", "i", "", "Interface name")
    
    cmd.RunE = func(cmd *cobra.Command, args []string) error {
        // 1. Parse flags and arguments
        // 2. Validate inputs using cmd/common.go helpers
        // 3. Call API client methods from pkg/gokeenrestapi
        // 4. Use gokeenlog for output
        // 5. Return error or nil
        return nil
    }
    
    return cmd
}
```

Key rules:
- Use `RunE` not `Run` to return errors properly
- Don't access `config.Cfg` or API singletons in command constructor
- Validation should happen before API calls
- Use `gokeenlog` for all output, never `fmt.Println`

### Step 3: Register in `cmd/root.go`
```go
rootCmd.AddCommand(newMyCommandCmd())
```

Add the registration call in the `init()` function after other commands.

## API Client Usage Patterns

### Authentication
Authentication is automatic - handled in `root.go` PersistentPreRunE:
```go
if err := gokeenrestapi.Common.Auth(); err != nil {
    return err
}
```

Commands don't need to call `Auth()` explicitly.

### Making API Calls
```go
// IP route operations
err := gokeenrestapi.Ip.AddRoutes(routes, interfaceName)
err := gokeenrestapi.Ip.DeleteRoutes(routes, interfaceName)

// DNS-routing operations
err := gokeenrestapi.DnsRouting.AddDomains(domains, interfaceName)
err := gokeenrestapi.DnsRouting.DeleteDomains(domains, interfaceName)

// Validation
exists, err := gokeenrestapi.Checks.InterfaceExists(interfaceName)

// RCI command execution
output, err := gokeenrestapi.Common.ExecutePostParse(command)
```

### Error Handling with Multiple Operations
```go
var errs error
for _, route := range routes {
    if err := gokeenrestapi.Ip.AddRoute(route, iface); err != nil {
        errs = multierr.Append(errs, err)
    }
}
return errs
```

## Configuration Access Patterns

### Loading Config
Config is loaded automatically in `root.go` PersistentPreRunE:
```go
config.Cfg, err = config.LoadConfig(configPath)
```

### Accessing Config in Commands
```go
// Access router URL
url := config.Cfg.KeeneticURL

// Access credentials (if not in env vars)
login := config.Cfg.KeeneticLogin
password := config.Cfg.KeeneticPassword

// Access resource lists
batFiles := config.Cfg.BatFiles
domainFiles := config.Cfg.DomainFiles
```

### Config Expansion Pattern
YAML configs support referencing other YAML files:
```yaml
domain-files:
  - path: domains/telegram.yaml  # References another YAML with domain list
```

The `LoadConfig()` function automatically expands these references and resolves paths relative to the referencing file's directory.

## Testing Patterns

### Unit Tests with Mock Router
```go
func TestMyCommand(t *testing.T) {
    // Setup mock router
    cleanup := gokeenrestapi.SetupMockRouterForTest()
    defer cleanup()
    
    // Test command logic
    cmd := newMyCommandCmd()
    err := cmd.RunE(cmd, []string{})
    assert.NoError(t, err)
}
```

### Property-Based Tests
File naming: `*_property_test.go`

```go
func TestProperty_MyInvariant(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // Generate test inputs
        input := rapid.String().Draw(t, "input")
        
        // Test invariant
        result := MyFunction(input)
        
        // Assert property holds
        if !PropertyHolds(result) {
            t.Fatalf("property violated for input: %v", input)
        }
    })
}
```

### Test Suite Pattern
```go
type MyTestSuite struct {
    suite.Suite
}

func (s *MyTestSuite) SetupTest() {
    // Setup before each test
}

func (s *MyTestSuite) TestSomething() {
    s.NoError(err)
    s.Equal(expected, actual)
}

func TestMyTestSuite(t *testing.T) {
    suite.Run(t, new(MyTestSuite))
}
```

## Logging Patterns

Always use `internal/gokeenlog` for output:

```go
// Standard messages
gokeenlog.Info("Operation completed successfully")
gokeenlog.Debug("Detailed debug information")
gokeenlog.Error("Error occurred: %v", err)

// Formatted messages
gokeenlog.Infof("Processing %d items", count)

// Sub-step messages (indented)
gokeenlog.InfoSubStepf("  - Processing item %s", name)

// Visual separators
gokeenlog.HorizontalLine()
```

Never use:
- `fmt.Println()` - breaks log formatting
- `log.Println()` - bypasses gokeenlog structure
- `os.Exit()` in commands - return errors instead

## File Organization Rules

### When to Use `cmd/common.go`
Place in `cmd/common.go`:
- Validation functions used by multiple commands
- User prompt helpers
- Shared error formatting
- Common flag parsing logic

Don't place in `cmd/common.go`:
- Command-specific logic
- API client calls
- Business logic (belongs in `pkg/`)

### When to Use `pkg/` vs `internal/`
Use `pkg/`:
- Code that external projects might import
- API client implementations
- Configuration structures
- Data models

Use `internal/`:
- Code specific to this CLI tool
- Logging utilities
- Caching implementations
- Version information

### Test File Placement
- Place tests in the same directory as the code being tested
- Use `*_test.go` for unit tests
- Use `*_property_test.go` for property-based tests
- Integration tests at repository root: `docker_*_test.go`

## Naming Conventions

### Files
- Command files: `<command_name>.go` (e.g., `add_routes.go`)
- Test files: `<name>_test.go`
- Property test files: `<name>_property_test.go`

### Functions
- Command constructors: `newXxxCmd()` (lowercase `new`, returns `*cobra.Command`)
- Exported functions: PascalCase
- Private functions: camelCase

### Variables
- Constants: PascalCase with prefix (e.g., `CmdAddRoutes`)
- Global variables: PascalCase (e.g., `config.Cfg`)
- Local variables: camelCase

### Packages
- Single word, lowercase (e.g., `config`, `gokeenlog`)
- No underscores or hyphens
- Descriptive but concise
