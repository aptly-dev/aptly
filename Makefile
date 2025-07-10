# Modern Makefile for aptly with improved tooling and practices

SHELL := /bin/bash
.DEFAULT_GOAL := help
.PHONY: help

# Version and build info
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOPATH := $(shell go env GOPATH)
BINPATH := $(GOPATH)/bin
GOOS := $(shell go env GOHOSTOS)
GOARCH := $(shell go env GOHOSTARCH)

# Tool versions
GOLANGCI_VERSION := v1.64.5
AIR_VERSION := v1.52.3
SWAG_VERSION := v1.16.4
GOVULNCHECK_VERSION := latest

# Build parameters
BINARY_NAME := aptly
BUILD_DIR := build
COVERAGE_DIR := coverage
COVERAGE_FILE := $(COVERAGE_DIR)/coverage.out

# Docker parameters
DOCKER_IMAGE := aptly/aptly
DOCKER_TAG := $(VERSION)

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_RED := \033[31m
COLOR_BLUE := \033[34m

##@ General

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

version: ## Show version
	@ci="" ; \
	if [ "`make -s releasetype`" = "ci" ]; then \
		ci=`TZ=UTC git show -s --format='+%cd.%h' --date=format-local:'%Y%m%d%H%M%S'`; \
	fi ; \
	if which dpkg-parsechangelog > /dev/null 2>&1; then \
		echo `dpkg-parsechangelog -S Version`$$ci; \
	else \
		echo `grep ^aptly -m1  debian/changelog | sed 's/.*(\([^)]\+\)).*/\1/'`$$ci ; \
	fi

releasetype: # Print release type: ci (on any branch/commit), release (on a tag)
	@reltype=ci ; \
	gitbranch=`git rev-parse --abbrev-ref HEAD` ; \
	if [ "$$gitbranch" = "HEAD" ] && [ "$$FORCE_CI" != "true" ]; then \
		gittag=`git describe --tags --exact-match 2>/dev/null` ;\
		if echo "$$gittag" | grep -q '^v[0-9]'; then \
			reltype=release ; \
		fi ; \
	fi ; \
	echo $$reltype

##@ Development

prepare: ## Prepare development environment
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Preparing development environment...$(COLOR_RESET)"
	$(GOMOD) download
	$(GOMOD) verify
	$(GOMOD) tidy -v
	@go generate ./...

dev-tools: ## Install development tools
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Installing development tools...$(COLOR_RESET)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION)
	@go install github.com/air-verse/air@$(AIR_VERSION)
	@go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)
	@go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Development tools installed$(COLOR_RESET)"

##@ Build

build: prepare swagger ## Build aptly binary
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Building aptly...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(COLOR_RESET)"

build-all: prepare swagger ## Build for all platforms
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Building for all platforms...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	# Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64
	# macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64
	# Windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Multi-platform build complete$(COLOR_RESET)"

install: build ## Install aptly to GOPATH/bin
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Installing aptly...$(COLOR_RESET)"
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(BINPATH)/
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Installed to $(BINPATH)/$(BINARY_NAME)$(COLOR_RESET)"

##@ Testing

test: prepare test-unit test-integration ## Run all tests

test-unit: prepare swagger etcd-install ## Run unit tests
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Running unit tests...$(COLOR_RESET)"
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_DIR)/unit.out -covermode=atomic ./...
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Unit tests complete$(COLOR_RESET)"

test-integration: prepare swagger etcd-install ## Run integration tests
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Running integration tests...$(COLOR_RESET)"
	@mkdir -p $(COVERAGE_DIR)
	# Download fixtures if needed
	@if [ ! -e ~/aptly-fixture-db ]; then \
		git clone https://github.com/aptly-dev/aptly-fixture-db.git ~/aptly-fixture-db/; \
	fi
	@if [ ! -e ~/aptly-fixture-pool ]; then \
		git clone https://github.com/aptly-dev/aptly-fixture-pool.git ~/aptly-fixture-pool/; \
	fi
	# Run system tests
	PATH=$(BINPATH):$$PATH python3 system/run.py --coverage-dir $(COVERAGE_DIR) $(TEST)
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Integration tests complete$(COLOR_RESET)"

test-race: ## Run tests with race detector
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Running tests with race detector...$(COLOR_RESET)"
	$(GOTEST) -race -short ./...
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Race detection complete$(COLOR_RESET)"

coverage: test ## Generate coverage report
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Generating coverage report...$(COLOR_RESET)"
	@mkdir -p $(COVERAGE_DIR)
	@go tool cover -html=$(COVERAGE_DIR)/unit.out -o $(COVERAGE_DIR)/coverage.html
	@go tool cover -func=$(COVERAGE_DIR)/unit.out
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Coverage report: $(COVERAGE_DIR)/coverage.html$(COLOR_RESET)"

benchmark: ## Run benchmarks
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Running benchmarks...$(COLOR_RESET)"
	$(GOTEST) -bench=. -benchmem ./deb ./files ./utils
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Benchmarks complete$(COLOR_RESET)"

##@ Code Quality

lint: dev-tools ## Run linters
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Running linters...$(COLOR_RESET)"
	@golangci-lint run --timeout=5m
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Linting complete$(COLOR_RESET)"

fmt: ## Format code
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Formatting code...$(COLOR_RESET)"
	@$(GOFMT) -w -s .
	@$(GOMOD) tidy
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Code formatted$(COLOR_RESET)"

