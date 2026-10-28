---
inclusion: always
---

# Product Overview

`gokeenapi` is a Go CLI for automating Keenetic (Netcraze) routers over the REST/RCI API. It manages static IP routes, DNS records, DNS-based routing policies, WireGuard/AmneziaWG VPN, known hosts, and scheduled multi-router jobs.

Human-facing docs live in `README.md`, `docs/`, and `SCHEDULER.md`. Agent rules live in `.kiro/steering/` (see `AGENTS.md`).

When adding a command, follow the checklist in `structure.md`.

## Commands Reference

Names and aliases are defined in `cmd/constants.go`. Always add both `CmdXxx` (kebab-case constant) and `AliasesXxx` (compact strings, typically no hyphens — extra hyphenated aliases exist for a few commands).

| Command | Aliases | Purpose |
|---|---|---|
| `add-routes` | `addroutes`, `ar` | Add static IP routes from bat-files or URLs |
| `delete-routes` | `deleteroutes`, `dr` | Delete static IP routes listed in config |
| `delete-all-routes` | `deleteallroutes`, `dar` | Remove all static routes in a single request |
| `add-dns-records` | `adddnsrecords`, `adr` | Add static DNS A/AAAA records |
| `delete-dns-records` | `deletednsrecords`, `ddr` | Delete static DNS records |
| `add-dns-routing` | `adddnsrouting`, `adnsr`, `adddnsroutes`, `add-dns-routes` | Add policy-based DNS routing by domain |
| `delete-dns-routing` | `deletednsrouting`, `ddnsr`, `deletednsroutes`, `delete-dns-routes` | Delete DNS-routing groups from config |
| `delete-all-dns-routing` | `deletealldnsrouting`, `dadnsr` | Remove every DNS-routing group on the router; prompts unless `--force` |
| `add-awg` | `addawg`, `aawg` | Add WireGuard/AWG VPN from a `.conf` file (`--conf-file`, optional `--name`) |
| `update-awg` | `updateawg`, `uawg` | Update an existing AWG interface (`--conf-file`, `--interface-id`, optional `--dry-run`) |
| `delete-known-hosts` | `deleteknownhosts`, `dkh` | Remove known host entries (regex supported) |
| `exec` | `e` | Execute arbitrary Keenetic RCI parse commands |
| `exporter` | `metrics` | Serve cached Prometheus metrics from multiple routers |
| `scheduler` | `schedule`, `sched` | Run automated scheduled tasks from a scheduler YAML |
| `show-interfaces` | `showinterfaces`, `si`, `showinterface`, `show-interface` | List router interfaces |
| `version` | — | Print build version info |

`PersistentPreRunE` in `cmd/root.go` skips config load and auth for `completion`, `help`, `scheduler`, `exporter`, and `version`.

Route and DNS-routing commands take the interface from the YAML config, not from a CLI argument. `add-awg` / `update-awg` take `--conf-file` (and `update-awg` requires `--interface-id`).

`update-awg` is implemented in `cmd/configure_awg.go` (historical filename). Do not add a second `update_awg.go`.

## API Singletons

Never instantiate — use the package-level values in `pkg/gokeenrestapi/`. Validate with `Checks` before other API calls.

| Singleton | Responsibility |
|---|---|
| `gokeenrestapi.Common` | Auth, raw RCI (`ExecutePostParse`, `ExecuteGetSubPath`, `ExecutePostSubPath`), config save |
| `gokeenrestapi.Ip` | Routes, DNS records, known hosts (`AddRoutesFromBatFile`, `AddRoutesFromBatUrl`, `DeleteRoutes`, `DeleteAllRoutes`, `AddDnsRecords`, `DeleteDnsRecords`, `DeleteKnownHosts`) |
| `gokeenrestapi.DnsRouting` | DNS-routing groups (`AddDnsRoutingGroups`, `DeleteDnsRoutingGroups`, `DeleteAllDnsRoutingGroups`, `LoadDomainsFromFile`, `LoadDomainsFromURL`, `CheckDnsRoutingSupport`) |
| `gokeenrestapi.AwgConf` | AWG/WireGuard (`AddInterface`, `ConfigureOrUpdateInterface`, `DiffUpdate`, `PlanUpdate`) |
| `gokeenrestapi.Checks` | Input validation (`CheckInterfaceId`, `CheckInterfaceExists`, `CheckComponentInstalled`, `CheckAWGInterfaceExistsFromConfFile`) |
| `gokeenrestapi.Interface` | Interface listing, wait-until-up, global IP |
| `gokeenrestapi.Components` | Installed component list |

