<div align="center">

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

[🚀 Quick Start](#-quick-start) • [📖 Documentation](#-commands) • [🎨 GUI Version](https://github.com/Noksa/gokeenapiui) • [🤝 Contributing](#-contributing)

</div>

---

## ✨ Why Choose gokeenapi?

<table>
<tr>
<td width="50%">

### 💻 **Automate Everything**
Manage routes, DNS records, WireGuard connections, and known hosts with simple commands

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

`gokeenapi` is configured using a `yaml` file. You can find an example [here](https://github.com/Noksa/gokeenapi/blob/main/config_example.yaml).

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

The tool automatically detects `.yaml`/`.yml` files in the `bat-file` array and expands them to their contained bat-file paths. Relative paths in YAML list files are resolved relative to the main config file's directory.

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

The tool automatically detects `.yaml`/`.yml` files in the `bat-url` array and expands them to their contained bat-url paths. Relative paths in YAML list files are resolved relative to the main config file's directory.

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

For security, you can store sensitive credentials as environment variables instead of in the config file:

- `GOKEENAPI_KEENETIC_LOGIN` - Router admin login
- `GOKEENAPI_KEENETIC_PASSWORD` - Router admin password

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

*Aliases: `showinterfaces`, `showifaces`, `si`*

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
