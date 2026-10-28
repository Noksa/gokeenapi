# Local exporter monitoring

Run the current checkout's exporter together with Prometheus and Grafana. Docker Engine and Docker Compose v2.24+ are required. No separate Go installation is needed for the stack: Compose builds the exporter with the repository Dockerfile.

From the repository root:

```bash
make exporter-local-up EXPORTER_CONFIG=/path/to/exporter.yaml
make exporter-local-status
make exporter-local-logs
make exporter-local-down
```

`EXPORTER_CONFIG` may be absolute or relative to the repository root. Its containing directory is mounted read-only at `/etc/gokeenapi`; router paths in the exporter YAML must be relative paths within that directory (subdirectories are supported). The container uses your host UID/GID so private `0600` files remain readable without changing permissions. The exporter loads only connection settings, so route/domain source files are not needed.

For example, place both files in a private directory:

```yaml
# exporter.yaml
listen: 127.0.0.1:9479
routers:
  - name: home
    config: router.yaml
```

```yaml
# router.yaml
keenetic:
  url: https://router.example.net
  login: monitor
  password: replace-me
```

The stand passes `--listen 0.0.0.0:9479` inside the exporter container; the original YAML is unchanged. All host ports are published on `127.0.0.1`:

| Service | URL |
|---|---|
| Grafana | <http://127.0.0.1:3000/d/gokeenapi-keenetic> |
| Prometheus | <http://127.0.0.1:9090> |
| Exporter | <http://127.0.0.1:9479/metrics> |

Grafana provisions its datasource and dashboard automatically and opens the dashboard as its home page. Anonymous access has the Viewer role; this local stand has no login or dashboard editing UI. The rendered dashboard in `.runtime/<project>/dashboards` is generated from the canonical `grafana/keenetic.json`, with the provisioning datasource UID substituted. `make exporter-local-up` regenerates it after dashboard changes.

Prometheus scrapes every 15 seconds and keeps seven days of history. The exporter retains its configured polling intervals independently of scraping. Inspect collector success/freshness before interpreting measurements; a healthy exporter endpoint does not imply every router is reachable. Rate graphs need several polls, typically 2–5 minutes.

Every explicit `up` rebuilds and recreates the services, applying in-place edits to exporter/router YAML and provisioning files. Monitoring history remains in the named volumes. Stopping the stack preserves Prometheus/Grafana named volumes. `down`, `status` and `logs` do not need `EXPORTER_CONFIG`. To select another isolated stack, use the same `EXPORTER_PROJECT` for every command. Ports can be overridden:

```bash
make exporter-local-up EXPORTER_CONFIG=/path/to/exporter.yaml \
  EXPORTER_PROJECT=keenetic-lab \
  GRAFANA_PORT=13000 PROMETHEUS_PORT=19090 EXPORTER_PORT=19479
make exporter-local-down EXPORTER_PROJECT=keenetic-lab
```

`GOKEENAPI_KEENETIC_LOGIN` and `GOKEENAPI_KEENETIC_PASSWORD`, if set in the shell, are forwarded only to the exporter and override credentials for every router. Existing YAML credentials work without those variables. The stack never copies router passwords into generated files or images.

## Measure router load

The opt-in experiment requires Go and Entware SSH (`awk`, `usleep`). Any valid exporter YAML with exactly one router and all collectors enabled is supported (omitted settings use defaults). SSH must target that same router; the test does not verify API/SSH identity. Use normal intervals; stop any other host processes/external exporters polling that router first. The test requires exactly one running Compose exporter and verifies its project (`EXPORTER_PROJECT`, default `gokeenapi-exporter-local`). It temporarily stops that container and restores it through cleanup, including ordinary test interruption/failure. After a forced process kill, check the stack manually.

```bash
GOKEENAPI_LIVE_EXPORTER_CONFIG="$(pwd)/custom/exporter_lab.yaml" \
GOKEENAPI_LIVE_ROUTER_SSH=root@router.example \
GOKEENAPI_LIVE_REPORT="$(pwd)/custom/exporter-load-$(date +%Y%m%d-%H%M%S).json" \
go run github.com/onsi/ginkgo/v2/ginkgo --procs=1 \
  --focus='Live exporter router load' -v ./internal/exporter
```

Config and report paths must be absolute. Keep local configs and reports under gitignored `custom/` (or outside the checkout). The report must be a new file: existing files, symlinks and hardlinks are rejected before the stack is stopped. It is reserved with mode `0600`; a failed experiment may leave an empty report. Only the report basename is added to Ginkgo output; credentials and configuration contents are never serialized.

One persistent SSH session samples `/proc/stat` and `/proc/meminfo` every 250 ms without API requests. Phases: baseline 30 s, normal polling 120 s (including the initial collection of all collectors), cache-only scrapes at about 100/s for 30 s, recovery 30 s. The cache phase must add zero API requests. JSON records CPU, memory and cumulative request counts to correlate spikes with polling; console output includes mean/p95/maximum CPU. Used memory here uses Linux MemTotal, so absolute usage can differ from Keenetic; compare changes. Background traffic and the SSH sampler exist throughout; one run cannot establish the cause of every spike. Peaks shorter than 250 ms are averaged. Without these variables the test skips and never contacts a real router.