Auth is automatic in `PersistentPreRunE`. Commands must never call `Auth()` themselves.

## Router YAML (`config.yaml`)

Schema: `pkg/config/main_config.go`. Annotated example: `config_example.yaml`. Field-by-field user reference: `docs/config-reference.md`.

Required at runtime (`cmd/common.go` `checkRequiredFields`):

- `keenetic.url`
- `keenetic.login` (or `GOKEENAPI_KEENETIC_LOGIN`)
- `keenetic.password` (or `GOKEENAPI_KEENETIC_PASSWORD`)

YAML credentials are allowed; env vars override YAML and are the recommended way to store secrets. A world-readable config file prints a warning.

| Env var | Role |
|---|---|
| `GOKEENAPI_CONFIG` | Config path (alternative to `--config`). The official image sets `/etc/gokeenapi/config.yaml`. An explicit `--config` wins. |
| `GOKEENAPI_KEENETIC_LOGIN` | Overrides `keenetic.login` |
| `GOKEENAPI_KEENETIC_PASSWORD` | Overrides `keenetic.password` |
| `GOKEENAPI_INSIDE_DOCKER` | Forces `dataDir` to `/etc/gokeenapi` (set in the official image). The session cache directory is `/etc/gokeenapi/.gokeenapi` and must be a directory, not a file. |

Other router-config fields: `keenetic.tls_skip_verify`, `keenetic.timeout` (default `30s`), `dataDir`, `routes[]`, `dns.records[]`, `dns.routes.groups[]`, `logs.debug`, `cache.urlTtl`.

Configs support **expansion**: a `.yaml`/`.yml` path in `bat-file`, `bat-url`, `domain-file`, or `domain-url` is replaced by the list of the same name inside that file. Paths resolve relative to the referencing file. A YAML referenced from `bat-file` contributes only `bat-file` entries (same for the other keys).

DNS-routing `groups` entries may be:

- a group object (`name`, `interfaceId`, `domain-file` / `domain-url`)
- a string path to a YAML file with a `groups:` list
- `{file: common.yaml, interfaceId: Wireguard1}` — import that file and override `interfaceId` on every imported group

## Scheduler YAML (`scheduler.yaml`)

Schema: `pkg/config/scheduler_config.go`. User guide: `SCHEDULER.md`. Example: `scheduler_example.yaml`.

Each task has `name`, `commands`, `configs`, and either `interval` (e.g. `3h`) or `times` (e.g. `["06:00"]`). Optional: `retry`, `retryDelay`, `strategy` (`sequential` default, or `parallel` — constants `StrategySequential` / `StrategyParallel` in `cmd/constants.go`).

A command entry is either a string (`add-routes`) or `{name: delete-all-dns-routing, args: ["--force"]}`.

## Input File Formats

**Bat-files** (`batfiles/`) — Windows-style route commands, parsed by `routeRegex` in `pkg/gokeenrestapi/ip.go`:

```
route add 11.11.11.11 MASK 255.255.255.255 0.0.0.0
```

`#`-prefixed lines and blank lines are skipped. CIDR-only lines are **not** accepted.

**Domain-files** (`custom/domains/`) — DNS-routing sources:

- One domain per line: `example.com`, `*.example.com`
- `#`-prefixed lines and blank lines are ignored
- Wildcards and IDNA-encoded internationalized domains are supported

## Interface Names

Interface IDs are case-sensitive: `Wireguard0`, `Wireguard1`, `ISP`, `HomeNetwork`. Use `show-interfaces` to discover them.

## Behavioral Contracts

