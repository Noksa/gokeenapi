---
inclusion: fileMatch
fileMatchPattern: '.github/workflows/*.yml'
---

# GitHub Actions Workflow Guidelines

## Project Workflow Structure

This project uses `.github/workflows/ci.yml` with three parallel jobs:

| Job | Purpose | Key Steps |
|-----|---------|-----------|
| `build` | Compile verification | `go build -v ./...` |
| `lint` | Code quality | `golangci-lint-action` with 15m timeout |
| `test` | Unit + integration tests | `make test` (includes Docker tests) |

## Required Patterns

When modifying workflows:

- **Go setup**: `actions/setup-go@v6` with `go-version: '1.25'` and `cache: true`
- **Linting**: `golangci/golangci-lint-action@v9` with `version: v2.6`
- **Checkout**: `actions/checkout@v5`
- **Docker**: `docker/setup-buildx-action@v3` (required for integration tests)
- **Triggers**: Push/PR on `main`, `master`, `develop` branches

## Best Practices

- Pin actions to major versions (`@v5`, `@v6`) for stability with security updates
- Use semantic Go version (`1.25` not `1.25.0`) for patch flexibility
- Enable `cache: true` on setup-go - avoids redundant `go mod download`
- Set `timeout-minutes` on long-running steps (lint uses 15m)
- Keep jobs parallel and independent for faster CI

## Post-Change Review

After code changes, check if workflows need updates for:

- New dependencies requiring additional setup steps
- New test types or coverage requirements
- Build process changes (Makefile targets, Docker requirements)

## Do NOT

- Add redundant `go mod download` steps (caching handles this)
- Hardcode credentials (use GitHub Secrets)
- Create overly complex matrix builds unless testing multiple Go versions
- Duplicate lint checks already in `golangci-lint` (fmt, vet included)
