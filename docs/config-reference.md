# 📋 Config Reference

Complete reference for all fields in the `config.yaml` file. See [config_example.yaml](../config_example.yaml) for a fully annotated example.

> 🇷🇺 [Русская версия](config-reference-ru.md)

## Table of Contents

- [`keenetic` — Router connection](#keenetic--router-connection)
- [`dataDir` — Data directory](#datadir--data-directory)
- [`routes` — Static routes](#routes--static-routes)
- [`dns.records` — Static DNS records](#dnsrecords--static-dns-records)
- [`dns.routes.groups` — DNS-routing groups](#dnsroutesgroups--dns-routing-groups)
- [`add-awg` / `update-awg` — WireGuard commands](#add-awg--update-awg--wireguard-commands)
- [`logs` — Logging](#logs--logging)
- [`cache` — Caching](#cache--caching)

---

## `keenetic` — Router connection

Required by every command.

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `url` | string | ✅ | — | Router URL. Accepts IP address or KeenDNS hostname. Supports `http://` and `https://`. Example: `http://192.168.1.1` |
| `login` | string | ✅ | — | Router admin username. Overridden by `GOKEENAPI_KEENETIC_LOGIN` env var. |
| `password` | string | ✅ | — | Router admin password. Overridden by `GOKEENAPI_KEENETIC_PASSWORD` env var. |
| `tls_skip_verify` | bool | ❌ | `false` | Disable TLS certificate verification. Enable when the router uses a self-signed certificate. |
| `timeout` | duration | ❌ | `30s` | HTTP request timeout for router API calls. Increase for routers with large IP route tables where requests (e.g. fetching static routes) exceed the default. Accepts Go duration strings: `30s`, `1m`, `2m`. |

---

## `dataDir` — Data directory

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `dataDir` | string | ❌ | system default | Custom directory for application data files. Automatically set to `/etc/gokeenapi` when `GOKEENAPI_INSIDE_DOCKER` env var is present. |

---

## `routes` — Static routes

Used by: `add-routes`, `delete-routes`.

A list of per-interface routing configurations.

| Field | Type | Required | Description |
|---|---|---|---|
| `interfaceId` | string | ✅ | Target interface ID (e.g. `Wireguard0`). Run `show-interfaces` to list available IDs. |
| `bat-file` | list of strings | ❌ | Paths to local `.bat` files with `route add` commands. A `.yaml`/`.yml` path is expanded to the `bat-file` list it contains. Relative paths are resolved from the config file's directory. |
| `bat-url` | list of strings | ❌ | Remote URLs serving `.bat` files. A `.yaml`/`.yml` path is expanded to the `bat-url` list it contains. |

At least one of `bat-file` or `bat-url` should be provided per entry.

---

## `dns.records` — Static DNS records

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

---

## `dns.routes.groups` — DNS-routing groups

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

---

## `add-awg` / `update-awg` — WireGuard commands

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

---

## `logs` — Logging

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `debug` | bool | ❌ | `false` | Enable debug-level logging. Also controllable with the `--debug` global flag. |

---

## `cache` — Caching

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `urlTtl` | duration | ❌ | `1m` | How long to cache content downloaded from `bat-url` and `domain-url`. Accepts Go duration strings: `30s`, `5m`, `1h`. |
