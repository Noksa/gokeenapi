---
inclusion: fileMatch
fileMatchPattern: ['docs/**', 'README*.md', 'SCHEDULER*.md', 'CONTRIBUTING.md', 'AGENTS.md', 'config_example.yaml', 'scheduler_example.yaml']
---

# Documentation Map

Two audiences. Do not mix them.

## Agent docs (source of truth for coding)

Read `AGENTS.md`, then `.kiro/steering/`. Those files describe how to change this repo. When a rule in steering disagrees with a comment in code, fix the stale side after checking `pkg/` / `cmd/`.

## Human docs (users and contributors)

| Path | Audience | Contents |
|------|----------|----------|
| `README.md` / `README_RU.md` | Users | Landing page, quick start, env vars, command tutorials |
| `docs/config-reference.md` / `docs/config-reference-ru.md` | Users | Field-by-field `config.yaml` schema |
| `config_example.yaml` | Users | Fully annotated router config |
| `SCHEDULER.md` / `SCHEDULER_RU.md` | Users | Scheduler concepts and `scheduler.yaml` |
| `scheduler_example.yaml` | Users | Annotated scheduler config |
| `CONTRIBUTING.md` | Contributors | Go version, make targets, Conventional Commits, PR flow |

`docs/` is **only** the human config reference (English + Russian). It is not agent instruction and not a dump of the whole product.

`custom/` is local operator data (real router configs, domain lists, helper scripts). Do not document it as public API.

## Keep in sync

| You changed | Also update |
|-------------|-------------|
| `pkg/config/main_config.go` fields or YAML tags | `docs/config-reference.md`, `docs/config-reference-ru.md`, `config_example.yaml`, and the config section of `product.md` |
| Scheduler schema (`pkg/config/scheduler_config.go`) | `SCHEDULER.md`, `SCHEDULER_RU.md`, `scheduler_example.yaml`, `product.md` |
| Command name, alias, or flag | `cmd/constants.go` (source), `product.md`, `README.md` / `README_RU.md` |
| Env var | `README.md` / `README_RU.md`, `product.md` |
| Make target or Go version | `tech.md`, `CONTRIBUTING.md`, `.github/workflows/` |
| User-visible behavior | README (and RU) plus the relevant `docs/` or `SCHEDULER.md` page |

English and Russian pairs must stay equivalent. Change both in the same edit.

Do not invent new top-level user manuals. Prefer extending README, `docs/config-reference.md`, or `SCHEDULER.md`.
