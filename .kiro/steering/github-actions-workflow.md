---
inclusion: fileMatch
fileMatchPattern: '.github/workflows/*.yml'
---

# GitHub Actions Workflow Guidelines

Release tagging and notes: `release-procedure.md`.
Go version and local make targets: `tech.md`.

## Workflows

| File | Trigger | Role |
|------|---------|------|
| `.github/workflows/ci.yml` | push/PR to `main`, `master`, `develop` | Parallel `build`, `lint`, `test`, `security` |
| `.github/workflows/release.yml` | push of `v*` tags | Cross-compile 6 binaries, GitHub release, multi-arch Docker push |

## CI jobs (`ci.yml`)

| Job | Purpose | Command | Typical duration |
|-----|---------|---------|------------------|
| `build` | Compile verification | `go build -v ./...` | ~1–2 min |
| `lint` | Code quality | `golangci-lint-action` | ~5–15 min |
| `test` | Unit + Docker integration | `make test-ci` | ~3–5 min |
| `security` | Scan the built Linux image for critical OS and Go vulnerabilities | Docker Buildx + Trivy | Varies with the vulnerability DB cache |

All four run in parallel on `ubuntu-latest`. The security job builds a local
`linux/amd64` image and fails on any `CRITICAL` vulnerability in the final image.
`HIGH` findings are outside this release gate. Trivy scans both Alpine packages
and Go modules embedded in the binary.

## Required Action Versions (CI)

Match what is already in `.github/workflows/ci.yml` unless you are intentionally bumping:

```yaml
- uses: actions/checkout@v5
- uses: actions/setup-go@v6
  with:
    go-version: '1.26'
    cache: true
- uses: golangci/golangci-lint-action@v9
  with:
    version: v2.12
    args: --timeout=15m
- uses: docker/setup-buildx-action@v3
```

Pinning:

- Actions: major version (`@v5`, `@v6`) for security patches
- Go: semantic `1.26` (not `1.26.0`) so patch releases flow in
- golangci-lint: pin the exact version used in CI (`v2.12`)

`release.yml` currently uses `actions/checkout@v6` and `docker/build-push-action@v6`. Keep each workflow internally consistent; do not silently mix majors inside one file.

## Job Patterns

### Build

```yaml
- name: Build
  run: go build -v ./...
```

### Lint

```yaml
- name: golangci-lint
  uses: golangci/golangci-lint-action@v9
  with:
    version: v2.12
    args: --timeout=15m
```

This action covers fmt/vet/linters from `.golangci.yml`. Local equivalent is `make lint` (`scripts/check.sh`).

### Test

```yaml
- name: Setup Docker Buildx
  uses: docker/setup-buildx-action@v3

- name: Run tests
  run: make test-ci
```

`make test-ci` (not `make test`) is what CI runs: race, randomized order, coverage, JSON report. Docker Buildx is required for `docker_*_test.go`.

## Caching

`cache: true` on `setup-go` already caches modules and build artifacts.

Do **not** add:

```yaml
# wrong — redundant with setup-go cache
- name: Download dependencies
  run: go mod download
```

## Triggers (CI)

```yaml
on:
  push:
    branches: [main, master, develop]
  pull_request:
    branches: [main, master, develop]
```

## Local vs CI

| Local | CI | Purpose |
|-------|-----|---------|
| `make lint` | `lint` job | fmt, vet, golangci-lint |
| `make test-ci` | `test` job | Race + randomized tests, including Docker integration |
| `go build ./...` | `build` job | Compile |

Run `make lint` and `make test` (or `make test-ci`) before pushing.

## When to Update Workflows

1. Go version in `go.mod` / Dockerfile changes → update `go-version` in **both** `ci.yml` and `release.yml`
2. New test services → add setup steps
3. Linter version or `.golangci.yml` changes
4. Makefile test/lint entrypoints change
5. Integration tests need extra Docker/QEMU setup

## Common Mistakes

- Adding `go mod download` (setup-go cache handles it)
- Hardcoding secrets — use `${{ secrets.SECRET_NAME }}` or `${{ vars.NAME }}`
- Serializing the three CI jobs
- Duplicating fmt/vet next to golangci-lint
- Using `latest` tags
- Skipping Docker Buildx on the test job
- Lint timeout under 15m
- Documenting `make test` as the CI command — CI uses `make test-ci`

## Debugging

1. Build: syntax, imports, types
2. Lint: reproduce with `make lint`
3. Test: `make test-ci` locally with Docker running
4. Timeouts: raise `timeout-minutes` on the step

## Release workflow notes

`release.yml` builds `linux/darwin/windows` × `amd64/arm64` via `make binaries TARGET=os/arch`, then:

- GitHub release with `generated/*.tar.gz` (`softprops/action-gh-release@v2`)
- Docker Hub image `noksa/gokeenapi` for `linux/amd64,linux/arm64` (`docker/build-push-action@v6`)

Do not fold image push into CI. Tags (`stable`, `X.Y.Z`, `X.Y`, prerelease behavior): `release-procedure.md`.
