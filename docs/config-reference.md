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
| `tls_skip_verify` | bool | ❌ | `false` | Disable TLS certificate verification for the router connection only. External URL lists still require valid certificates. Enable when the router uses a self-signed certificate. |
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

Each entry is one of:

- a **group object**
- a **string path** to a `.yaml` file that contains a `groups:` list (shared groups across router configs)
- an **import object** with `file:` plus optional `interfaceId` override

**Group object fields:**

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | ✅ | Unique name for the object-group on the router; whitespace and router command separators are rejected. |
| `interfaceId` | string | ✅ | Target interface for routing matched domain traffic (e.g. `Wireguard0`). Run `show-interfaces` to list available IDs. |
| `domain-file` | list of strings | ❌ | Paths to local `.txt` files with one domain, IPv4 address, or IPv4 subnet per line (lines starting with `#` are comments). A `.yaml`/`.yml` path is expanded to the `domain-file` list it contains. |
| `domain-url` | list of strings | ❌ | Remote URLs serving lists in the same format. A `.yaml`/`.yml` path is expanded to the `domain-url` list it contains. |
| `file` | string | ❌ | Path to a YAML file with a `groups:` list. When set, this entry is an import, not a group. Combined with `interfaceId`, every imported group uses that interface. |

At least one of `domain-file` or `domain-url` is required per real group (not per import).

`add-dns-routing` sorts and deduplicates all entries in a group, then places at most 300 in each router object-group. For example, 397 domains in `facebook` create `facebook` (300) and `facebook-2` (97), both routed through the configured interface. If the source changes, the command reconciles both groups; when it shrinks to 300 or fewer, it removes the obsolete `facebook-2` route and object-group. An empty source leaves existing groups unchanged. Names such as `facebook-2` are reserved for automatic chunks of `facebook`. Existing suffix groups are checked for contiguous names, plausible domain partitioning, and matching routes before updates or cleanup. An empty suffix group left by an interrupted run can be resumed. Changing `interfaceId` migrates both routes.

Domain entries are normalized to lowercase ASCII with IDNA; a trailing dot is removed. IPv4 addresses and subnets are supported; subnets are normalized to the network address. `full:` and `domain:` rules are accepted; `regexp:`, `keyword:`, and `include:` rules are skipped. A nonempty source with no usable entries is an error. Relative local paths in `domain-file` and `domain-url` inside imported groups files are resolved from the imported file's directory. A duplicate dns-proxy route for a group stops reconciliation until the ambiguous router state is resolved.

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

      # Import shared groups from a file (as-is)
      - common/shared_groups.yaml

      # Import the same file but force every group onto Wireguard1
      - file: common/shared_groups.yaml
        interfaceId: Wireguard1
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
| `--dry-run` | ❌ (`update-awg` only) | Print a unified diff of pending changes and do not apply them. |

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

---

## See also

- [config_example.yaml](../config_example.yaml) — annotated router config
- [SCHEDULER.md](../SCHEDULER.md) — `scheduler.yaml` (separate file; not part of router `config.yaml`)
- [README.md](../README.md) — environment variables and command tutorials

## Prometheus exporter

`gokeenapi exporter --config exporter.yaml` (alias `metrics`) loads a separate exporter YAML. `GOKEENAPI_CONFIG` is also supported; an explicit `--config` wins. It references ordinary router YAML files but reads only their `keenetic` connection settings, without expanding routes, domain files, or URLs. It performs read-only RCI operations and never saves router configuration.

| Field | Default | Meaning |
|---|---|---|
| `listen` | `127.0.0.1:9479` | HTTP listening address; endpoint is `GET /metrics` |
| `timeout` | `10s` | Deadline for an entire router collection cycle; range `1s`–`5m` |
| `max_concurrent_routers` | `4` | Simultaneous router cycles; range 1–64 |
| `max_backoff` | `5m` | Exponential failure retry cap; a collector never retries faster than its normal interval; range `1s`–`24h` |
| `routers[].name` | required | Unique stable name, exposed as the `router` label |
| `routers[].config` | required | Router YAML path, relative to the exporter YAML |
| `routers[].interfaces` | automatic | Optional list of case-sensitive interface IDs to monitor |
| `collectors.<name>.enabled` | `true` | Enable or disable one collection group |
| `collectors.<name>.interval` | below | Poll interval; range `5s`–`24h` |
| `collectors.<name>.max_age` | twice the interval | Maximum lifetime of a successful snapshot; at least its interval, at most `168h` |

Groups: `system`, `dns`, and `interfaces` default to `30s`; `hosts` to `1m`; `routing` to `5m`; `info` to `1h`. Unknown exporter fields and collector names are rejected. At least one router and one enabled group are required. Router credentials can use `GOKEENAPI_KEENETIC_LOGIN` and `GOKEENAPI_KEENETIC_PASSWORD`; these overrides apply to **all** referenced routers. Use separate YAML credentials when routers require different accounts, and keep those files private. Router URLs must be HTTP(S) origins without embedded credentials, paths, queries, or fragments. A positive `keenetic.timeout` can shorten the exporter cycle's HTTP request timeout.

