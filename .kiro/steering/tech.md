---
inclusion: always
---

# Tech Stack

## Language & Runtime

- Go 1.25+ with standard project layout (`cmd/`, `pkg/`, `internal/`)
- Use `go.mod` as source of truth for version and dependencies

## Key Dependencies

| Package | Import Path | Purpose |
|---------|-------------|---------|
| cobra | `github.com/spf13/cobra` | CLI framework |
| resty | `github.com/go-resty/resty/v2` | HTTP client |
| yaml.v3 | `gopkg.in/yaml.v3` | YAML parsing |
| testify | `github.com/stretchr/testify` | Test assertions/suites |
| rapid | `pgregory.net/rapid` | Property-based testing |
| multierr | `go.uber.org/multierr` | Error aggregation |
| go-version | `github.com/hashicorp/go-version` | Semantic version parsing |

## Build Commands

```bash
make lint           # go fmt, go vet, modernize, golangci-lint
make test           # Run all tests
make build          # Lint + Docker build
make test-coverage  # Tests with coverage report
```

## Code Style Rules

- Run `make lint` before committing - it executes `scripts/check.sh`
- Linting includes: `go mod tidy` → `go fmt` → `go vet` → `modernize -fix` → `golangci-lint`
- Use `multierr` for aggregating multiple errors
- Prefer `resty` over `net/http` for API calls

## Testing Conventions

| Pattern | Framework | Location |
|---------|-----------|----------|
| `*_test.go` | Standard Go + testify | Same directory as source |
| `*_property_test.go` | rapid | Same directory as source |
| Test suites | `testify/suite` | Grouped related tests |

- Use unified mock router: `pkg/gokeenrestapi/mock_router.go`
- Call `SetupMockRouterForTest()` for router API tests
- Property tests validate invariants across generated inputs

## Configuration

- YAML config files parsed via `pkg/config/`
- Environment variables:
  - `GOKEENAPI_KEENETIC_LOGIN` - Router username
  - `GOKEENAPI_KEENETIC_PASSWORD` - Router password
  - `GOKEENAPI_CONFIG` - Config file path

## Docker

- Multi-arch: `linux/amd64`, `linux/arm64`
- Image: `noksa/gokeenapi:stable`
- Uses BuildKit via `docker buildx build`