- **Idempotency**: add/delete is safe to repeat — no duplicates on add, no errors on missing items during delete
- **Fail-fast validation**: `gokeenrestapi.Checks` before other API calls
- **Error aggregation**: `multierr.Append()` when iterating lists — collect all errors
- **DNS-routing firmware**: Keenetic OS 5.0.1+
- **DNS-routing group size**: after sorting and deduplication, `add-dns-routing` splits each source into groups of at most 300 domains. The configured name holds the first chunk; later chunks use `name-2`, `name-3`, etc. Repeated runs reconcile each chunk and remove obsolete overflow groups. These suffix names are reserved for the source group.
- **DNS-routing source reconciliation**: every valid nonempty source is reconciled regardless of how much its domain count changed. Empty or unreadable sources do not remove existing router state.
- **DNS-routing sources and safety**: normalize domains to lowercase ASCII with IDNA and IPv4 subnets to their network address; skip unsupported `regexp:`, `keyword:`, and `include:` rules; resolve relative paths in imported group files from the imported file's directory. Reject duplicate dns-proxy routes for a group, failed router GETs, and invalid source group names before reconciliation. Router-only `tls_skip_verify` does not disable certificate checks for external URL lists.
- **Scheduler shutdown**: cancellation interrupts retry delays and waits for queue producers to stop safely.
- **AWG**: `update-awg` supports standard WireGuard, AmneziaWG 1.0 (Jc, Jmin, Jmax, S1, S2, H1–H4), and AWG 2.0 (S3, S4, I1–I5; needs Keenetic OS 5.1+). `--dry-run` prints a unified diff and does not apply changes
- **Config save**: mutating RCI batches end with `system configuration save` via `Common.EnsureSaveConfigAtEnd`

## Target Use Cases

- VPN split-tunneling: send selected IPs/domains through VPN, the rest through ISP
- Selective routing: different traffic through different interfaces
- Automated updates: scheduler refreshes route/domain lists from remote URLs
- Multi-site management: one binary, many Keenetic routers

## Exporter YAML (`exporter.yaml`)

Schema: `pkg/config/exporter_config.go`. Example: `exporter_example.yaml`. EN/RU configuration references contain the full field and metric reference. Dashboard: `grafana/keenetic.json`.

`exporter` / `metrics` skips normal root config/auth initialization. It loads its own YAML via `LoadExporterConfig`, accepts `GOKEENAPI_CONFIG`, and reads only connection settings from existing router YAMLs. Never mutate `config.Cfg` or use the CLI's global cookies/caches to poll multiple routers. Obtain isolated read-only sessions via `gokeenrestapi.Common.NewMonitoringSession`; construction does no network I/O.

Defaults: `listen: 127.0.0.1:9479`, `timeout: 10s`, `max_concurrent_routers: 4`, `max_backoff: 5m`. Router entries require unique `name` and `config`; optional `interfaces` selects case-sensitive interface IDs. Paths resolve relative to exporter YAML. The existing credential environment overrides apply to all referenced routers.

Collector settings are `enabled`, `interval`, `max_age`. Defaults: system/DNS/interfaces 30s, hosts 1m, routing 5m, info 1h; max_age is twice interval. Routing is scheduled/reported as independent static_routes/dns_groups/dns_routes collectors. Polling is serialized per router and bounded across routers; failures back off. HTTP `/metrics` only exposes cached successful snapshots, with health/freshness/timestamps; expired or unavailable measurements must never become fabricated zeroes. RCI `show` reads are batched in groups of at most 100; configuration paths use GET. Never resolve domain lists, read full running-config, or invoke mutating commands from exporter.

Exporter HTTP serving uses `net/http`; outgoing router HTTP continues to use resty. Signal shutdown must drain command cleanup before main exits.

`exporter --listen <host:port>` overrides YAML listen for this process with the same validation (numeric port 1–65535). The local Compose stack passes `--listen 0.0.0.0:9479`; host ports remain loopback-bound. `make exporter-local-up EXPORTER_CONFIG=...` mounts the config directory read-only; referenced router YAML paths must stay relative within that directory. Grafana provisions datasource/dashboard automatically from the canonical JSON via generated gitignored runtime assets. Down/status/logs do not require private config. Details: `docker/infra-exporter-local/README.md` and its RU pair.

Exporter display labels use router descriptions/names with technical IDs retained for uniqueness. Add labels directly to existing series rather than metadata series or per-suffix DNS counters; renaming starts a new labelled series. Optional policy names use read-only `/rci/ip/policy` at 5m with 10m expiry and backoff, without invalidating core DNS statistics. Interface names expire with the interface collector. Used RAM excludes free/cache/buffers; physical link speed is optional and distinct from traffic.

Opt-in test variables `GOKEENAPI_LIVE_EXPORTER_CONFIG`, `GOKEENAPI_LIVE_ROUTER_SSH`, `GOKEENAPI_LIVE_REPORT` enable the live router load experiment (absolute config/report paths, report must not exist and is reserved as 0600 before stopping the stand); ordinary tests must skip it. It stops and restores the sole Compose exporter in `EXPORTER_PROJECT` and uses a persistent SSH sampler instead of extra RCI probes. This test intentionally uses a real router, unlike normal API tests using the unified mock. See local Compose guides for procedure and limits.
