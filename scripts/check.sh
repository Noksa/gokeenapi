#!/usr/bin/env bash

set -euo pipefail
echo "Linting..."
go mod tidy
go fmt ./...
go vet ./...
golangci-lint run --timeout 15m --color=always
echo "Checked!"