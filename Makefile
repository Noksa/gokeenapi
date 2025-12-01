SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec
ROOT_DIR = $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

.DEFAULT_GOAL = help

.PHONY: help
help: ## Show help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: test
test: ## Run tests
	go test ./... -v

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

.PHONY: lint
lint: ## Lint me
	@chmod +x ./scripts/check.sh
	@./scripts/check.sh

.PHONY: build
build: lint ## Build
	@chmod +x ./scripts/build.sh
	@./scripts/build.sh

.PHONY: binaries
binaries: lint ## Build
	@chmod +x ./scripts/create_binaries.sh
	@cd ./scripts && ./create_binaries.sh --version ${VERSION}

.PHONY: docker-build-test
docker-build-test: ## Build Docker image for testing (no push)
	docker build -t gokeenapi-test:local \
		--build-arg GOKEENAPI_VERSION=test \
		--build-arg GOKEENAPI_BUILDDATE="$$(date)" \
		-f Dockerfile .