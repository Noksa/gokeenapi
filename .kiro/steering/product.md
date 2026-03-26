---
inclusion: always
---

# Product Overview

`gokeenapi` is a Go CLI tool for automating Keenetic (Netcraze) router management via REST API. It manages IP routes, DNS records, DNS-based routing policies, WireGuard VPN, and executes arbitrary router RCI commands.

## Commands Reference

| Command | Aliases | Purpose |
|---------|---------|---------|
| `add-routes` | `ar`, `addroutes` | Add static IP routes from bat-files or URLs |
| `delete-routes` | `dr`, `deleteroutes` | Delete static IP routes |
| `add-dns-records` | `adnsr` | Add static DNS A/AAAA records |
| `delete-dns-records` | `ddnsr` | Delete static DNS records |
| `add-dns-routing` | `adnsr`, `adddnsrouting` | Add policy-based DNS routing by domain |
| `delete-dns-routing` | `ddnsr`, `deletednsrouting` | Delete DNS routing entries |
| `add-awg` | `aawg` | Add WireGuard VPN from .conf file |
| `configure-awg` | `cawg` | Configure existing AWG interface |
| `delete-known-hosts` | `dkh` | Remove known host entries (regex supported) |
| `exec` | `e` | Execute arbitrary Keenetic RCI commands |
| `scheduler` | `s`, `sch` | Run automated scheduled tasks |
| `show-interfaces` | `si`, `showi` | List all router interfaces |

All command names and aliases are defined in `cmd/constants.go`. Always define both `CmdXxx` (constant) and `AliasesXxx` (slice) when adding a command.

## Input File Formats

**Bat-files** (`batfiles/`) — IP route sources:
- One IPv4 CIDR or address per line: `192.168.1.0/24`, `10.0.0.1`
- `#`-prefixed lines and empty lines are ignored

**Domain-files** (`custom/domains/`) — DNS-routing sources:
- One domain per line: `example.com`, `*.example.com`
- `#`-prefixed lines and empty lines are ignored
- Supports wildcards and IDNA-encoded internationalized domains

## YAML Configuration

- Each router has its own config YAML (see `custom/config_*.yaml` for examples)
- Configs support **expansion**: reference other YAML files for lists via `bat-file`, `bat-url`, `domain-file`, `domain-url` list types; paths resolve relative to the referencing file
- Scheduler config (`custom/scheduler.yaml`) references per-router configs for multi-router automation
- Only required field: `keenetic-url`; credentials should use env vars, not YAML

## Authentication

Resolved in priority order (env vars win):
1. `GOKEENAPI_KEENETIC_LOGIN` / `GOKEENAPI_KEENETIC_PASSWORD`
2. `keenetic-login` / `keenetic-password` in config YAML

Auth is automatic via `root.go` `PersistentPreRunE`. Commands never call `Auth()` directly.

## Interface Names

Interface names are case-sensitive: `Wireguard0`, `Wireguard1`, `ISP`, `HomeNetwork`. Use `show-interfaces` to discover available names. Interface name is a required argument for route and DNS-routing commands.

## Behavioral Contracts

- **Idempotency**: add/delete operations are safe to repeat — no duplicates created, no errors on missing items
- **Fail-fast validation**: validate inputs (IPs, domains, interface existence) before any API call
- **Error aggregation**: always use `multierr.Append()` when iterating lists — collect all errors, not just the first
- **DNS-routing firmware**: requires Keenetic firmware 5.0.1+

## Adding a New Command — Checklist

1. `cmd/constants.go` — add `CmdXxx` constant (kebab-case) and `AliasesXxx` slice (compact, no hyphens)
2. `cmd/<command_name>.go` — implement `newXxxCmd() *cobra.Command`
   - Use `RunE`, never `Run`
   - Access `config.Cfg` and API singletons only inside `RunE`, never in the constructor
   - Use `gokeenlog` for all output — never `fmt.Println` or `log.Println`
   - Return errors from `RunE` — never `os.Exit()`
   - Validate inputs via `gokeenrestapi.Checks` before calling other API singletons
3. `cmd/root.go` — register with `rootCmd.AddCommand(newXxxCmd())`
4. Write `*_test.go` (unit) and `*_property_test.go` (property-based) tests — activate `golang-testing` skill for patterns
5. Update `custom/` example configs and `README.md` if the feature affects configuration

## Target Use Cases

- VPN split-tunneling: route specific IPs/domains through VPN, rest through ISP
- Selective routing: different traffic types through different interfaces
- Automated updates: scheduler keeps route/domain lists current from remote URLs
- Multi-site management: single tool manages multiple Keenetic routers
