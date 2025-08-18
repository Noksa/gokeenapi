#!/usr/bin/env bash

set -euo pipefail

export DOCKER_BUILDKIT=1

docker buildx build -t noksa/gokeenapi:stable --platform "linux/amd64,linux/arm64" --pull --push \
--build-arg="GOKEENAPI_VERSION=stable" --build-arg="GOKEENAPI_BUILDDATE=$(date)" . -f Dockerfile