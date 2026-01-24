#!/usr/bin/env bash

set -euo pipefail
echo "Linting..."
go mod tidy
go fmt ./...
go vet ./...
go run golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest -fix ./...
golangci-lint run --timeout 15m --color=always
echo "Checked!"