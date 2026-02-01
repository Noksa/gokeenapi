---
inclusion: always
---

# Project Structure

```
gokeenapi/
├── main.go                 # Entry point - calls cmd.NewRootCmd().Execute()
├── cmd/                    # CLI commands (Cobra framework)
│   ├── root.go             # Root command, registers all subcommands
│   ├── constants.go        # Command names (CmdXxx) and aliases (AliasesXxx)
│   ├── common.go           # Shared utilities (validation, prompts)
│   ├── <command>.go        # Individual command implementations
│   └── *_test.go           # Command tests
├── pkg/                    # Public packages (importable)
│   ├── config/             # Configuration loading and validation
│   ├── gokeenrestapi/      # Keenetic REST API client
│   └── gokeenrestapimodels/ # API response/request models
├── internal/               # Private packages (not importable externally)
│   ├── gokeenlog/          # Logging (Info, Debug, Error, etc.)
│   ├── gokeencache/        # In-memory caching
│   ├── gokeenspinner/      # CLI progress indicators
│   └── gokeenversion/      # Version info
├── custom/                 # Example configs and domain lists
├── batfiles/               # Example bat files for routes
├── scripts/                # Build and lint scripts
└── Makefile                # Build targets
```

## Command Implementation Pattern

When creating or modifying commands in `cmd/`:

1. **Define constants** in `constants.go`:
   ```go
   const CmdMyCommand = "my-command"
   var AliasesMyCommand = []string{"mycommand", "mc"}
   ```

2. **Create command file** `my_command.go`:
   ```go
   func newMyCommandCmd() *cobra.Command {
       cmd := &cobra.Command{
           Use:     CmdMyCommand,
           Aliases: AliasesMyCommand,
           Short:   "Brief description",
           Long:    `Detailed description with examples`,
       }
       cmd.RunE = func(cmd *cobra.Command, args []string) error {
           // Implementation
           return nil
       }
       return cmd
   }
   ```

3. **Register in `root.go`**:
   ```go
   rootCmd.AddCommand(newMyCommandCmd())
   ```

## API Client Usage

Access router API through singleton instances in `pkg/gokeenrestapi/`:

- `gokeenrestapi.Common.Auth()` - Authenticate (called automatically in PersistentPreRunE)
- `gokeenrestapi.Common.ExecutePostParse()` - Execute CLI commands via RCI
- `gokeenrestapi.Ip.*` - IP route operations
- `gokeenrestapi.DnsRouting.*` - DNS-routing operations
- `gokeenrestapi.Checks.*` - Validation helpers (interface exists, etc.)

## Configuration Access

- `config.Cfg` - Global config instance (loaded in root.go PersistentPreRunE)
- `config.LoadConfig(path)` - Load and validate YAML config
- Config struct: `pkg/config/main_config.go` → `GokeenapiConfig`

## Testing Conventions

- Use `SetupMockRouterForTest()` from `pkg/gokeenrestapi/mock_router.go`
- Property tests: `*_property_test.go` using `rapid` framework
- Test suites: `testify/suite` for grouped tests
- See `mock-testing-guidelines.md` for mock usage rules

## Logging

Use `internal/gokeenlog` for consistent output:
- `gokeenlog.Info()`, `gokeenlog.Debug()`, `gokeenlog.Error()`
- `gokeenlog.InfoSubStepf()` - Indented sub-step messages
- `gokeenlog.HorizontalLine()` - Visual separator
