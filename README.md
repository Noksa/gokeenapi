<div align="center">

# 🚀 gokeenapi

**Automate your Keenetic router management with ease**

<p align="center">
  <video src="https://github.com/user-attachments/assets/404e89cc-4675-42c4-ae93-4a0955b06348" width="100%"></video>
</p>

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Docker Pulls](https://img.shields.io/docker/pulls/noksa/gokeenapi)](https://hub.docker.com/r/noksa/gokeenapi)
[![GitHub release](https://img.shields.io/github/release/Noksa/gokeenapi.svg)](https://github.com/Noksa/gokeenapi/releases)

*Tired of clicking through Keenetic web interface? Automate your Keenetic router management with simple CLI commands.*

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

### Environment Variables

For security, you can store sensitive credentials as environment variables instead of in the config file:

- `GOKEENAPI_KEENETIC_LOGIN` - Router admin login
- `GOKEENAPI_KEENETIC_PASSWORD` - Router admin password

---

## 🔧 Supported Routers

`gokeenapi` has been tested with the following Keenetic router models:

- **Keenetic Start**
- **Keenetic Viva** 
- **Keenetic Giga**

Since the utility works with Keenetic Start (the most affordable model in the lineup), it should be compatible with all Keenetic router models.

---

## 🎬 Video Demos

Check out these video demonstrations (in Russian) to see `gokeenapi` in action:

*   [Routes Management](https://www.youtube.com/watch?v=lKX74btFypY)

---

### 📚 Commands

Here are some of the things you can do with `gokeenapi`. For a full list of commands and options, use the `--help` flag.

```shell
./gokeenapi --help
```

#### `show-interfaces`

*Aliases: `showinterfaces`, `showifaces`, `si`*

Displays all available interfaces on your Keenetic router.

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

Execute custom Keenetic CLI commands directly on your router.

```shell
# Show system information
./gokeenapi exec --config my_config.yaml show version

# Display interface statistics
./gokeenapi exec --config my_config.yaml show interface

# Show routing table
./gokeenapi exec --config my_config.yaml show ip route
```

#### `scheduler`

*Aliases: `schedule`, `sched`*

Run scheduled tasks based on scheduler configuration file. Executes commands at specified intervals or fixed times.

```shell
# Run scheduler with config
./gokeenapi scheduler --config scheduler.yaml
```

Example scheduler configuration:

```yaml
tasks:
  - name: "Update routes every 3 hours"
    commands:
      - add-routes
    configs:
      - /path/to/router1.yaml
      - /path/to/router2.yaml
    interval: "3h"
  
  - name: "Refresh routes daily"
    commands:
      - delete-routes
      - add-routes
    configs:
      - /path/to/router1.yaml
    times:
      - "02:00"
```

See [scheduler_example.yaml](scheduler_example.yaml) for more examples.

---

### 🤝 Contributing

Contributions are welcome! If you have any ideas, suggestions, or bug reports, please open an issue or create a pull request.

---

### 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
