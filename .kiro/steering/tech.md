# Tech Stack

## Language & Runtime

- **Go 1.25+** (see `go.mod`)
- Standard Go project layout with `cmd/`, `pkg/`, `internal/`

## Key Dependencies

- **cobra** (`github.com/spf13/cobra`) - CLI framework
- **resty** (`github.com/go-resty/resty/v2`) - HTTP client for REST API
- **yaml.v3** (`gopkg.in/yaml.v3`) - YAML configuration parsing
- **testify** (`github.com/stretchr/testify`) - Testing assertions and suites
- **rapid** (`pgregory.net/rapid`) - Property-based testing framework
- **spinner** (`github.com/briandowns/spinner`) - CLI progress indicators
- **color** (`github.com/fatih/color`) - Colored terminal output
- **go-cache** (`github.com/patrickmn/go-cache`) - In-memory caching
- **multierr** (`go.uber.org/multierr`) - Error aggregation

## Build & Development

### Common Commands

```bash
make lint      # Run linting (go fmt, go vet, golangci-lint)
make test      # Run all tests
make build     # Lint + build Docker image
make test-coverage  # Tests with coverage report
```

### Linting Pipeline (`scripts/check.sh`)

1. `go mod tidy`
2. `go fmt ./...`
3. `go vet ./...`
4. `modernize -fix ./...`
5. `golangci-lint run --timeout 15m`

### Docker

- Multi-arch builds (linux/amd64, linux/arm64)
- Image: `noksa/gokeenapi:stable`
- Build: `docker buildx build` with BuildKit

## Testing

- Standard Go tests (`*_test.go`)
- Property-based tests (`*_property_test.go`) using `rapid`
- Test suites using `testify/suite`
- Unified mock router in `pkg/gokeenrestapi/mock_router.go`

## Configuration

- YAML-based config files
- Environment variables: `GOKEENAPI_KEENETIC_LOGIN`, `GOKEENAPI_KEENETIC_PASSWORD`, `GOKEENAPI_CONFIG`
