---
inclusion: always
---

# Product Overview

gokeenapi is a Go CLI tool for automating Keenetic (Netcraze) router management via REST API. It enables network administrators to manage routes, DNS records, DNS-based routing policies, WireGuard VPN, and execute custom router commands programmatically.

## Core Features

### Route Management
Commands: `add-routes`, `delete-routes`
- Add/delete static IP routes from .bat files or URLs
- Supports IPv4 CIDR notation (e.g., `192.168.1.0/24`)
- Routes are applied to specified interface (e.g., `Wireguard0`)
- Bat-files: plain text, one IP/subnet per line, `#` for comments

### DNS Records
Commands: `add-dns-records`, `delete-dns-records`
- Manage static DNS A/AAAA records for local domain resolution
- Maps domain names to IP addresses on the router's DNS server
- Useful for internal network services and custom domain resolution

### DNS-Routing (Policy-Based Routing)
Commands: `add-dns-routing`, `delete-dns-routing`
- Route traffic by domain name instead of IP address
- Requires Keenetic firmware 5.0.1+ with DNS-routing feature
- Domain-files: plain text, one domain per line, `#` for comments
- Supports wildcard domains (e.g., `*.example.com`)
- Routes DNS queries and subsequent traffic through specified interface

### WireGuard (AWG) Management
Commands: `add-awg`, `configure-awg`
- Configure WireGuard VPN connections from .conf files
- Parses standard WireGuard configuration format
- Creates/updates AWG interface on Keenetic router
- Handles peer configuration, allowed IPs, endpoints

### Known Hosts Cleanup
Command: `delete-known-hosts`
- Remove entries from router's known hosts list
- Supports regex pattern matching for bulk deletion
- Useful for clearing stale SSH/connection records

### Custom Command Execution
Command: `exec`
- Execute arbitrary Keenetic CLI (RCI) commands
- Direct access to router's command-line interface via API
- Returns command output for scripting/automation

### Scheduler
Command: `scheduler`
- Automated task execution at intervals or specific times
- Supports multiple routers with separate configs
- Cron-like scheduling for periodic updates
- See `SCHEDULER.md` for configuration details

### Interface Discovery
Command: `show-interfaces`
- List all available router interfaces (WAN, LAN, VPN, etc.)
- Shows interface names needed for route/DNS-routing commands
- Displays interface status and configuration

## Key Concepts for AI Assistants

### File Format Conventions
- **Bat-files**: Plain text files with IP addresses/subnets for routing
  - One entry per line: `192.168.1.0/24` or `10.0.0.1`
  - Comments start with `#`
  - Empty lines are ignored
  - Located in `batfiles/` directory by convention
  
- **Domain-files**: Plain text files with domains for DNS-routing
  - One domain per line: `example.com` or `*.example.com`
  - Comments start with `#`
  - Empty lines are ignored
  - Located in `custom/domains/` directory by convention

### YAML Configuration Patterns
- **Config expansion**: YAML files can reference other YAML files containing lists
  - Supported list types: `bat-file`, `bat-url`, `domain-file`, `domain-url`
  - Paths in referenced YAML files are resolved relative to that YAML file's directory
  - Example: `custom/config_example.yaml` references `custom/domains/telegram.yaml`
  
- **Multi-router support**: Each router needs its own config file
  - Scheduler can manage multiple routers with separate configs
  - Config specifies router URL, credentials, and resource lists

### Authentication & Environment
- Credentials via environment variables (preferred for security):
  - `GOKEENAPI_KEENETIC_LOGIN` - Router username
  - `GOKEENAPI_KEENETIC_PASSWORD` - Router password
- Credentials can also be in config YAML (less secure)
- Authentication happens automatically in `root.go` PersistentPreRunE

### Command Naming Conventions
- All commands have short aliases defined in `cmd/constants.go`
- Primary command names use kebab-case: `add-dns-routing`
- Aliases are compact: `adnsr`, `adddnsrouting`
- When adding new commands, always define both `CmdXxx` constant and `AliasesXxx` slice

### Interface Names
- Keenetic uses specific interface naming: `Wireguard0`, `Wireguard1`, `ISP`, `HomeNetwork`
- Interface names are case-sensitive
- Use `show-interfaces` command to discover available interfaces
- Interface names are required for route and DNS-routing operations

## Product Behavior Rules

### Idempotency
- Add operations are idempotent: adding existing routes/domains is safe (no duplicates)
- Delete operations are idempotent: deleting non-existent items doesn't error
- Commands validate input before making API calls

### Error Handling
- Commands fail fast on invalid input (bad IPs, malformed domains)
- API errors are surfaced with clear messages
- Use `multierr` to aggregate multiple errors when processing lists

### Validation
- IP addresses/subnets validated before API calls
- Domains validated with IDNA encoding support (internationalized domains)
- Interface existence checked before route/DNS-routing operations
- Config files validated on load (required fields, format)

### Output & Logging
- Use `gokeenlog` package for consistent output formatting
- Info messages for normal operations
- Debug messages for detailed API interactions (when debug flag set)
- Error messages for failures with actionable guidance
- Progress indicators via `gokeenspinner` for long operations

## Target Use Cases

1. **VPN Split-Tunneling**: Route specific IPs/domains through VPN, rest through ISP
2. **Selective Routing**: Different traffic types through different interfaces
3. **Automated Updates**: Scheduler keeps route/domain lists current from URLs
4. **Multi-Site Management**: Single tool manages multiple Keenetic routers
5. **Custom Automation**: Exec command enables scripted router configuration

## When Working on This Codebase

- New commands should follow the pattern in `cmd/` (see structure.md)
- Always add command aliases in `constants.go`
- Use existing API client methods in `pkg/gokeenrestapi/`
- Write both unit tests and property tests for new functionality
- Update example configs in `custom/` if adding new features
- Document new commands in README.md with examples
