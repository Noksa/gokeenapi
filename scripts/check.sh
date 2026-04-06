#!/usr/bin/env bash

# shellcheck disable=SC1091
source "$(dirname "$(realpath "$0")")/common.sh"

cyber_step "Linting"

cyber_log "go mod tidy"
go mod tidy

cyber_log "go fmt"
go fmt ./...

cyber_log "go vet"
go vet ./...

cyber_log "go modernize"
go run golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest -fix ./...

cyber_log "golangci-lint"
golangci-lint run --timeout 15m --color=always

cyber_ok "All checks passed"
