#!/usr/bin/env bash

# shellcheck disable=SC1091
source "$(dirname "$(realpath "$0")")/common.sh"

cyber_step "Docker Build"

export DOCKER_BUILDKIT=1

cyber_log "Building multi-arch image noksa/gokeenapi:stable"
docker buildx build -t noksa/gokeenapi:stable --platform "linux/amd64,linux/arm64" --pull --push \
  --build-arg="GOKEENAPI_VERSION=stable" --build-arg="GOKEENAPI_BUILDDATE=$(date)" . -f Dockerfile

cyber_ok "Docker image pushed"
