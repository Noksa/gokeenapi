# Contributing to gokeenapi

Thank you for your interest in contributing!

## Prerequisites

- **Go 1.26+** — see [go.mod](go.mod) for the exact requirement
- **Docker** — required for `make docker-build-test`

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
