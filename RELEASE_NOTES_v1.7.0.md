# Release Notes — v1.7.0

> Changes since [v1.6.1](https://github.com/Noksa/gokeenapi/releases/tag/v1.6.1)

---

## Security Fixes

- **SHA-256 cache filenames** — cache filenames now use SHA-256 instead of MD5, eliminating a weak-hash collision risk.
- **Restricted file permissions** — cache directories are now created with `0700` (owner-only), closing a world-readable data exposure.
- **Non-root container** — the Docker image now runs as a dedicated `gokeenapi` user instead of root; temp directories are created with `0777` to preserve access for the unprivileged user.
- **World-readable config warning** — the application now prints a warning on startup when the config file is readable by group or others.

---

## Bug Fixes

### Concurrency / Race Conditions
- `fix(api)` — protect shared `restyClient` singleton and `cleanedOldCacheFiles` with sync primitives, eliminating data races under concurrent requests.
- `fix(api)` — restore the `restyClient` singleton race fix that was accidentally reverted in an earlier PR.

### Scheduler
- `fix(scheduler)` — aggregate errors from parallel goroutines instead of losing all but the last one.
- `fix(scheduler)` — use `select` on queue send so a context cancellation is respected immediately instead of blocking.

### Auth / API Client
- `fix(auth)` — break the infinite retry loop in `authRetryMiddleware` when a 401 persists (was retrying forever on bad credentials).
- `fix(auth)` — return the error from `GetApiClient` instead of panicking.

### Cache
- `fix(cache)` — ignore `os.ErrNotExist` during cache cleanup `WalkDir` (harmless race between listing and deletion).

### REST API
- `fix(gokeenrestapi)` — restore `ErrNotExist` guards, HTTP status check, and `unmarshal continue` that were lost in a rebase.
- `fix(restapi)` — skip the current batch on unmarshal error in `ExecutePostParse` instead of aborting the whole operation.
- `fix(ip)` — check the HTTP status code in `AddRoutesFromBatUrl` before processing the response body.

### Config / Runtime
- `fix(common)` — allocate a new slice in `EnsureSaveConfigAtEnd` to prevent backing-array mutation when multiple callers append to the same slice.
- `fix(main)` — store and `defer cancel` from `signal.NotifyContext` to avoid a context leak on shutdown.

---

## New Features

- **`tls_skip_verify` option** (`feat(config)`) — routers now accept a `tls_skip_verify` field to disable TLS certificate verification for self-signed or internal certificates.
- **Common groups in `dns.routes.groups`** (`feat`) — DNS route groups can now reference shared/common groups, reducing repetition across multiple route definitions.

---

## Documentation

- Added **CONTRIBUTING.md** with build, test, and pull-request instructions.
- Documented all four **environment variables** in README and README_RU.
- Added **missing command aliases** to README and README_RU.
- Added the **project logo** to README.
- Rewrote and restructured README for clarity.

---

## CI / Testing Improvements

- **Migrated tests to Ginkgo** — the full test suite now uses Ginkgo/Gomega BDD-style specs for consistent structure and better failure output.
- Added unit tests for `EnsureSaveConfigAtEnd` and `SaveConfig`.
- Added tests for the 300-domain router limit and duplicate-domain warning.
- Expanded test coverage for DNS, routes, and scheduler commands.
- Fixed Go version mismatch in CI (`go-version` bumped from 1.25 → 1.26 to match `go.mod`).
- Replaced the `ginkgo` binary invocation in Makefile with `go run` so the version always tracks `go.mod`.
- Refactored test infrastructure and CI workflow for parallel execution and coverage reporting.

---

## Breaking Changes

None.
