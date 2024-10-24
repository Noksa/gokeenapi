#!/usr/bin/env bash

# All required fields are set in yaml configs including passwords and keenetic interface id
# Using this simple script it is possible to refresh routes across all keenetic routers at once if they work with KeenDNS
# Specify --force flag to run delete-routes before running add-routes

set -euo pipefail

IMAGE="noksa/gokeenapi:stable"
FORCE="n"

while [[ $# -gt 0 ]]; do
  case $1 in
    --force)
      FORCE="y"
      shift
      ;;
    --image)
      IMAGE="${2}"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

echo "Pulling ${IMAGE}..."
docker pull "${IMAGE}"

for file in config_*.yaml; do
  if [[ "${file}" == "config_example.yaml" ]]; then
    continue
  fi
  f="$(realpath "${file}")"
  echo "Running gokeenapi with ${f} config file"
  if [[ "${FORCE}" == "y" ]]; then
    docker run --rm -ti -v "${f}":"/config.yaml" "${IMAGE}" delete-routes --config /config.yaml
  fi
  docker run --rm -ti -v "${f}":"/config.yaml" "${IMAGE}" add-routes --config /config.yaml
done