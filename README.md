<div align="center">

<img src="logo.png" alt="gokeenapi logo" width="512">

# 🚀 gokeenapi

**Automate your Keenetic (Netcraze) router management with ease**

<p align="center">
  <video src="https://github.com/user-attachments/assets/404e89cc-4675-42c4-ae93-4a0955b06348" width="100%"></video>
</p>

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker Pulls](https://img.shields.io/docker/pulls/noksa/gokeenapi)](https://hub.docker.com/r/noksa/gokeenapi)
[![GitHub release](https://img.shields.io/github/release/Noksa/gokeenapi.svg)](https://github.com/Noksa/gokeenapi/releases)

*Tired of clicking through Keenetic (Netcraze) web interface? Automate your Keenetic (Netcraze) router management with simple CLI commands.*

<div align="center">

### [🇷🇺 **Русская версия документации** 🇷🇺](README_RU.md)

</div>

[🚀 Quick Start](#-quick-start) • [📖 Documentation](#-commands) • [📋 Config Reference](#-config-reference) • [🎨 GUI Version](https://github.com/Noksa/gokeenapiui) • [🤝 Contributing](#-contributing)

</div>

---

## ✨ Why Choose gokeenapi?

<table>
<tr>
<td width="50%">

### 💻 **Automate Everything**
Manage routes, DNS records, DNS-routing, WireGuard connections, and known hosts with simple commands

### ⚙️ **Zero Router Setup**
No complex configuration needed on your router - just provide the address

</td>
<td width="50%">

### 🌐 **Works Anywhere**
LAN or Internet access via KeenDNS - your choice

### 🎯 **Precise Control**
Delete static routes for specific interfaces without affecting others

</td>
</tr>
</table>

---

## 🎨 Prefer a GUI?

Not a command-line person? We've got you covered! Check out our user-friendly GUI version:

<div align="center">

### [🎨 **GUI Version Available** 🚀](https://github.com/Noksa/gokeenapiui)

[![GUI Version](https://img.shields.io/badge/🎨_Try_GUI_Version-Click_Here-brightgreen?style=for-the-badge&logo=github)](https://github.com/Noksa/gokeenapiui)

</div>

---

## 🚀 Quick Start

The easiest way to get started is by using Docker or by downloading the latest release.

### 🐳 Docker (Recommended)

Using Docker is the recommended way to run `gokeenapi`.

```bash
# Pull the Docker image
export GOKEENAPI_IMAGE="noksa/gokeenapi:stable"
docker pull "${GOKEENAPI_IMAGE}"

# Run a command
docker run --rm -ti -v "$(pwd)/config_example.yaml":/gokeenapi/config.yaml \
  "${GOKEENAPI_IMAGE}" show-interfaces --config /gokeenapi/config.yaml
```

### 📦 Latest Release

Download the latest release for your platform:

<div align="center">

[![Download Latest](https://img.shields.io/badge/📦_Download-Latest_Release-green?style=for-the-badge)](https://github.com/Noksa/gokeenapi/releases)

</div>

---

## ⚙️ Configuration

`gokeenapi` is configured using a `yaml` file. You can find an example [here](https://github.com/Noksa/gokeenapi/blob/main/config_example.yaml). For a complete description of every field, see the [Config Reference](#-config-reference) section below.

To use your configuration file, pass the `--config <path>` flag with your command.

### Reusable Bat-File Lists

When managing multiple routers with the same routing configuration, you can create a YAML file containing a list of bat-file paths and reference it across multiple configs:

**batfiles/common-routes.yaml:**
```yaml
bat-file:
  - /path/to/discord.bat
  - /path/to/youtube.bat
  - /path/to/instagram.bat
```

**Router config:**
```yaml
routes:
  - interfaceId: Wireguard0
    bat-file:
      - batfiles/common-routes.yaml  # Automatically expanded
      - /path/to/router-specific.bat # Can mix with regular files
```

The tool automatically detects `.yaml`/`.yml` files in the `bat-file` array and expands them to their contained bat-file paths. Relative paths in YAML list files are resolved relative to the YAML file's directory.

### Reusable Bat-URL Lists

Similar to bat-file lists, you can create reusable YAML files containing bat-url paths:

**batfiles/common-urls.yaml:**
```yaml
bat-url:
  - https://example.com/discord.bat
  - https://example.com/youtube.bat
  - https://example.com/instagram.bat
```

**Router config:**
```yaml
routes:
  - interfaceId: Wireguard0
    bat-url:
      - batfiles/common-urls.yaml    # Automatically expanded
      - https://example.com/extra.bat # Can mix with regular URLs
```

The tool automatically detects `.yaml`/`.yml` files in the `bat-url` array and expands them to their contained bat-url paths. Relative paths in YAML list files are resolved relative to the YAML file's directory.

**Note:** You can combine both `bat-file` and `bat-url` in the same YAML file. When a YAML file is referenced in `bat-file`, only its `bat-file` list is expanded. When referenced in `bat-url`, only its `bat-url` list is expanded:

```yaml
bat-file:
  - /path/to/file1.bat
  - /path/to/file2.bat
bat-url:
  - https://example.com/url1.bat
  - https://example.com/url2.bat
```

This allows you to maintain both local files and remote URLs in a single reusable YAML file, referencing it appropriately in your config.

### Environment Variables

All configuration options can be set via environment variables:

| Variable | Description |
|---|---|
| `GOKEENAPI_CONFIG` | Path to config file (alternative to `--config`) |
| `GOKEENAPI_KEENETIC_LOGIN` | Router admin login |
| `GOKEENAPI_KEENETIC_PASSWORD` | Router admin password |
| `GOKEENAPI_INSIDE_DOCKER` | When set, uses `/etc/gokeenapi` as the data directory |

`GOKEENAPI_KEENETIC_LOGIN` and `GOKEENAPI_KEENETIC_PASSWORD` are particularly useful for keeping sensitive credentials out of config files. `GOKEENAPI_INSIDE_DOCKER` is set automatically in the official Docker image.

> **Security recommendation**: Store credentials using environment variables instead of writing them directly into the config file. Config files stored with world-readable permissions (e.g. `0644`) will trigger a runtime warning. Restrict permissions with `chmod 600 config.yaml` and use `GOKEENAPI_KEENETIC_LOGIN` / `GOKEENAPI_KEENETIC_PASSWORD` to pass credentials. Add `config.yaml` and `config_*.yaml` to your `.gitignore` to prevent accidental commits (the project's default `.gitignore` already includes these patterns).

---

## 📋 Config Reference

Complete reference for all fields in the `config.yaml` file. See [config_example.yaml](config_example.yaml) for a fully annotated example.

### `keenetic` — Router connection

Required by every command.

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `url` | string | ✅ | — | Router URL. Accepts IP address or KeenDNS hostname. Supports `http://` and `https://`. Example: `http://192.168.1.1` |
| `login` | string | ✅ | — | Router admin username. Overridden by `GOKEENAPI_KEENETIC_LOGIN` env var. |
| `password` | string | ✅ | — | Router admin password. Overridden by `GOKEENAPI_KEENETIC_PASSWORD` env var. |
| `tls_skip_verify` | bool | ❌ | `false` | Disable TLS certificate verification. Enable when the router uses a self-signed certificate. |

### `dataDir` — Data directory

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `dataDir` | string | ❌ | system default | Custom directory for application data files. Automatically set to `/etc/gokeenapi` when `GOKEENAPI_INSIDE_DOCKER` env var is present. |

### `routes` — Static routes

Used by: `add-routes`, `delete-routes`.

A list of per-interface routing configurations.

| Field | Type | Required | Description |
|---|---|---|---|
| `interfaceId` | string | ✅ | Target interface ID (e.g. `Wireguard0`). Run `show-interfaces` to list available IDs. |
| `bat-file` | list of strings | ❌ | Paths to local `.bat` files with `route add` commands. A `.yaml`/`.yml` path is expanded to the `bat-file` list it contains. Relative paths are resolved from the config file's directory. |
| `bat-url` | list of strings | ❌ | Remote URLs serving `.bat` files. A `.yaml`/`.yml` path is expanded to the `bat-url` list it contains. |

At least one of `bat-file` or `bat-url` should be provided per entry.

### `dns.records` — Static DNS records

Used by: `add-dns-records`, `delete-dns-records`.

A list of static DNS host entries to create in the router's local resolver. Each entry maps one domain name to one or more IPv4 addresses.

| Field | Type | Required | Description |
|---|---|---|---|
| `domain` | string | ✅ | Domain name to resolve (e.g. `myserver.local`). |
| `ip` | list of strings | ✅ | One or more IPv4 addresses the domain should resolve to. |

Example:

```yaml
dns:
  records:
    - domain: myserver.local
      ip:
        - 192.168.1.100
        - 192.168.1.101
    - domain: db.local
      ip:
        - 10.0.0.5
```

### `dns.routes.groups` — DNS-routing groups

Used by: `add-dns-routing`, `delete-dns-routing`.

A list of domain groups for policy-based routing. Each group creates an object-group and a dns-proxy route on the router. Requires Keenetic firmware ≥ 5.0.1.

Each entry is either a **group object** or a **path to a `.yaml` file** that contains a `groups:` list (for sharing common groups across multiple router configs).

**Group object fields:**

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | ✅ | Unique name for the object-group on the router. |
| `interfaceId` | string | ✅ | Target interface for routing matched domain traffic (e.g. `Wireguard0`). Run `show-interfaces` to list available IDs. |
| `domain-file` | list of strings | ❌ | Paths to local `.txt` files with one domain per line (lines starting with `#` are comments). A `.yaml`/`.yml` path is expanded to the `domain-file` list it contains. |
| `domain-url` | list of strings | ❌ | Remote URLs serving domain lists (one domain per line). A `.yaml`/`.yml` path is expanded to the `domain-url` list it contains. |

At least one of `domain-file` or `domain-url` is required per group.

Example:

```yaml
dns:
  routes:
    groups:
      - name: streaming
        domain-file:
          - domains/netflix.txt
          - domains/youtube.txt
        domain-url:
          - https://example.com/spotify-domains.txt
        interfaceId: Wireguard0

      - name: work-vpn
        domain-file:
          - domains/internal.txt
        interfaceId: GigabitEthernet0/Vlan4

      # Import shared groups from a file
      - common/shared_groups.yaml
```

### `add-awg` / `update-awg` — WireGuard commands

These commands do not read any command-specific section from the config file. They require only the `keenetic` connection block. The WireGuard configuration is supplied via the `--conf-file` CLI flag, which points to a standard WireGuard `.conf` file:

```ini
[Interface]
PrivateKey = <private-key>
Address = 10.0.0.2/32

[Peer]
PublicKey = <public-key>
Endpoint = vpn.example.com:51820
AllowedIPs = 0.0.0.0/0
```

| CLI flag | Required | Description |
|---|---|---|
| `--conf-file` | ✅ | Path to a WireGuard `.conf` file. |
| `--name` | ❌ (`add-awg` only) | Name for the new interface. Auto-generated if omitted. |
| `--interface-id` | ✅ (`update-awg` only) | ID of the existing interface to update. |

### `logs` — Logging

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `debug` | bool | ❌ | `false` | Enable debug-level logging. Also controllable with the `--debug` global flag. |

### `cache` — Caching

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `urlTtl` | duration | ❌ | `1m` | How long to cache content downloaded from `bat-url` and `domain-url`. Accepts Go duration strings: `30s`, `5m`, `1h`. |

---

## 🔧 Supported Routers

`gokeenapi` has been tested with the following Keenetic (Netcraze) router models:

- **Keenetic (Netcraze) Start**
- **Keenetic (Netcraze) Viva** 
- **Keenetic (Netcraze) Giga**

Since the utility works with Keenetic (Netcraze) Start (the most affordable model in the lineup), it should be compatible with all Keenetic (Netcraze) router models.

---

## 🎬 Video Demos

Check out these video demonstrations (in Russian) to see `gokeenapi` in action:

*   [Routes Management](https://www.youtube.com/watch?v=lKX74btFypY)

---

## 🕐 Scheduler - Automated Task Execution

The scheduler allows you to automate router management by running tasks at specified intervals or fixed times. This is perfect for keeping routes and DNS records up-to-date automatically.

### Key Features

- **Interval-based execution**: Run tasks every N hours/minutes (e.g., every 3 hours)
- **Time-based execution**: Run tasks at specific times (e.g., at 02:00, 06:00, 12:00)
- **Command chaining**: Execute multiple commands sequentially (e.g., delete-routes → add-routes)
- **Multi-router support**: Manage multiple routers with a single task
- **Retry mechanism**: Automatically retry failed tasks with configurable delay
- **Sequential execution**: Tasks run in a queue to avoid conflicts

### Quick Start

```shell
# Run scheduler with config
./gokeenapi scheduler --config scheduler.yaml
```

### Example Configuration

```yaml
tasks:
  - name: "Update routes every 3 hours"
    commands:
      - add-routes
    configs:
      - /path/to/router1.yaml
      - /path/to/router2.yaml
      - /path/to/router3.yaml
    interval: "3h"
  
  - name: "Refresh routes daily with retry"
    commands:
      - delete-routes
      - add-routes
    configs:
      - /path/to/router1.yaml
    times:
      - "02:00"
    retry: 3           # Retry up to 3 times on failure
    retryDelay: "30s"  # Wait 30 seconds between retries
```

📖 **[Read full Scheduler documentation →](SCHEDULER.md)**

See also: [scheduler_example.yaml](scheduler_example.yaml)

---

### 📚 Commands

Here are some of the things you can do with `gokeenapi`. For a full list of commands and options, use the `--help` flag.

```shell
./gokeenapi --help
```

#### `show-interfaces`

*Aliases: `showinterfaces`, `si`, `showinterface`, `show-interface`*

Displays all available interfaces on your Keenetic (Netcraze) router.

```shell
# Show all interfaces
./gokeenapi show-interfaces --config my_config.yaml

# Show only WireGuard interfaces
./gokeenapi show-interfaces --config my_config.yaml --type Wireguard
```

#### `add-routes`

*Aliases: `addroutes`, `ar`*

Adds static routes to your router.

```shell
./gokeenapi add-routes --config my_config.yaml
```

#### `delete-routes`

*Aliases: `deleteroutes`, `dr`*

Deletes static routes for a specific interface.

```shell
# Delete routes for all interfaces in the config file
./gokeenapi delete-routes --config my_config.yaml

# Delete routes for a specific interface
./gokeenapi delete-routes --config my_config.yaml --interface-id <your-interface-id>

# Delete routes without confirmation prompt
./gokeenapi delete-routes --config my_config.yaml --force
```

#### `add-dns-records`

*Aliases: `adddnsrecords`, `adr`*

Adds static DNS records.

```shell
./gokeenapi add-dns-records --config my_config.yaml
```

#### `delete-dns-records`

*Aliases: `deletednsrecords`, `ddr`*

Deletes static DNS records based on your configuration file.

```shell
./gokeenapi delete-dns-records --config my_config.yaml
```

#### `add-dns-routing`

*Aliases: `adddnsrouting`, `adnsr`, `adddnsroutes`, `add-dns-routes`*

Adds DNS-routing rules (policy-based routing by domain) to your router. This feature allows you to route traffic for specific domains through designated network interfaces.

**Requirements:** Keenetic firmware version 5.0.1 or higher

```shell
./gokeenapi add-dns-routing --config my_config.yaml
```

**How it works:**
- Loads domains from local .txt files and remote URLs
- Creates domain groups (object-groups) containing your specified domains and IP addresses
- Associates each group with a network interface via dns-proxy routes
- Traffic for domains in a group is automatically routed through the specified interface

**Domain sources:**
- Local .txt files with one domain per line (supports comments with #)
- Remote URLs serving domain lists
- YAML files containing lists of domain-file or domain-url paths (for organization)

**YAML expansion:** The tool automatically detects `.yaml`/`.yml` files in the `domain-file` and `domain-url` arrays and expands them to their contained domain paths (similar to bat-file/bat-url expansion).

**NEW: Reusable DNS Routing Groups**

You can now create shared YAML files containing complete DNS routing group definitions and import them across multiple router configs. This is different from domain-file/domain-url expansion - you're importing entire group definitions, not just domain lists.

**custom/common_dns_groups.yaml:**
```yaml
groups:
  - name: youtube
    domain-url:
      - domains/youtube.yaml
    interfaceId: Wireguard0
  - name: telegram
    domain-url:
      - domains/telegram.yaml
    interfaceId: Wireguard0
  - name: trackers
    domain-file:
      - domains/trackers.yaml
    interfaceId: Wireguard0
```

**Router config:**
```yaml
dns:
  routes:
    groups:
      - common_dns_groups.yaml    # Import all groups from file
      - name: router-specific     # Mix with router-specific groups
        domain-file:
          - domains/local.txt
        interfaceId: GigabitEthernet0
```

This allows you to maintain common DNS routing rules in one place and share them across all your routers. When you add telegram to one router, just update `common_dns_groups.yaml` and all routers using it will get the update.

**Example use cases:**
- Route social media traffic through a VPN (Wireguard0)
- Route streaming services through a different connection
- Split traffic by domain for load balancing or privacy
- Use community-maintained domain lists from URLs

#### `delete-dns-routing`

*Aliases: `deletednsrouting`, `ddnsr`, `deletednsroutes`, `delete-dns-routes`*

Deletes DNS-routing rules that match your configuration file.

```shell
# Delete DNS-routing rules with confirmation prompt
./gokeenapi delete-dns-routing --config my_config.yaml

# Delete DNS-routing rules without confirmation prompt
./gokeenapi delete-dns-routing --config my_config.yaml --force
```

The command will:
1. Identify dns-proxy routes and object-groups matching your configuration
2. Display the rules to be deleted
3. Request confirmation (unless `--force` flag is used)
4. Remove dns-proxy routes first, then object-groups

#### `add-awg`

*Aliases: `addawg`, `aawg`*

Adds a new WireGuard connection from a `.conf` file.

```shell
./gokeenapi add-awg --config my_config.yaml --conf-file <path-to-conf> --name MySuperInterface
```

#### `update-awg`

*Aliases: `updateawg`, `uawg`*

Updates an existing WireGuard connection from a `.conf` file.

```shell
./gokeenapi update-awg --config my_config.yaml --conf-file <path-to-conf> --interface-id <interface-id>
```

#### `delete-known-hosts`

*Aliases: `deleteknownhosts`, `dkh`*

Deletes known hosts by name or MAC using regex pattern.

```shell
# Delete hosts by name pattern
./gokeenapi delete-known-hosts --config my_config.yaml --name-pattern "pattern"

# Delete hosts by MAC pattern
./gokeenapi delete-known-hosts --config my_config.yaml --mac-pattern "pattern"

# Delete hosts without confirmation prompt
./gokeenapi delete-known-hosts --config my_config.yaml --name-pattern "pattern" --force
```

#### `scheduler`

*Aliases: `schedule`, `sched`*

Runs automated tasks at specified intervals or fixed times. See [Scheduler documentation](SCHEDULER.md) for the full configuration reference.

```shell
./gokeenapi scheduler --config scheduler.yaml
```

#### `exec`

*Aliases: `e`*

Execute custom Keenetic (Netcraze) CLI commands directly on your router.

```shell
# Show system information
./gokeenapi exec --config my_config.yaml show version

# Display interface statistics
./gokeenapi exec --config my_config.yaml show interface

# Show routing table
./gokeenapi exec --config my_config.yaml show ip route
```

---

### 🤝 Contributing

Contributions are welcome! If you have any ideas, suggestions, or bug reports, please open an issue or create a pull request.

---

### 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
