---
inclusion: always
---

# Tech Stack & Development Rules

## Language & Runtime

- Go 1.25+ with standard project layout (`cmd/`, `pkg/`, `internal/`)
- `go.mod` is the source of truth for version and dependencies
- NEVER modify `go.mod` manually - use `go get` or `go mod tidy`

## Critical Dependencies

| Package | Import Path | When to Use |
|---------|-------------|-------------|
| cobra | `github.com/spf13/cobra` | All CLI commands - use `RunE` not `Run` |
| resty | `github.com/go-resty/resty/v2` | HTTP/API calls - NEVER use `net/http` directly |
| yaml.v3 | `gopkg.in/yaml.v3` | YAML parsing - use `yaml.Unmarshal()` |
| testify | `github.com/stretchr/testify` | Assertions: `assert.NoError()`, `assert.Equal()` |
| rapid | `pgregory.net/rapid` | Property tests in `*_property_test.go` files |
| multierr | `go.uber.org/multierr` | Aggregating errors: `multierr.Append(errs, err)` |
| go-version | `github.com/hashicorp/go-version` | Version comparison - use `version.NewVersion()` |

## Mandatory Build & Test Commands

Before any commit or code change:
```bash
make lint           # REQUIRED before committing
make test           # REQUIRED - all tests must pass
make test-coverage  # Use when adding new functionality
```

Linting pipeline (executed by `make lint`):
1. `go mod tidy` - Clean dependencies
2. `go fmt` - Format code
3. `go vet` - Static analysis
4. `modernize -fix` - Update deprecated patterns
5. `golangci-lint` - Comprehensive linting

## Code Style Requirements

### Error Handling
- ALWAYS use `multierr.Append()` when processing lists/loops
- Return errors from `RunE` functions - NEVER call `os.Exit()` in commands
- Validate inputs before API calls - fail fast with clear messages

### API Client Usage
- ALWAYS use `resty` for HTTP requests
- NEVER use `net/http` directly
- Use singleton instances from `pkg/gokeenrestapi/` package

### Logging
- ALWAYS use `gokeenlog` package for output
- NEVER use `fmt.Println()`, `log.Println()`, or `print()`
- Available methods: `Info()`, `Debug()`, `Error()`, `Infof()`, `InfoSubStepf()`

## Testing Requirements

### File Naming
- Unit tests: `*_test.go` (same directory as source)
- Property tests: `*_property_test.go` (same directory as source)
- Integration tests: `docker_*_test.go` (repository root)

### Test Setup Pattern
```go
func TestMyFunction(t *testing.T) {
    cleanup := gokeenrestapi.SetupMockRouterForTest()
    defer cleanup()
    // Test code here
}
```

### Property Test Pattern
```go
func TestProperty_MyInvariant(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        input := rapid.String().Draw(t, "input")
        result := MyFunction(input)
        // Assert invariant holds
    })
}
```

### Test Suite Pattern
```go
type MyTestSuite struct {
    suite.Suite
}

func (s *MyTestSuite) TestSomething() {
    s.NoError(err)
    s.Equal(expected, actual)
}

func TestMyTestSuite(t *testing.T) {
    suite.Run(t, new(MyTestSuite))
}
```

## Configuration Rules

- Config loaded automatically in `root.go` PersistentPreRunE
- Access via global `config.Cfg` variable
- Environment variables override config file values:
  - `GOKEENAPI_KEENETIC_LOGIN` - Router username
  - `GOKEENAPI_KEENETIC_PASSWORD` - Router password
  - `GOKEENAPI_CONFIG` - Config file path
- YAML files support expansion - paths resolved relative to referencing file

## Docker Build

- Multi-arch support: `linux/amd64`, `linux/arm64`
- Image: `noksa/gokeenapi:stable`
- Uses BuildKit: `docker buildx build`
- Dockerfile at repository root

## Common Mistakes to Avoid

- ❌ Using `fmt.Println()` instead of `gokeenlog`
- ❌ Using `net/http` instead of `resty`
- ❌ Calling `os.Exit()` in command functions
- ❌ Not using `multierr` for error aggregation
- ❌ Forgetting to run `make lint` before committing
- ❌ Using `Run` instead of `RunE` in cobra commands
- ❌ Manually editing `go.mod` instead of using `go get`
- ❌ Not calling `SetupMockRouterForTest()` in API tests
