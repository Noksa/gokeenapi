---
inclusion: always
---

# Product Overview

**gokeenapi** is a Go CLI tool for automating Keenetic (Netcraze) router management via REST API.

## Core Features

| Feature | Description | Commands |
|---------|-------------|----------|
| Route Management | Add/delete static routes from .bat files and URLs | `add-routes`, `delete-routes` |
| DNS Records | Manage static DNS records for local domain resolution | `add-dns-records`, `delete-dns-records` |
| DNS-Routing | Policy-based routing by domain (requires firmware 5.0.1+) | `add-dns-routing`, `delete-dns-routing` |
| WireGuard (AWG) | Configure WireGuard VPN connections from .conf files | `add-awg`, `update-awg` |
| Known Hosts | Clean up known hosts with regex pattern matching | `delete-known-hosts` |
| Custom Commands | Execute arbitrary Keenetic CLI commands | `exec` |
| Scheduler | Automated task execution at intervals or specific times | `scheduler` |
| Interface Discovery | List available router interfaces | `show-interfaces` |

## Key Concepts

- **Bat-files**: Text files with IP addresses/subnets for routing (one per line, `#` for comments)
- **Domain-files**: Text files with domains for DNS-routing (one per line, `#` for comments)
- **YAML expansion**: Config files can reference `.yaml`/`.yml` files containing lists of `bat-file`, `bat-url`, `domain-file`, or `domain-url` entries - paths are resolved relative to the YAML file's directory
- **Multi-router**: Each router requires its own config file; scheduler can manage multiple routers with separate configs
- **Environment variables**: `GOKEENAPI_KEENETIC_LOGIN` and `GOKEENAPI_KEENETIC_PASSWORD` for credentials

## Command Aliases

All commands have short aliases (defined in `cmd/constants.go`):
- `show-interfaces` → `si`, `showifaces`
- `add-routes` → `ar`, `addroutes`
- `delete-routes` → `dr`, `deleteroutes`
- `add-dns-routing` → `adnsr`, `adddnsrouting`
- `delete-dns-routing` → `ddnsr`, `deletednsrouting`

## Target Users

Network administrators automating Keenetic router configuration for VPN split-tunneling and selective routing scenarios.
