---
inclusion: always
---

# Project Overview

This is **gokeenapi**, a Go CLI tool for automating Keenetic (Netcraze) router management. It handles routes, DNS records, WireGuard connections, and scheduled tasks.

## Initial Context Gathering

When starting work on this project:

1. Read `README.md` for feature overview and command documentation
1. Read `SCHEDULER.md` for Scheduler feature overview and related commands
3. Check `Makefile` for available build, test, and lint targets
4. Examine `go.mod` for Go version and dependencies
5. Review `main.go` for application entry point
6. Explore `cmd/` for command implementations (uses Cobra framework)
7. Explore `pkg/` for core business logic and API clients
8. Explore `internal/` for internal utilities (logging, caching, spinner)

## Architecture Patterns

- **Command structure**: Uses [Cobra](https://github.com/spf13/cobra) CLI framework in `cmd/` directory
- **Configuration**: YAML-based config files parsed in `pkg/config/`
- **API client**: REST API client for Keenetic routers in `pkg/gokeenrestapi/`
- **Models**: API response models in `pkg/gokeenrestapimodels/`
- **Internal utilities**: Logging, caching, and UI components in `internal/`

## Code Conventions

- Follow existing patterns - consistency over novelty
- Use the project's internal utilities (`gokeenlog`, `gokeencache`, `gokeenspinner`)
- Match error handling patterns found in existing commands
- Maintain command aliases (e.g., `show-interfaces` has aliases `showinterfaces`, `showifaces`, `si`) with constants

## Build & Validation

- Run `make lint` to check code quality (runs `scripts/check.sh`)
- Run `make test` to execute all tests
- Run `make build` to build the binary (includes linting)
- Tests use standard Go testing with property-based tests using `rapid` framework
- Property tests are named `*_property_test.go`

## Key Features to Understand

- **Scheduler**: Automated task execution at intervals or specific times (`cmd/scheduler.go`)
- **Bat-file expansion**: YAML files can reference other YAML files containing bat-file lists
- **Multi-router support is not implemented**: Single config CAN'T manage multiple routers
- **Environment variables**: Credentials can be stored as `GOKEENAPI_KEENETIC_LOGIN` and `GOKEENAPI_KEENETIC_PASSWORD`

## Testing Philosophy

- Maintain existing tests when modifying code
- Do not add new tests unless explicitly requested
- Test files follow Go conventions (`*_test.go`)

## Common Workflows

- Adding a new command: Create in `cmd/`, follow existing command patterns, register in `cmd/root.go`
- Modifying API client: Update `pkg/gokeenrestapi/`, ensure models in `pkg/gokeenrestapimodels/` match
- Configuration changes: Update `pkg/config/`, maintain backward compatibility
- Scheduler tasks: Modify `cmd/scheduler.go` and related test files