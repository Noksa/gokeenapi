# AGENTS.md

Instructions for coding agents working in this repository.

## Source of truth

Read **`.kiro/steering/`** before changing code or docs. Those files are the project rules. This file only points at them.

| File | Read when |
|------|-----------|
| [product.md](.kiro/steering/product.md) | Always. CLI purpose, commands, config, contracts. |
| [structure.md](.kiro/steering/structure.md) | Always. Layout, naming, adding a command, API usage. |
| [tech.md](.kiro/steering/tech.md) | Always. Go version, deps, make targets, anti-patterns. |
| [mock-testing-guidelines.md](.kiro/steering/mock-testing-guidelines.md) | Tests or `pkg/gokeenrestapi/mock_*.go`. |
| [github-actions-workflow.md](.kiro/steering/github-actions-workflow.md) | `.github/workflows/`. |
| [release-procedure.md](.kiro/steering/release-procedure.md) | Cutting a release. |
| [documentation.md](.kiro/steering/documentation.md) | README, `docs/`, SCHEDULER, CONTRIBUTING, example YAMLs. |

If steering disagrees with the code, trust `cmd/`, `pkg/`, `go.mod`, and `.github/workflows/`, then fix the stale steering file in the same change.

## Not agent instructions

`docs/`, `README.md`, `README_RU.md`, `SCHEDULER.md`, `SCHEDULER_RU.md`, and `CONTRIBUTING.md` are **human-facing**. Use them to understand the product, not as coding rules. Keep them in sync when user-visible behavior or config schema changes (see `documentation.md`).

## Default checks

```bash
make lint
make test
```

Full target list: `tech.md`.
