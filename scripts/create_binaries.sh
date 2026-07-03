#!/usr/bin/env bash

# shellcheck disable=SC1091
source "$(dirname "$(realpath "$0")")/common.sh"

VERSION="undefined"
BUILDDATE="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
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

# If VERSION is still "undefined", try to derive from GITHUB_REF_NAME (CI tag)
if [[ "$VERSION" == "undefined" && -n "${GITHUB_REF_NAME:-}" ]]; then
  VERSION="${GITHUB_REF_NAME#v}"
fi

cyber_step "Create Release Archives"
cyber_log "Version: ${CYBER_G}${VERSION}${CYBER_X}"

rm -rf "${PROJECT_DIR}/generated"
mkdir -p "${PROJECT_DIR}/generated"
cd "${PROJECT_DIR}/generated"

LDFLAGS="-s -w -X \"github.com/noksa/gokeenapi/internal/gokeenversion.version=${VERSION}\" -X \"github.com/noksa/gokeenapi/internal/gokeenversion.buildDate=${BUILDDATE}\""

TAR="tar"
if command -v gtar &>/dev/null; then
  TAR="gtar"
fi

ALL_ARCH="amd64 arm64"
ALL_OS="linux darwin windows"

# Support TARGET variable to build a single platform (e.g., TARGET=linux/amd64)
if [ -n "${TARGET:-}" ]; then
  IFS='/' read -r ALL_OS ALL_ARCH <<< "$TARGET"
  cyber_log "Target: ${CYBER_C}${TARGET}${CYBER_X}"
fi

for A in $ALL_ARCH; do
  for O in $ALL_OS; do
    output="gokeenapi"
    if [[ "$O" == "windows" ]]; then
      output="gokeenapi.exe"
    fi

    cyber_log "Building ${CYBER_C}${O}/${A}${CYBER_X}"
    CGO_ENABLED=0 GOARCH=$A GOOS=$O go build -ldflags "${LDFLAGS}" -o "${output}" "${PROJECT_DIR}/main.go"

    archive="gokeenapi_${VERSION}_${O}_${A}.tar.gz"
    $TAR -czf "${archive}" "${output}"
    rm -f "${output}"

    cyber_ok "Created ${CYBER_G}${archive}${CYBER_X}"
  done
done

cyber_ok "All archives created in ${CYBER_G}generated/${CYBER_X}"
