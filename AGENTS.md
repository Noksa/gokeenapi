# AGENTS.md

Compact rules for coding agents. Deep project rules live in **`.kiro/steering/`** — read the matching file before changing that area. If steering disagrees with the code, trust `cmd/`, `pkg/`, `go.mod`, and `.github/workflows/`, then fix the stale steering file in the same change.

| File | Read when |
|------|-----------|
| [product.md](.kiro/steering/product.md) | Commands, config schema, behavioral contracts |
| [structure.md](.kiro/steering/structure.md) | Layout, adding a command, API usage |
| [tech.md](.kiro/steering/tech.md) | Go version, make targets, anti-patterns |
| [mock-testing-guidelines.md](.kiro/steering/mock-testing-guidelines.md) | Tests or `pkg/gokeenrestapi/mock_*.go` |
| [github-actions-workflow.md](.kiro/steering/github-actions-workflow.md) | `.github/workflows/` |
| [release-procedure.md](.kiro/steering/release-procedure.md) | Cutting a release |
| [documentation.md](.kiro/steering/documentation.md) | README, `docs/`, SCHEDULER, CONTRIBUTING, example YAMLs |

`docs/`, `README*.md`, `SCHEDULER*.md`, and `CONTRIBUTING.md` are human-facing, not coding rules. **Mandatory:** every user-visible behavior change or fix must update both `README.md` and `README_RU.md` in the same change. Update the relevant EN+RU `SCHEDULER*.md`, `docs/*.md`, and tracked `*_example.yaml` files whenever that behavior or schema appears there. Keep each language pair equivalent.

## Checks

```bash
make lint                    # mutates the tree: tidy, fmt, modernize -fix, then golangci-lint
make test                    # Ginkgo via `go run` from go.mod — do not put `go test` in Make/CI
make test-focus FOCUS="…"    # Ginkgo --focus
make test-ci                 # what CI runs: race + randomized + cover; needs Docker
```

- Go **1.26** (`go.mod`). Do not edit `go.mod` by hand — `go get` / `go mod tidy`.
- `make lint` needs a local `golangci-lint` (CI pins **v2.12**). First `make` curl-fetches gitignored `.cyber.sh`.
- **`make build` is lint + multi-arch `docker buildx --push` to `noksa/gokeenapi`. Do not run it for a local binary.** Use `go build .` or `make binaries`. Local image only: `make docker-build-test` → `gokeenapi-test:local`.

`docker_*_test.go` at repo root skip if the daemon is down. Temp dirs are created under `$HOME` (Colima). CI test job requires Docker Buildx.

## Architecture agents guess wrong

- `cmd/` is Cobra only (`RunE`, never `Run` or `os.Exit`). Logic lives in `pkg/` / `internal/`.
- API clients are package-level singletons (`gokeenrestapi.Common`, `.Ip`, `.DnsRouting`, `.AwgConf`, `.Checks`, `.Interface`, `.Components`). Never instantiate. Types: `pkg/gokeenrestapimodels`. Production HTTP: resty, not `net/http`.
- `config.Cfg` is loaded once in `PersistentPreRunE`. Touch it only inside `RunE`, never in constructors. Commands must not call `Auth()` (automatic). Skip load/auth: `completion`, `help`, `scheduler`, `version`.
- Scheduler uses `config.LoadSchedulerConfig` and re-execs `os.Executable()` per command with `--config <router.yaml>`.
- `update-awg` is implemented in `cmd/configure_awg.go`. Do not add `update_awg.go`.
- Logging: only `internal/gokeenlog`. Exception: `confirmAction` in `cmd/common.go`.
- Validate with `Checks` before other API calls. Aggregate loop errors with `multierr.Append`. Mutating RCI batches end with `Common.EnsureSaveConfigAtEnd`.
- YAML: `gopkg.in/yaml.v3` in production (do not switch to `go.yaml.in/yaml/v3`).
- Bat-files: `route add <ip> MASK <mask> <gw>` — CIDR-only lines are rejected. Interface IDs are case-sensitive (`Wireguard0`).
- New command: `CmdXxx` + `AliasesXxx` in `cmd/constants.go` → `cmd/<snake>.go` → `rootCmd.AddCommand` in `NewRootCmd()`.

## Tests

Ginkgo v2 + Gomega. Never testify. Every package has `suite_test.go`. Property tests: `*_property_test.go` with `rapid.Check(GinkgoT(), …)` inside `It`. Use `Eventually(…).WithTimeout(…).WithPolling(…)`, not positional args.

One mock: `pkg/gokeenrestapi/mock_router.go` plus `mock_router_handlers_*.go`. Extend it; never a second mock. Call `SetupMockRouterForTest()`. In `cmd/`, `setupMockRouter` already calls `Auth()` — do not Auth again in `BeforeEach`.

## Git / docs

- `custom/` is gitignored (local operator data, not public API). `config.yaml` and `config_*.yaml` are ignored; `config_example.yaml` is tracked. `generated/` is release output.
- Commits: Conventional Commits (`feat`/`fix`/`refactor`/`docs`/`test`/`chore`/`ci`).
- Command/alias/flag, env var, or `pkg/config` schema changes: update `product.md` plus the matching human docs (both languages).
- Before finishing any user-visible change, verify that `README.md` and `README_RU.md` describe it and that relevant scheduler guides, config references, and example YAMLs match.
