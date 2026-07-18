#!/usr/bin/env bash

# shellcheck disable=SC1091
source "$(dirname "$(realpath "$0")")/common.sh"

TAG="${TAG:-stable}"

cyber_step "Docker Build"

export DOCKER_BUILDKIT=1

cyber_log "Building multi-arch image noksa/gokeenapi:${TAG}"
docker buildx build -t "noksa/gokeenapi:${TAG}" --platform "linux/amd64,linux/arm64" --pull --push \
  --build-arg="GOKEENAPI_VERSION=${TAG}" --build-arg="GOKEENAPI_BUILDDATE=$(date)" . -f Dockerfile

cyber_ok "Docker image pushed: noksa/gokeenapi:${TAG}"
