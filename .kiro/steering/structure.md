# Project Structure

```
gokeenapi/
├── main.go                 # Entry point
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go             # Root command, registers all subcommands
│   ├── constants.go        # Command names and aliases
│   ├── common.go           # Shared utilities (validation, prompts)
│   ├── add_routes.go       # Route management commands
│   ├── add_dns_routing.go  # DNS-routing commands
│   ├── scheduler.go        # Scheduler command
│   └── *_test.go           # Command tests
├── pkg/                    # Public packages
│   ├── config/             # Configuration loading and validation
│   │   ├── main_config.go  # Main config structures and loading
│   │   └── scheduler_config.go  # Scheduler-specific config
│   ├── gokeenrestapi/      # Keenetic REST API client
│   │   ├── common.go       # Auth, HTTP client, core API methods
│   │   ├── dns_routing.go  # DNS-routing API operations
│   │   ├── ip.go           # IP route operations
│   │   ├── mock_router.go  # Unified mock for testing
│   │   └── *_property_test.go  # Property-based tests
│   └── gokeenrestapimodels/ # API response/request models
├── internal/               # Private packages
│   ├── gokeenlog/          # Logging utilities
│   ├── gokeencache/        # Caching utilities
│   ├── gokeenspinner/      # CLI spinner/progress
│   └── gokeenversion/      # Version info
├── custom/                 # Example configs and domain lists
│   ├── domains/            # Domain list files (.txt, .yaml)
│   └── *.yaml              # Example router configs
├── batfiles/               # Example bat files for routes
├── scripts/                # Build and lint scripts
└── Makefile                # Build targets
```

## Key Patterns

### Command Structure

Commands in `cmd/` follow this pattern:
- `newXxxCmd()` function creates the Cobra command
- Command name constant in `constants.go`
- Aliases array in `constants.go`
- Tests in `*_test.go`

### API Client

- `pkg/gokeenrestapi/common.go` - Core client with auth
- `Common.Auth()` - Authenticate before operations
- `Common.ExecutePostParse()` - Execute CLI commands via RCI
- `Common.GetApiClient()` - Get configured HTTP client

### Configuration

- `pkg/config/main_config.go` - Main config struct (`GokeenapiConfig`)
- `config.LoadConfig(path)` - Load and validate config
- `config.Cfg` - Global config instance
- YAML expansion happens during load (bat-file, domain-file lists)

### Testing

- Use `SetupMockRouterForTest()` from `pkg/gokeenrestapi/mock_router.go`
- Property tests use `rapid` framework
- Test suites use `testify/suite`
