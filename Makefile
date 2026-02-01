SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec
ROOT_DIR = $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

.DEFAULT_GOAL = help

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Cyberpunk DevOps Theme - cache library locally for performance
CYBER_CACHE := .cyber.sh
CYBER_URL := https://raw.githubusercontent.com/Noksa/install-scripts/main/cyberpunk.sh

# Fetch cyberpunk library once (cached)
$(CYBER_CACHE):
	@curl -s $(CYBER_URL) > $(CYBER_CACHE)

##@ General

.PHONY: help
help: $(CYBER_CACHE) ## Display this help
	@source $(CYBER_CACHE) && { \
		echo ""; \
		echo -e "$${CYBER_D}╔═══════════════════════════════════════════════════════════════╗$${CYBER_X}"; \
		echo -e "$${CYBER_D}║$${CYBER_X}  $${CYBER_M}⚡$${CYBER_X} $${CYBER_B}$${CYBER_C}GOKEENAPI$${CYBER_X}"; \
		echo -e "$${CYBER_D}║$${CYBER_X}  $${CYBER_W}Keenetic Router Automation CLI$${CYBER_X}"; \
		echo -e "$${CYBER_D}╚═══════════════════════════════════════════════════════════════╝$${CYBER_X}"; \
	}
	@awk 'BEGIN {FS = ":.*##"; printf "\n\033[36mUsage:\033[0m make \033[35m<target>\033[0m\n\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m \033[37m%s\033[0m\n", $$1, $$2 } /^##@/ { printf "\n\033[35m⚡ %s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: lint
lint: $(CYBER_CACHE) ## Run linter via scripts/check.sh
	@source $(CYBER_CACHE) && { \
		echo ""; \
		echo -e "$${CYBER_D}╔═══════════════════════════════════════════════════════════════╗$${CYBER_X}"; \
		echo -e "$${CYBER_D}║$${CYBER_X}  $${CYBER_M}🔍$${CYBER_X} $${CYBER_B}$${CYBER_C}CODE ANALYSIS$${CYBER_X}"; \
		echo -e "$${CYBER_D}╚═══════════════════════════════════════════════════════════════╝$${CYBER_X}"; \
		cyber_step "Running linter..."; \
	}
	@chmod +x ./scripts/check.sh
	@./scripts/check.sh
	@source $(CYBER_CACHE) && cyber_ok "Lint passed"

##@ Testing

.PHONY: test
test: $(CYBER_CACHE) ## Run tests
	@source $(CYBER_CACHE) && { \
		echo ""; \
		echo -e "$${CYBER_D}╔═══════════════════════════════════════════════════════════════╗$${CYBER_X}"; \
		echo -e "$${CYBER_D}║$${CYBER_X}  $${CYBER_M}🧪$${CYBER_X} $${CYBER_B}$${CYBER_C}RUNNING TESTS$${CYBER_X}"; \
		echo -e "$${CYBER_D}╚═══════════════════════════════════════════════════════════════╝$${CYBER_X}"; \
		cyber_step "Executing test suite..."; \
	}
	@go test ./... -v
	@source $(CYBER_CACHE) && cyber_ok "All tests passed"

.PHONY: test-coverage
test-coverage: $(CYBER_CACHE) ## Run tests with coverage
	@source $(CYBER_CACHE) && { \
		echo ""; \
		echo -e "$${CYBER_D}╔═══════════════════════════════════════════════════════════════╗$${CYBER_X}"; \
		echo -e "$${CYBER_D}║$${CYBER_X}  $${CYBER_M}📊$${CYBER_X} $${CYBER_B}$${CYBER_C}TEST COVERAGE$${CYBER_X}"; \
		echo -e "$${CYBER_D}╚═══════════════════════════════════════════════════════════════╝$${CYBER_X}"; \
		cyber_step "Running tests with coverage..."; \
	}
	@go test ./... -v -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@source $(CYBER_CACHE) && cyber_ok "Coverage report generated: coverage.html"

##@ Build

