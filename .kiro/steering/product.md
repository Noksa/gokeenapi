# Product Overview

**gokeenapi** is a Go CLI tool for automating Keenetic (Netcraze) router management via REST API.

## Core Features

- **Route Management**: Add/delete static routes from .bat files and URLs
- **DNS Records**: Manage static DNS records for local domain resolution
- **DNS-Routing**: Policy-based routing by domain (route specific domains through designated interfaces)
- **WireGuard (AWG)**: Configure WireGuard VPN connections from .conf files
- **Known Hosts**: Clean up known hosts with pattern matching
- **Custom Commands**: Execute arbitrary Keenetic CLI commands via `exec`
- **Scheduler**: Automated task execution at intervals or specific times

## Key Concepts

- **Bat-files**: Text files containing IP addresses/subnets for routing (one per line)
- **Domain-files**: Text files containing domains for DNS-routing (one per line)
- **YAML expansion**: Config files can reference other YAML files containing lists of bat-files, bat-urls, domain-files, or domain-urls for reusability
- **Multi-router**: Each router requires its own config file; scheduler can manage multiple routers

## Target Users

Network administrators and power users who want to automate Keenetic router configuration, especially for VPN split-tunneling and selective routing scenarios.
