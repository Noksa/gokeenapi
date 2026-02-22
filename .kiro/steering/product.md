---
inclusion: always
---

# Product Overview

gokeenapi is a Go CLI tool for automating Keenetic (Netcraze) router management via REST API. It manages routes, DNS records, DNS-based routing policies, WireGuard VPN, and executes custom router commands programmatically.

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

All command names and aliases are defined in `cmd/constants.go`. When adding a new command, always define both `CmdXxx` (constant) and `AliasesXxx` (slice) there.

## Input File Formats

**Bat-files** (`batfiles/` directory) — IP routes source:
- One IPv4 CIDR or address per line: `192.168.1.0/24`, `10.0.0.1`
- Lines starting with `#` are comments; empty lines are ignored

**Domain-files** (`custom/domains/` directory) — DNS-routing source:
- One domain per line: `example.com`, `*.example.com`
- Lines starting with `#` are comments; empty lines are ignored
- Supports wildcard and IDNA-encoded internationalized domains

## YAML Configuration

- Each router has its own config YAML (see `custom/config_*.yaml` for examples)
- Config files support **expansion**: they can reference other YAML files for lists
  - Supported list types: `bat-file`, `bat-url`, `domain-file`, `domain-url`
  - Referenced file paths resolve relative to the referencing file's directory
- Scheduler config (`custom/scheduler.yaml`) references per-router configs for multi-router automation
- Required fields: `keenetic-url`; credentials prefer env vars over YAML

## Authentication

Credentials are resolved in this order (env vars take precedence):
- `GOKEENAPI_KEENETIC_LOGIN` / `GOKEENAPI_KEENETIC_PASSWORD`
- `keenetic-login` / `keenetic-password` fields in config YAML

Authentication is automatic — handled in `root.go` PersistentPreRunE. Commands never call `Auth()` directly.

## Interface Names

Keenetic interface names are case-sensitive: `Wireguard0`, `Wireguard1`, `ISP`, `HomeNetwork`. Use `show-interfaces` to discover available names. Interface name is a required argument for route and DNS-routing commands.

## Behavioral Contracts

- **Idempotency**: add/delete operations are safe to repeat — no duplicates created, no errors on missing items
- **Fail-fast validation**: inputs (IPs, domains, interface existence) are validated before any API call
- **Error aggregation**: use `multierr.Append()` when processing lists so all errors are collected, not just the first
- **DNS-routing firmware requirement**: requires Keenetic firmware 5.0.1+

## Adding New Commands — Checklist

1. Add `CmdXxx` constant and `AliasesXxx` slice in `cmd/constants.go`
2. Create `cmd/<command_name>.go` with `newXxxCmd()` returning `*cobra.Command`
   - Use `RunE`, not `Run`
   - Use `gokeenlog` for all output — never `fmt.Println`
   - Call API singletons (`gokeenrestapi.Ip`, `gokeenrestapi.DnsRouting`, etc.) inside `RunE`, not in the constructor
3. Register with `rootCmd.AddCommand(newXxxCmd())` in `cmd/root.go`
4. Write unit tests (`*_test.go`) and property tests (`*_property_test.go`)
5. Update `custom/` example configs and `README.md` if the feature affects configuration

## Target Use Cases

- VPN split-tunneling: route specific IPs/domains through VPN, rest through ISP
- Selective routing: different traffic types through different interfaces
- Automated updates: scheduler keeps route/domain lists current from remote URLs
- Multi-site management: single tool manages multiple Keenetic routers