.PHONY: build
build: $(CYBER_CACHE) lint ## Build (includes linting)
	@source $(CYBER_CACHE) && { \
		echo ""; \
		echo -e "$${CYBER_D}╔═══════════════════════════════════════════════════════════════╗$${CYBER_X}"; \
		echo -e "$${CYBER_D}║$${CYBER_X}  $${CYBER_M}🔨$${CYBER_X} $${CYBER_B}$${CYBER_C}BUILD$${CYBER_X}"; \
		echo -e "$${CYBER_D}╚═══════════════════════════════════════════════════════════════╝$${CYBER_X}"; \
		cyber_step "Building binary..."; \
	}
	@chmod +x ./scripts/build.sh
	@./scripts/build.sh
	@source $(CYBER_CACHE) && cyber_ok "Build complete"

.PHONY: binaries
binaries: $(CYBER_CACHE) lint ## Build release binaries (VERSION=<version>)
	@source $(CYBER_CACHE) && { \
		echo ""; \
		echo -e "$${CYBER_D}╔═══════════════════════════════════════════════════════════════╗$${CYBER_X}"; \
		echo -e "$${CYBER_D}║$${CYBER_X}  $${CYBER_M}📦$${CYBER_X} $${CYBER_B}$${CYBER_C}RELEASE BINARIES$${CYBER_X}"; \
		echo -e "$${CYBER_D}╠═══════════════════════════════════════════════════════════════╣$${CYBER_X}"; \
		echo -e "$${CYBER_D}║$${CYBER_X}  $${CYBER_W}Version$${CYBER_X} $${CYBER_C}→$${CYBER_X} $${CYBER_G}$(VERSION)$${CYBER_X}"; \
		echo -e "$${CYBER_D}╚═══════════════════════════════════════════════════════════════╝$${CYBER_X}"; \
		cyber_step "Creating release binaries..."; \
	}
	@chmod +x ./scripts/create_binaries.sh
	@cd ./scripts && ./create_binaries.sh --version $(VERSION)
	@source $(CYBER_CACHE) && cyber_ok "Binaries created"

##@ Docker

.PHONY: docker-build-test
docker-build-test: $(CYBER_CACHE) ## Build Docker image for testing (no push)
	@source $(CYBER_CACHE) && { \
		echo ""; \
		echo -e "$${CYBER_D}╔═══════════════════════════════════════════════════════════════╗$${CYBER_X}"; \
		echo -e "$${CYBER_D}║$${CYBER_X}  $${CYBER_M}🐳$${CYBER_X} $${CYBER_B}$${CYBER_C}DOCKER BUILD$${CYBER_X}"; \
		echo -e "$${CYBER_D}╠═══════════════════════════════════════════════════════════════╣$${CYBER_X}"; \
		echo -e "$${CYBER_D}║$${CYBER_X}  $${CYBER_W}Image$${CYBER_X} $${CYBER_C}→$${CYBER_X} $${CYBER_G}gokeenapi-test:local$${CYBER_X}"; \
		echo -e "$${CYBER_D}╚═══════════════════════════════════════════════════════════════╝$${CYBER_X}"; \
		cyber_step "Building docker image..."; \
	}
	@docker build -t gokeenapi-test:local \
		--build-arg GOKEENAPI_VERSION=test \
		--build-arg GOKEENAPI_BUILDDATE="$$(date)" \
		-f Dockerfile .
	@source $(CYBER_CACHE) && cyber_ok "Docker image built successfully"

##@ Maintenance

.PHONY: cyber-update
cyber-update: ## Update cached cyberpunk theme library
	@rm -f $(CYBER_CACHE)
	@curl -s $(CYBER_URL) > $(CYBER_CACHE)
	@source $(CYBER_CACHE) && cyber_ok "Cyberpunk theme updated"

.PHONY: clean
clean: $(CYBER_CACHE) ## Clean build artifacts and caches
	@source $(CYBER_CACHE) && { \
		echo ""; \
		echo -e "$${CYBER_Y}╔═══════════════════════════════════════════════════════════════╗$${CYBER_X}"; \
		echo -e "$${CYBER_Y}║$${CYBER_X}  $${CYBER_Y}🧹 CLEANING BUILD ARTIFACTS$${CYBER_X}"; \
		echo -e "$${CYBER_Y}╚═══════════════════════════════════════════════════════════════╝$${CYBER_X}"; \
		cyber_step "Removing build artifacts..."; \
	}
	@rm -f coverage.out coverage.html
	@rm -f gokeenapi gokeenapiw
	@source $(CYBER_CACHE) && cyber_ok "Clean complete"