vet: ## Run go vet
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Running go vet...$(COLOR_RESET)"
	@go vet ./...
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Vet complete$(COLOR_RESET)"

security: dev-tools ## Run security checks
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Running security checks...$(COLOR_RESET)"
	@govulncheck ./...
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Security check complete$(COLOR_RESET)"

##@ Dependencies

deps-update: ## Update dependencies
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Updating dependencies...$(COLOR_RESET)"
	@./scripts/update-deps.sh
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Dependencies updated$(COLOR_RESET)"

deps-check: ## Check for outdated dependencies
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Checking for outdated dependencies...$(COLOR_RESET)"
	@go list -u -m all | grep '\[' || echo "All dependencies are up to date!"

deps-graph: ## Generate dependency graph
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Generating dependency graph...$(COLOR_RESET)"
	@go mod graph | grep -v '@' | sort | uniq

##@ Documentation

swagger: swagger-install ## Generate Swagger documentation
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Generating Swagger documentation...$(COLOR_RESET)"
	@cp docs/swagger.conf.tpl docs/swagger.conf
	@echo "// @version $(VERSION)" >> docs/swagger.conf
	@swag init --parseDependency --parseInternal --markdownFiles docs --generalInfo docs/swagger.conf
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Swagger docs generated$(COLOR_RESET)"

swagger-install: ## Install swagger tools
	@test -f $(BINPATH)/swag || go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)

docs: swagger ## Generate all documentation
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Documentation generated$(COLOR_RESET)"

##@ Development Server

serve: dev-tools prepare ## Run development server with hot reload
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Starting development server...$(COLOR_RESET)"
	@cp debian/aptly.conf ~/.aptly.conf || true
	@sed -i.bak '/enable_swagger_endpoint/s/false/true/' ~/.aptly.conf || true
	@air -build.pre_cmd 'swag init -q --markdownFiles docs --generalInfo docs/swagger.conf' \
		-build.exclude_dir docs,system,debian,pgp/keyrings,pgp/test-bins,completion.d,man,deb/testdata,console,_man,systemd,obj-x86_64-linux-gnu \
		-- api serve -listen 0.0.0.0:3142

##@ Docker

docker-build: ## Build Docker image
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Building Docker image...$(COLOR_RESET)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -t $(DOCKER_IMAGE):latest .
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(COLOR_RESET)"

docker-push: ## Push Docker image
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Pushing Docker image...$(COLOR_RESET)"
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Docker image pushed$(COLOR_RESET)"

##@ Cleanup

clean: ## Clean build artifacts
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(BUILD_DIR) $(COVERAGE_DIR)
	@rm -f docs/docs.go docs/swagger.json docs/swagger.yaml docs/swagger.conf
	@rm -rf obj-* *.out *.test
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Clean complete$(COLOR_RESET)"

clean-deps: ## Clean dependency cache
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Cleaning dependency cache...$(COLOR_RESET)"
	@go clean -modcache
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Dependency cache cleaned$(COLOR_RESET)"

##@ CI/CD

ci: prepare lint test security ## Run CI pipeline
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ CI pipeline complete$(COLOR_RESET)"

release: clean build-all ## Prepare release artifacts
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Preparing release...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)/release
	@for file in $(BUILD_DIR)/$(BINARY_NAME)-*; do \
		base=$$(basename $$file); \
		tar -czf $(BUILD_DIR)/release/$$base.tar.gz -C $(BUILD_DIR) $$base; \
	done
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Release artifacts ready in $(BUILD_DIR)/release$(COLOR_RESET)"

##@ Utilities

etcd-install: ## Install etcd for testing
	@test -d /tmp/aptly-etcd || system/t13_etcd/install-etcd.sh

etcd-start: ## Start etcd
	@mkdir -p /tmp/aptly-etcd-data
	@system/t13_etcd/start-etcd.sh > /tmp/aptly-etcd-data/etcd.log 2>&1 &
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ etcd started$(COLOR_RESET)"

etcd-stop: ## Stop etcd
	@kill `cat /tmp/etcd.pid` 2>/dev/null || true
	@rm -f /tmp/etcd.pid
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ etcd stopped$(COLOR_RESET)"

azurite-start: ## Start Azurite (Azure Storage Emulator) for tests
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Starting Azurite...$(COLOR_RESET)"
	@azurite -l /tmp/aptly-azurite > ~/.azurite.log 2>&1 & \
	echo $$! > ~/.azurite.pid
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Azurite started (PID: $$(cat ~/.azurite.pid))$(COLOR_RESET)"

azurite-stop: ## Stop Azurite
	@echo -e "$(COLOR_YELLOW)$(COLOR_BOLD)Stopping Azurite...$(COLOR_RESET)"
	@-kill `cat ~/.azurite.pid` 2>/dev/null || true
	@rm -f ~/.azurite.pid
	@echo -e "$(COLOR_GREEN)$(COLOR_BOLD)✓ Azurite stopped$(COLOR_RESET)"

.PHONY: all build build-all install test test-unit test-integration test-race coverage benchmark \
	lint fmt vet security deps-update deps-check deps-graph docs swagger swagger-install serve \
	docker-build docker-push clean clean-deps ci release prepare dev-tools etcd-install etcd-start etcd-stop \
	azurite-start azurite-stop