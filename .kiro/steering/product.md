---
inclusion: always
---

# Product Overview

`gokeenapi` is a Go CLI tool for automating Keenetic (Netcraze) router management via REST API. It manages IP routes, DNS records, DNS-based routing policies, WireGuard/AmneziaWG VPN, and executes arbitrary RCI commands.

## Commands Reference

All command names and aliases are defined in `cmd/constants.go`. Always define both `CmdXxx` (constant, kebab-case) and `AliasesXxx` (slice, compact no hyphens) when adding a command.

| Command | Aliases | Purpose |
|---|---|---|
| `add-routes` | `addroutes`, `ar` | Add static IP routes from bat-files or URLs |
| `delete-routes` | `deleteroutes`, `dr` | Delete static IP routes |
| `delete-all-routes` | `deleteallroutes`, `dar` | Remove all static routes in a single request |
| `add-dns-records` | `adddnsrecords`, `adr` | Add static DNS A/AAAA records |
| `delete-dns-records` | `deletednsrecords`, `ddr` | Delete static DNS records |
| `add-dns-routing` | `adddnsrouting`, `adnsr` | Add policy-based DNS routing by domain |
| `delete-dns-routing` | `deletednsrouting`, `ddnsr` | Delete DNS routing entries |
| `add-awg` | `addawg`, `aawg` | Add WireGuard/AWG VPN from a `.conf` file |
| `update-awg` | `updateawg`, `uawg` | Update existing AWG interface from a `.conf` file; supports `--dry-run` |
| `delete-known-hosts` | `deleteknownhosts`, `dkh` | Remove known host entries (regex supported) |
| `exec` | `e` | Execute arbitrary Keenetic RCI commands |
| `scheduler` | `schedule`, `sched` | Run automated scheduled tasks |
| `show-interfaces` | `showinterfaces`, `si` | List all router interfaces |
| `version` | — | Print build version info |

`PersistentPreRunE` in `root.go` skips auth/config init for `completion`, `help`, `scheduler`, and `version` commands.

## API Singletons

Never instantiate these — use the package-level singletons from `pkg/gokeenrestapi/`:

| Singleton | Responsibility |
|---|---|
| `gokeenrestapi.Common` | Auth, raw RCI execution |
| `gokeenrestapi.Ip` | IP route management (`AddRoutes`, `DeleteRoutes`, `DeleteAllRoutes`) |
| `gokeenrestapi.DnsRouting` | DNS-routing management |
| `gokeenrestapi.AwgConf` | AWG/WireGuard configuration and diff-update |
| `gokeenrestapi.Checks` | Input validation (`CheckInterfaceId`, `CheckInterfaceExists`, `CheckComponentInstalled`) |
| `gokeenrestapi.Interface` | Interface listing |

## Input File Formats

**Bat-files** (`batfiles/`) — IP route sources:
- One IPv4 CIDR or address per line: `192.168.1.0/24`, `10.0.0.1`
- `#`-prefixed lines and empty lines are ignored

**Domain-files** (`custom/domains/`) — DNS-routing sources:
- One domain per line: `example.com`, `*.example.com`
- `#`-prefixed lines and empty lines are ignored
- Supports wildcards and IDNA-encoded internationalized domains

## YAML Configuration

- Each router has its own config YAML (`custom/config_*.yaml`)
- Configs support **expansion**: reference other YAML files via `bat-file`, `bat-url`, `domain-file`, `domain-url` list types; paths resolve relative to the referencing file
- Scheduler config (`custom/scheduler.yaml`) references per-router configs for multi-router automation
- Scheduler supports `sequential` and `parallel` execution strategies (`cmd/constants.go`: `StrategySequential`, `StrategyParallel`)
- Only required field: `keenetic-url`; credentials must use env vars, not YAML

## Authentication

Resolved in priority order (env vars win):
1. `GOKEENAPI_KEENETIC_LOGIN` / `GOKEENAPI_KEENETIC_PASSWORD`
2. `keenetic-login` / `keenetic-password` in config YAML

Auth is handled automatically by `PersistentPreRunE` in `root.go`. Commands must never call `Auth()` directly.

## Interface Names

Interface names are case-sensitive: `Wireguard0`, `Wireguard1`, `ISP`, `HomeNetwork`. Use `show-interfaces` to discover available names. An interface name is a required argument for route and DNS-routing commands.

## Behavioral Contracts

- **Idempotency**: add/delete operations are safe to repeat — no duplicates created, no errors on missing items
- **Fail-fast validation**: validate inputs (IPs, domains, interface existence) via `gokeenrestapi.Checks` before any API call
- **Error aggregation**: always use `multierr.Append()` when iterating lists — collect all errors, not just the first
- **DNS-routing firmware**: requires Keenetic firmware 5.0.1+
- **AWG support**: `update-awg` handles standard WireGuard, AmneziaWG 1.0 (Jc, Jmin, Jmax, S1, S2, H1–H4), and AWG 2.0 (S3, S4, I1–I5) parameters

## Adding a New Command — Checklist

1. **`cmd/constants.go`** — add `CmdXxx` constant (kebab-case) and `AliasesXxx` slice (compact, no hyphens)
2. **`cmd/<command_name>.go`** — implement `newXxxCmd() *cobra.Command`
   - Always `RunE`, never `Run`
   - Access `config.Cfg` and API singletons only inside `RunE`, never in the constructor
   - Use `gokeenlog` for all output — never `fmt.Println` or `log.Println`
   - Return errors from `RunE` — never `os.Exit()`
   - Validate inputs via `gokeenrestapi.Checks` before calling other singletons
3. **`cmd/root.go`** — register with `rootCmd.AddCommand(newXxxCmd())`
4. Write `*_test.go` (unit) and `*_property_test.go` (property-based) — activate `golang-testing` skill for patterns
5. Update `custom/` example configs and `README.md` if the feature affects configuration

## Target Use Cases

- VPN split-tunneling: route specific IPs/domains through VPN, rest through ISP
- Selective routing: different traffic through different interfaces
- Automated updates: scheduler keeps route/domain lists current from remote URLs
- Multi-site management: single binary manages multiple Keenetic routers
