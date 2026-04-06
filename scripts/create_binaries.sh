#!/usr/bin/env bash

# shellcheck disable=SC1091
source "$(dirname "$(realpath "$0")")/common.sh"

DIR="binaries"
VERSION="undefined"
BUILDDATE="$(date)"
while [[ $# -gt 0 ]]; do
  case $1 in
    --version)
      VERSION="${2}"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done

cyber_step "Create Binaries"
cyber_log "Version: ${CYBER_G}${VERSION}${CYBER_X}"

rm -rf ../"${DIR}"
mkdir -p ../"${DIR}"
pushd ../"${DIR}" >/dev/null
trap 'popd >/dev/null; cyber_trap' EXIT ERR

ARCH="amd64 arm64"
OS="linux darwin windows"

for A in $ARCH; do
  for O in $OS; do
    output="gokeenapi_${VERSION}_${A}_${O}"
    if [[ "$O" == "windows" ]]; then
      output="${output}.exe"
    fi
    CGO_ENABLED=0 GOARCH=$A GOOS=$O go build -ldflags "-X \"github.com/noksa/gokeenapi/internal/gokeenversion.version=${VERSION}\" -X \"github.com/noksa/gokeenapi/internal/gokeenversion.buildDate=${BUILDDATE}\"" -o "${output}" ../main.go
    cyber_log "Built ${CYBER_C}${O}-${A}${CYBER_X}"
  done
done

cyber_ok "All binaries created"