Without an interface list, the exporter selects interfaces with an IP address, a default gateway, or type `Wireguard`. Explicit selection is recommended to control the command budget. Interface discovery and all selected interface statistics are batched where supported; batches contain at most 100 read commands. Router configuration paths use GET. Firmware without batch support falls back to individual GETs. Routing state uses three independent collectors (`static_routes`, `dns_groups`, `dns_routes`), so unsupported DNS-routing does not hide static routes.

Polling is independent of scraping. At most one cycle per router is active; late cycles do not accumulate. Router starts are spread across up to 30 seconds, with bounded concurrency. Sessions and connections are reused; authentication is repeated only when required. Failures double the failed collector's interval up to `max_backoff`, with a floor at its normal interval. Other collectors keep their schedules. No synthetic DNS queries, full running-config reads, or per-client details are collected.

`GET /metrics` serves the latest successful snapshot until its `max_age`; a failed refresh does not turn measurements into zeroes. Expired measurements disappear. `keenetic_collector_success` reports the last attempt, while `keenetic_collector_fresh` reports snapshot freshness. Check both, plus `keenetic_collector_last_success_timestamp_seconds`, when alerting. Before the first successful read, measurements and the last-success timestamp are absent. Health metrics and `keenetic_api_requests_total` remain available even when a router is offline.

| Metrics | Meaning |
|---|---|
| `keenetic_dns_requests_total`, `keenetic_dns_forwarded_requests_total`, `keenetic_dns_cache_hits_total` | Native DNS counters per `proxy`; use `rate(...[5m])` for QPS and `increase(...[24h])` for daily counts |
| `keenetic_dns_upstream_requests_total`, `keenetic_dns_upstream_responses_total`, `keenetic_dns_upstream_nxdomain_total` | Native upstream counters, labelled by `proxy`, `server`, and `port` |
| `keenetic_dns_upstream_response_duration_seconds` | Router-reported upstream average, not a histogram or percentile |
| `keenetic_hosts`, `keenetic_hosts_active`, `keenetic_hosts_registered` | Aggregate host counts; optional activity/registration counts appear only if the response supplies the needed fields |
| `keenetic_static_routes_count`, `keenetic_static_routes{interface}` | Total and per-interface counts of configured static routes |
| `keenetic_dns_groups`, `keenetic_dns_group_entries{group,entry_type}` | Group count and configured domain/IP entry counts; **not** the number of resolved IPs or matched connections |
| `keenetic_dns_routes_count`, `keenetic_dns_routes{group,interface}` | Counts of configured system DNS-routing rules |
| `keenetic_interface_up`, `keenetic_interface_connected` | Administrative state and operational connectivity of selected interfaces |
| `keenetic_interface_receive_bytes_total`, `keenetic_interface_transmit_bytes_total` | Native byte counters; use `8 * rate(...[5m])` for bits/s |
| `keenetic_interface_receive_packets_total`, `keenetic_interface_transmit_packets_total` | Packet counters; corresponding `*_errors_total` and `*_dropped_total` are exported where present |
| `keenetic_cpu_load_ratio`, `keenetic_uptime_seconds`, `keenetic_memory_{total,free,cache,buffers}_bytes` | CPU utilization, uptime, and memory in base units |
| `keenetic_connections_capacity`, `keenetic_connections_free` | Connection tracking capacity and free entries |
| `keenetic_router_info{model,version}` | Router identity, value 1 |
| `keenetic_collector_success`, `keenetic_collector_fresh`, `keenetic_collector_duration_seconds`, `keenetic_collector_errors_total` | Collector health, freshness, cycle duration through that collector, and failure count |
| `keenetic_collector_last_attempt_timestamp_seconds`, `keenetic_collector_last_success_timestamp_seconds` | Unix timestamps, labelled by `collector` |
| `keenetic_api_requests_total` | Actual HTTP attempts per router, including authentication and retries |

All metrics above have a `router` label. Counter resets on router/proxy restart are handled by Prometheus `rate`/`increase`. Keep scrape intervals at or below poll intervals, and use rate windows spanning several polls. No historical data is stored by the exporter; Prometheus stores history.

Example Prometheus scrape job:

```yaml
scrape_configs:
  - job_name: gokeenapi
    scrape_interval: 15s
    static_configs:
      - targets: ["127.0.0.1:9479"]
```

The example assumes Prometheus runs on the same host. Set a reachable `listen` address and target for other hosts or containers; the endpoint has no built-in authentication. Import [the Grafana dashboard](../grafana/keenetic.json) and select the Prometheus datasource. See [exporter_example.yaml](../exporter_example.yaml).

Router responses are limited to 8 MiB of decompressed data per HTTP request. Larger responses become collection errors. Dashboard stat panels use Instant queries; current DNS QPS and cache ratio also require successful, fresh DNS collection.

`exporter --listen <host:port>` overrides the YAML `listen` value for this process only; numeric ports 1–65535 are accepted. The [local Compose stack](../docker/infra-exporter-local/README.md) uses this to make the exporter reachable by Prometheus inside Docker without changing the config.
