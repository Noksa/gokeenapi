#!/usr/bin/env bash

set -euo pipefail
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

rm -rf ../"${DIR}"
mkdir -p ../"${DIR}"
pushd ../"${DIR}" >/dev/null
trap 'popd >/dev/null' exit err
ARCH="amd64 arm64"
OS="linux darwin windows"
echo "Building ${VERSION} version for all platforms..."
for A in $ARCH; do
  for O in $OS; do
    output="gokeenapi_${VERSION}_${A}_${O}"
    if [[ "$O" == "windows" ]]; then
      output="${output}.exe"
    fi
    CGO_ENABLED=0 GOARCH=$A GOOS=$O go build -ldflags "-X \"github.com/noksa/gokeenapi/internal/gokeenversion.version=${VERSION}\" -X \"github.com/noksa/gokeenapi/internal/gokeenversion.buildDate=${BUILDDATE}\"" -o "${output}" ../main.go
    echo "Built for ${O}-${A}"
#    gtar -czvf "gokeenapi_${VERSION}_${O}_${A}.tar.gz" "${output}" >/dev/null
#    rm -rf "${output}"
  done
done
