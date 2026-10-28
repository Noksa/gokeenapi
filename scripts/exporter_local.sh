#!/usr/bin/env bash

# shellcheck disable=SC1091
source "$(dirname "$(realpath "$0")")/common.sh"

action="${1:-}"
case "$action" in
  up|down|logs|status) ;;
  *) cyber_err "Usage: $0 {up|down|logs|status}"; exit 1 ;;
esac

export EXPORTER_PROJECT="${EXPORTER_PROJECT:-gokeenapi-exporter-local}"
if [[ ! "$EXPORTER_PROJECT" =~ ^[a-z0-9][a-z0-9_-]*$ ]]; then
  cyber_err "EXPORTER_PROJECT must contain lowercase letters, digits, underscores or hyphens"
  exit 1
fi

export GOKEENAPI_ROOT="$PROJECT_DIR"
EXPORTER_UID="$(id -u)"
EXPORTER_GID="$(id -g)"
export EXPORTER_UID EXPORTER_GID
export EXPORTER_RUNTIME_DIR="${PROJECT_DIR}/docker/infra-exporter-local/.runtime/${EXPORTER_PROJECT}"
# Down/logs/status need no private configuration. Compose only uses these defaults
# to parse its service definitions; it identifies existing containers by project.
export EXPORTER_CONFIG_DIR="$PROJECT_DIR"
export EXPORTER_CONFIG_NAME=exporter_example.yaml

if [[ "$action" == up ]]; then
  if [[ -z "${EXPORTER_CONFIG:-}" ]]; then
    cyber_err "Set EXPORTER_CONFIG to your exporter YAML path"
    exit 1
  fi
  config_path="$EXPORTER_CONFIG"
  if [[ "$config_path" != /* ]]; then config_path="${PROJECT_DIR}/${config_path}"; fi
  if [[ ! -f "$config_path" || ! -r "$config_path" ]]; then
    cyber_err "EXPORTER_CONFIG must be an existing readable file"
    exit 1
  fi
  config_path="$(realpath "$config_path")"
  EXPORTER_CONFIG_DIR="$(dirname "$config_path")"
  EXPORTER_CONFIG_NAME="$(basename "$config_path")"
  export EXPORTER_CONFIG_DIR EXPORTER_CONFIG_NAME
  mkdir -p "${EXPORTER_RUNTIME_DIR}/dashboards"
  chmod 755 "${EXPORTER_RUNTIME_DIR}/dashboards"
  # Grafana provisioning does not perform the import wizard's __inputs substitution.
  # Generate a local view from the canonical dashboard, keeping one tracked source.
  dashboard_tmp="$(mktemp "${EXPORTER_RUNTIME_DIR}/dashboards/.keenetic.XXXXXX")"
  trap 'rm -f -- "$dashboard_tmp"' EXIT
  # shellcheck disable=SC2016 # Replace the literal Grafana import placeholder.
  sed 's/${DS_PROMETHEUS}/gokeenapi-prometheus/g' \
    "${PROJECT_DIR}/grafana/keenetic.json" > "$dashboard_tmp"
  chmod 644 "$dashboard_tmp"
  mv -f -- "$dashboard_tmp" "${EXPORTER_RUNTIME_DIR}/dashboards/keenetic.json"
fi

compose=(docker compose --project-name "$EXPORTER_PROJECT" --file "${PROJECT_DIR}/docker/infra-exporter-local/compose.yaml")
case "$action" in
  up)
    cyber_step "Starting local exporter infrastructure"
    # Config file contents are not part of Compose's service hash. Recreate to
    # apply edited router lists, credentials, and intervals on every explicit up.
    "${compose[@]}" up --detach --build --force-recreate --wait --wait-timeout 120
    cyber_ok "Grafana: http://$("${compose[@]}" port grafana 3000)/d/gokeenapi-keenetic"
    cyber_log "Prometheus: http://$("${compose[@]}" port prometheus 9090)"
    cyber_log "Exporter: http://$("${compose[@]}" port exporter 9479)/metrics"
    ;;
  down)
    cyber_step "Stopping local exporter infrastructure"
    "${compose[@]}" down
    cyber_ok "Stopped; monitoring history is preserved"
    ;;
  logs) "${compose[@]}" logs --follow --tail 100 ;;
  status) "${compose[@]}" ps ;;
esac
