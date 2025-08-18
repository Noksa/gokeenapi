SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec
ROOT_DIR = $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

.DEFAULT_GOAL = help

.PHONY: help
help: ## Show help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: lint
lint: ## Lint me
	@chmod +x ./scripts/check.sh
	@./scripts/check.sh

.PHONY: build
build: ## Build
	@chmod +x ./scripts/build.sh
	@./scripts/build.sh