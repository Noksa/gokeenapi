---
inclusion: fileMatch
fileMatchPattern: '.github/workflows/*.yml'
---

# GitHub Actions Workflow Guidelines

## Workflow Architecture

This project uses `.github/workflows/ci.yml` with three parallel jobs for fast feedback:

| Job | Purpose | Commands | Duration |
|-----|---------|----------|----------|
| `build` | Compile verification | `go build -v ./...` | ~1-2 min |
| `lint` | Code quality checks | `golangci-lint-action` | ~5-15 min |
| `test` | Unit + integration tests | `make test` | ~3-5 min |

All jobs run in parallel on `ubuntu-latest` runners. The workflow triggers on push/PR to `main`, `master`, and `develop` branches.

## Required Action Versions

When modifying workflows, use these exact versions:

```yaml
- uses: actions/checkout@v5
- uses: actions/setup-go@v6
  with:
    go-version: '1.25'
    cache: true
- uses: golangci/golangci-lint-action@v9
  with:
    version: v2.6
    args: --timeout=15m
- uses: docker/setup-buildx-action@v3
```

Version pinning rules:
- Pin to major versions (`@v5`, `@v6`) for automatic security patches
- Use semantic Go version (`1.25` not `1.25.0`) to allow patch updates
- Specify golangci-lint version explicitly to ensure consistent linting

## Job Configuration Patterns

### Build Job
```yaml
- name: Build
  run: go build -v ./...
```
Verifies all packages compile without errors. Fast feedback on syntax/type errors.

### Lint Job
```yaml
- name: golangci-lint
  uses: golangci/golangci-lint-action@v9
  with:
    version: v2.6
    args: --timeout=15m
```
Runs comprehensive linting pipeline (see tech.md). Timeout set to 15m for large codebases. This action includes:
- `go fmt` - Code formatting
- `go vet` - Static analysis
- `golangci-lint` - Multiple linters (see `.golangci.yml`)

### Test Job
```yaml
- name: Setup Docker Buildx
  uses: docker/setup-buildx-action@v3

- name: Run tests
  run: make test
```
Runs all tests including Docker integration tests (`docker_*_test.go`). Docker Buildx required for integration tests that spin up containerized routers.

## Caching Strategy

The `cache: true` option in `setup-go` handles:
- Go module downloads (`go.sum` based)
- Build cache for faster compilation

Do NOT add manual caching steps like:
```yaml
# ❌ WRONG - redundant with setup-go cache
- name: Download dependencies
  run: go mod download
```

## Workflow Triggers

Standard trigger configuration:
```yaml
on:
  push:
    branches: [main, master, develop]
  pull_request:
    branches: [main, master, develop]
```

This ensures CI runs on:
- Direct pushes to protected branches
- All pull requests targeting these branches

## Integration with Project Standards

The CI workflow enforces the same standards as local development:

| Local Command | CI Equivalent | Purpose |
|---------------|---------------|---------|
| `make lint` | `lint` job | Code quality (fmt, vet, golangci-lint) |
| `make test` | `test` job | All tests including Docker integration |
| `go build ./...` | `build` job | Compilation verification |

Developers should run `make lint` and `make test` before pushing to catch issues early.

## When to Update Workflows

Update workflows when:

1. **Go version changes**: Update `go-version` in `setup-go` step
2. **New test requirements**: Add setup steps (databases, services, etc.)
3. **New linter rules**: Update golangci-lint version or `.golangci.yml`
4. **Build process changes**: Modify build job to match new Makefile targets
5. **New integration test dependencies**: Add service containers or setup actions

## Common Mistakes to Avoid

- ❌ Adding `go mod download` - setup-go caching handles this
- ❌ Hardcoding secrets - use `${{ secrets.SECRET_NAME }}`
- ❌ Running tests sequentially - keep jobs parallel for speed
- ❌ Duplicating lint checks - golangci-lint includes fmt/vet
- ❌ Using `latest` tags - pin to major versions for stability
- ❌ Skipping Docker setup - integration tests require it
- ❌ Setting short timeouts on lint - use 15m minimum

## Debugging Failed Workflows

When CI fails:

1. **Build failures**: Check for syntax errors, missing imports, type mismatches
2. **Lint failures**: Run `make lint` locally to see same errors
3. **Test failures**: Run `make test` locally, check Docker daemon is running
4. **Timeout issues**: Increase `timeout-minutes` on specific steps

The workflow logs show exact commands and output - use them to reproduce locally.

## Multi-Architecture Considerations

This project supports multi-arch Docker builds (`linux/amd64`, `linux/arm64`). If adding Docker build steps to CI:

```yaml
- name: Build Docker image
  uses: docker/build-push-action@v5
  with:
    platforms: linux/amd64,linux/arm64
    push: false  # Only push on release workflow
```

Keep build/push separate from CI to avoid unnecessary registry operations on every commit.
