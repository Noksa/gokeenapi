# Contributing to gokeenapi

Thank you for your interest in contributing!

## Prerequisites

- **Go 1.26+** — see [go.mod](go.mod) for the exact requirement
- **Docker** — required for `make docker-build-test`

## Project Structure

```
gokeenapi/
├── main.go                        # Entry point — calls cmd.NewRootCmd().Execute()
├── cmd/                           # CLI commands (Cobra)
│   ├── root.go                    # Root command + PersistentPreRunE (auth, config load)
│   ├── constants.go               # CmdXxx constants and AliasesXxx slices
│   ├── common.go                  # Shared validation, prompts, error helpers
│   ├── <command>.go               # One file per command (snake_case)
│   └── <command>_test.go          # Unit and property tests alongside source
├── pkg/
│   ├── config/                    # YAML config loading and expansion
│   ├── gokeenrestapi/             # Keenetic REST API singletons and mock router
│   └── gokeenrestapimodels/       # API request/response structs
├── internal/
│   ├── gokeenlog/                 # Structured logging (the only logging package allowed)
│   ├── gokeencache/               # In-memory API response cache
│   ├── gokeenspinner/             # CLI progress indicators
│   └── gokeenversion/             # Build-time version info
├── batfiles/                      # Example IP route bat-files (one CIDR per line)
├── docs/                          # Configuration reference documentation
├── scripts/                       # Build and lint helper scripts
├── config_example.yaml            # Annotated single-router config example
├── scheduler_example.yaml         # Annotated multi-router scheduler config example
├── Dockerfile                     # Multi-arch image build (amd64 + arm64)
└── Makefile                       # lint, build, test, coverage, docker targets
```

### Key layout rules

- `cmd/` — CLI layer only; all business logic lives in `pkg/` or `internal/`
- `pkg/` — externally importable packages
- `internal/` — packages that must not be imported outside this module
- Test files sit next to the source they test; integration tests (`docker_*_test.go`) live at the repository root
- Config examples (`config_example.yaml`, `scheduler_example.yaml`) must stay in sync with `pkg/config/` schema changes

## Commit Messages

This project follows [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<optional scope>): <short imperative summary>

[optional body]

[optional footer: Closes #N]
```

**Types:**

| Type | When to use |
|------|-------------|
| `feat` | New feature or command |
| `fix` | Bug fix |
| `refactor` | Code restructuring with no behaviour change |
| `docs` | Documentation only |
| `test` | Adding or correcting tests |
| `chore` | Maintenance: deps, CI, tooling |
| `ci` | CI/CD pipeline changes |

**Rules:**
- Subject line: ≤ 72 characters, present tense, no trailing period
- Scope is optional but encouraged for larger repos (e.g. `feat(dns-routing): …`)
- Examples:
  - `feat(scheduler): add retry on transient API errors`
  - `fix(add-routes): handle empty bat-file gracefully`
  - `docs: add project structure to CONTRIBUTING`
  - `test(config): add property tests for path resolution`
  - `chore: bump golangci-lint to v2.1`

## Build

```bash
make build              # lint + compile binary
make binaries VERSION=x.y.z   # cross-platform release binaries
```

## Test

```bash
make test               # full test suite (Ginkgo v2 + Gomega)
make test-short         # skip slow tests
make test-ci            # race detector + randomised order (CI-grade)
make test-coverage      # generate coverage.html
make test-focus FOCUS="pattern"   # run tests matching a pattern
```

Property-based tests use the `*_property_test.go` file suffix.

## Lint

```bash
make lint               # runs scripts/check.sh via golangci-lint
```

## Docker

```bash
make docker-build-test  # builds gokeenapi-test:local (no push)
```

## Pull Requests

1. Fork the repository and create a feature branch.
2. Write tests for new behaviour.
3. Ensure `make build` and `make test` pass locally.
4. Open a PR against `main` with a clear description of the change.

## Versioning

This project follows [Semantic Versioning](https://semver.org). Tags are created from `main` (`vX.Y.Z`).
