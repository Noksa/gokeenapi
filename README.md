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

[🚀 Quick Start](#-quick-start) • [📖 Documentation](#-commands) • [📋 Config Reference](docs/config-reference.md) • [🎨 GUI Version](https://github.com/Noksa/gokeenapiui) • [🤝 Contributing](#-contributing)

</div>

---

## ✨ About

`gokeenapi` is a CLI tool for automating Keenetic (Netcraze) router management. It handles routes, DNS records, DNS-routing, WireGuard connections, known hosts, and scheduled tasks — all via a YAML config file, with no changes required on the router side. Works over LAN or remotely via KeenDNS.

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

`gokeenapi` is configured using a `yaml` file. You can find an example [here](https://github.com/Noksa/gokeenapi/blob/main/config_example.yaml). For a complete description of every field, see the [Config Reference](docs/config-reference.md).

To use your configuration file, pass the `--config <path>` flag with your command.

### Reusable Bat-File and Bat-URL Lists

When managing multiple routers with the same routing configuration, you can create a shared YAML file containing `bat-file` paths, `bat-url` paths, or both, and reference it across multiple configs.

**batfiles/common.yaml:**
```yaml
bat-file:
  - /path/to/discord.bat
  - /path/to/youtube.bat
bat-url:
  - https://example.com/instagram.bat
  - https://example.com/extra.bat
```

**Router config:**
```yaml
routes:
  - interfaceId: Wireguard0
    bat-file:
      - batfiles/common.yaml         # Expanded: only bat-file entries are used
      - /path/to/router-specific.bat # Can mix with regular paths
    bat-url:
      - batfiles/common.yaml         # Expanded: only bat-url entries are used
      - https://example.com/other.bat
```

The tool automatically detects `.yaml`/`.yml` files in the `bat-file` and `bat-url` arrays and expands them to their respective list entries. When a YAML file is referenced in `bat-file`, only its `bat-file` list is used; when referenced in `bat-url`, only its `bat-url` list is used. Relative paths in YAML list files are resolved relative to the YAML file's directory.

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

### TLS Certificate Verification

When connecting to a router over HTTPS with a self-signed certificate, set `tls_skip_verify: true` under the `keenetic` key:

```yaml
keenetic:
  url: https://192.168.1.1
  login: admin
  password: secret
  tls_skip_verify: true  # Disable TLS verification for self-signed certificates
```

> **Note**: Only use `tls_skip_verify` on trusted local networks. Disabling certificate verification exposes the connection to man-in-the-middle attacks.

---

## 📋 Config Reference

For the complete reference of all `config.yaml` fields, see **[docs/config-reference.md](docs/config-reference.md)**.

See also [config_example.yaml](config_example.yaml) for a fully annotated example.

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

> **Tip:** To find interface IDs, run `show-interfaces`.

#### `delete-all-routes`

*Aliases: `deleteallroutes`, `dar`*

Deletes all static routes from the router in a single request, regardless of interface.

```shell
# Delete all routes (with confirmation prompt)
./gokeenapi delete-all-routes --config my_config.yaml

# Delete all routes without confirmation
./gokeenapi delete-all-routes --config my_config.yaml --force
```

> **Warning:** This removes every user-defined static route on the router at once. Use with caution.

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

Updates an existing WireGuard connection from a `.conf` file. Supports AmneziaWG (AWG 2.0) parameters.

```shell
./gokeenapi update-awg --config my_config.yaml --conf-file <path-to-conf> --interface-id <interface-id>

# Preview changes without applying them
./gokeenapi update-awg --config my_config.yaml --conf-file <path-to-conf> --interface-id <interface-id> --dry-run
```

> **Tip:** To find interface IDs, run `show-interfaces`. Use `--dry-run` to see a unified diff of what would change before applying.

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
