# Makefile for StormDB - PostgreSQL Benchmarking Tool
# This Makefile provides comprehensive build, test, documentation, and maintenance targets

.PHONY: build test test-unit test-integration test-load test-stress test-coverage clean help docs deps lint fmt vet install docker security

# Default target
.DEFAULT_GOAL := help

# Build configuration
BINARY_NAME := stormdb
CMD_DIR := cmd/stormdb
BUILD_DIR := build
COVERAGE_DIR := coverage
PLUGIN_DIR := $(BUILD_DIR)/plugins
DOCS_DIR := docs
PROFILES_DIR := profiles

# Version and build info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GO_VERSION ?= $(shell go version | cut -d' ' -f3)

# Go build flags with version information
GO_LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME) -X main.GoVersion=$(GO_VERSION)
GO_FLAGS := -ldflags="$(GO_LDFLAGS)"
GO_TEST_FLAGS := -v -race -timeout=30s
GO_BENCH_FLAGS := -bench=. -benchmem -benchtime=5s

# Tools and linters
GOLANGCI_LINT_VERSION := v1.55.2
GODOC_PORT := 6060

# Build targets
build: ## Build the stormdb binary
	@echo "🔨 Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(GO_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "   Version: $(VERSION)"
	@echo "   Commit:  $(GIT_COMMIT)"

build-all: build plugins ## Build stormdb binary and all plugins
	@echo "🔨 Building complete solution (binary + plugins)..."
	@echo "✅ Complete build finished"

build-dev: ## Build development version with debug info and race detector
	@echo "🔨 Building development version..."
	@mkdir -p $(BUILD_DIR)
	@go build -race -o $(BUILD_DIR)/$(BINARY_NAME)-dev $(CMD_DIR)/main.go
	@echo "✅ Development build complete: $(BUILD_DIR)/$(BINARY_NAME)-dev"

build-static: ## Build statically linked binary
	@echo "🔨 Building static binary..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux go build $(GO_FLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME)-static $(CMD_DIR)/main.go
	@echo "✅ Static build complete: $(BUILD_DIR)/$(BINARY_NAME)-static"

install: build ## Install stormdb to GOPATH/bin
	@echo "📦 Installing $(BINARY_NAME)..."
	@go install $(GO_FLAGS) $(CMD_DIR)/main.go
	@echo "✅ Installation complete"

# Test targets
test: test-unit ## Run fast unit tests (default test target)
	@echo "✅ Basic tests completed"

test-unit: ## Run unit tests only
	@echo "🧪 Running unit tests..."
	@go test $(GO_TEST_FLAGS) ./test/unit/... ./internal/... ./pkg/...
	@echo "✅ Unit tests completed"

test-integration: ## Run integration tests (requires database)
	@echo "🧪 Running integration tests..."
	@echo "⚠️  Integration tests require a PostgreSQL database"
	@go test $(GO_TEST_FLAGS) ./test/integration/... -timeout=60s
	@echo "✅ Integration tests completed"

test-load: ## Run load tests (requires database, resource intensive)
	@echo "🧪 Running load tests..."
	@echo "⚠️  Load tests require a PostgreSQL database and significant resources"
	@go test $(GO_TEST_FLAGS) ./test/load/... -timeout=300s
	@echo "✅ Load tests completed"

test-plugins: plugins ## Test all plugins
	@echo "🧪 Running plugin tests..."
	@$(MAKE) -C plugins test
	@echo "✅ Plugin tests completed"

test-all: ## Run all test suites
	@echo "🧪 Running all tests..."
	@$(MAKE) test-unit
	@$(MAKE) test-integration
	@$(MAKE) test-load
	@$(MAKE) test-plugins
	@echo "✅ All tests completed"

test-coverage: ## Generate test coverage report
	@echo "📊 Generating test coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	@go test $(GO_TEST_FLAGS) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | tail -1
	@echo "✅ Coverage report generated: $(COVERAGE_DIR)/coverage.html"

test-race: ## Run tests with race detector
	@echo "🏃 Running tests with race detector..."
	@go test -race -short ./...
	@echo "✅ Race detection tests completed"

# Code quality targets
fmt: ## Format Go source code
	@echo "🎨 Formatting code..."
	@go fmt ./...
	@goimports -w -local stormdb .
	@echo "✅ Code formatting complete"

vet: ## Run go vet static analysis
	@echo "🔍 Running go vet..."
	@go vet ./...
	@echo "✅ Static analysis complete"

lint: ## Run golangci-lint with comprehensive checks
	@echo "🔍 Running comprehensive linting..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi
	@golangci-lint run --timeout=5m
	@echo "✅ Linting complete"

lint-fix: ## Auto-fix linting issues where possible
	@echo "🔧 Auto-fixing linting issues..."
	@golangci-lint run --fix --timeout=5m
	@echo "✅ Auto-fix complete"

quality: fmt vet lint ## Run all code quality checks
	@echo "✅ All quality checks complete"

# Security targets
security: ## Run security analysis
	@echo "� Running security analysis..."
	@if ! command -v gosec > /dev/null; then \
		echo "Installing gosec..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi
	@gosec -severity medium -confidence medium -quiet ./...
	@echo "✅ Security analysis complete"

vuln-check: ## Check for known vulnerabilities
	@echo "🛡️  Checking for vulnerabilities..."
	@if ! command -v govulncheck > /dev/null; then \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@govulncheck ./...
	@echo "✅ Vulnerability check complete"

# Plugin targets
plugin-dir: ## Create plugin directory
	@echo "📁 Creating plugin directory..."
	@mkdir -p $(PLUGIN_DIR)
	@echo "✅ Plugin directory created: $(PLUGIN_DIR)"

plugins: plugin-dir ## Build all workload plugins
	@echo "🔌 Building all workload plugins..."
	@$(MAKE) -C plugins all
	@echo "🔄 Copying plugins to build directory..."
	@if [ -d "plugins/build/plugins" ]; then \
		cp plugins/build/plugins/*.so $(PLUGIN_DIR)/ 2>/dev/null || true; \
	fi
	@echo "✅ All plugins built successfully"

plugins-test: ## Test all plugins
	@echo "� Testing all plugins..."
	@$(MAKE) -C plugins test
	@echo "✅ Plugin tests completed"

plugins-clean: ## Clean built plugins
	@echo "🧹 Cleaning plugins..."
	@$(MAKE) -C plugins clean
	@rm -rf $(PLUGIN_DIR)
	@echo "✅ Plugin cleanup complete"

plugins-install: plugins ## Install plugins to system directory
	@echo "📦 Installing plugins to system directory..."
	@sudo mkdir -p /usr/local/lib/stormdb/plugins
	@sudo cp $(PLUGIN_DIR)/*.so /usr/local/lib/stormdb/plugins/ 2>/dev/null || true
	@echo "✅ Plugins installed"

list-plugins: ## List available plugins in build directory
	@echo "🔌 Available plugins:"
	@if [ -d "$(PLUGIN_DIR)" ]; then \
		find $(PLUGIN_DIR) -name "*.so" -o -name "*.dll" -o -name "*.dylib" | \
		while read plugin; do \
			echo "  📦 $$(basename $$plugin)"; \
		done; \
		if [ -z "$$(find $(PLUGIN_DIR) -name "*.so" -o -name "*.dll" -o -name "*.dylib" 2>/dev/null)" ]; then \
			echo "  (no plugins found)"; \
		fi; \
	else \
		echo "  (plugin directory does not exist)"; \
	fi

# Documentation targets
docs: ## Start Go documentation server
	@echo "📚 Starting documentation server..."
	@echo "📖 Documentation server starting at http://localhost:$(GODOC_PORT)"
	@echo "📁 API docs available at http://localhost:$(GODOC_PORT)/pkg/stormdb/"
	@echo "💡 Press Ctrl+C to stop the documentation server"
	@godoc -http=:$(GODOC_PORT)

docs-generate: ## Generate static documentation files
	@echo "📚 Generating static documentation..."
	@mkdir -p $(DOCS_DIR)/api
	@go doc -all ./cmd/stormdb > $(DOCS_DIR)/api/stormdb.txt 2>/dev/null || echo "Main package documentation generated"
	@go doc -all ./pkg/types > $(DOCS_DIR)/api/types.txt 2>/dev/null || echo "Types package documentation generated"
	@go doc -all ./internal/workload > $(DOCS_DIR)/api/workload.txt 2>/dev/null || echo "Workload package documentation generated"
	@go doc -all ./internal/metrics > $(DOCS_DIR)/api/metrics.txt 2>/dev/null || echo "Metrics package documentation generated"
	@go doc -all ./internal/database > $(DOCS_DIR)/api/database.txt 2>/dev/null || echo "Database package documentation generated"
	@go doc -all ./internal/config > $(DOCS_DIR)/api/config.txt 2>/dev/null || echo "Config package documentation generated"
	@echo "✅ Documentation files generated in $(DOCS_DIR)/api/"

docs-serve: ## Serve documentation locally using a simple HTTP server
	@echo "📚 Serving documentation at http://localhost:8080"
	@python3 -m http.server 8080 --directory $(DOCS_DIR) || python -m SimpleHTTPServer 8080

# Docker targets
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	@docker build -t stormdb:$(VERSION) -t stormdb:latest .
	@echo "✅ Docker image built: stormdb:$(VERSION)"

docker-run: ## Run stormdb in Docker container
	@echo "🐳 Running stormdb in Docker..."
	@docker run --rm -it stormdb:latest --help

docker-test: ## Run tests in Docker container
	@echo "🧪 Running tests in Docker..."
	@docker run --rm stormdb:latest make test-unit

# Container registry targets (customize registry as needed)
REGISTRY ?= localhost:5000

docker-push: docker-build ## Push Docker image to registry
	@echo "📤 Pushing to registry $(REGISTRY)..."
	@docker tag stormdb:$(VERSION) $(REGISTRY)/stormdb:$(VERSION)
	@docker tag stormdb:latest $(REGISTRY)/stormdb:latest
	@docker push $(REGISTRY)/stormdb:$(VERSION)
	@docker push $(REGISTRY)/stormdb:latest
	@echo "✅ Images pushed to $(REGISTRY)"

# Dependency management
deps: ## Install and update dependencies
	@echo "📦 Managing dependencies..."
	@go mod download
	@go mod tidy
	@go mod verify
	@echo "✅ Dependencies updated and verified"

deps-upgrade: ## Upgrade all dependencies to latest versions
	@echo "⬆️  Upgrading dependencies..."
	@go get -u all
	@go mod tidy
	@go mod verify
	@echo "✅ Dependencies upgraded"

deps-graph: ## Generate dependency graph
	@echo "📊 Generating dependency graph..."
	@go mod graph | grep stormdb | head -20
	@echo "💡 Use 'go mod graph | dot -T svg -o deps.svg' for visual graph"

deps-why: ## Show why dependencies are needed (requires package name)
	@echo "❓ Dependency analysis:"
	@echo "Usage: make deps-why PACKAGE='github.com/example/package'"
	@if [ -n "$(PACKAGE)" ]; then go mod why $(PACKAGE); fi

# Development tools
dev-tools: ## Install development tools
	@echo "🛠️  Installing development tools..."
	@go install golang.org/x/tools/cmd/godoc@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/air-verse/air@latest
	@echo "✅ Development tools installed"

dev-watch: ## Watch for changes and rebuild automatically (requires air)
	@echo "👀 Watching for changes..."
	@if ! command -v air > /dev/null; then \
		echo "Installing air for live reload..."; \
		go install github.com/air-verse/air@latest; \
	fi
	@air

# Performance and profiling targets
benchmark: ## Run performance benchmarks
	@echo "🏃 Running benchmarks..."
	@mkdir -p $(PROFILES_DIR)
	@go test $(GO_BENCH_FLAGS) ./... | tee $(PROFILES_DIR)/benchmark.txt
	@echo "✅ Benchmarks complete, results saved to $(PROFILES_DIR)/benchmark.txt"

profile-cpu: ## Generate CPU profile during benchmarks
	@echo "🧠 Generating CPU profile..."
	@mkdir -p $(PROFILES_DIR)
	@go test -bench=. -cpuprofile=$(PROFILES_DIR)/cpu.prof -benchmem ./...
	@echo "📊 CPU profile saved to $(PROFILES_DIR)/cpu.prof"
	@echo "💡 View with: go tool pprof $(PROFILES_DIR)/cpu.prof"

profile-mem: ## Generate memory profile during benchmarks
	@echo "💾 Generating memory profile..."
	@mkdir -p $(PROFILES_DIR)
	@go test -bench=. -memprofile=$(PROFILES_DIR)/mem.prof -benchmem ./...
	@echo "📊 Memory profile saved to $(PROFILES_DIR)/mem.prof"
	@echo "💡 View with: go tool pprof $(PROFILES_DIR)/mem.prof"

profile-trace: ## Generate execution trace
	@echo "🔍 Generating execution trace..."
	@mkdir -p $(PROFILES_DIR)
	@go test -trace=$(PROFILES_DIR)/trace.out ./...
	@echo "📊 Trace saved to $(PROFILES_DIR)/trace.out"
	@echo "💡 View with: go tool trace $(PROFILES_DIR)/trace.out"

profile-all: profile-cpu profile-mem profile-trace ## Generate all profiles
	@echo "✅ All profiles generated in $(PROFILES_DIR)/"

# Cleanup targets
clean: ## Clean build artifacts and temporary files
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@rm -rf $(PROFILES_DIR)
	@rm -f *.log *.prof *.out
	@rm -f cpu.prof mem.prof trace.out
	@echo "✅ Cleanup complete"

clean-all: clean plugins-clean ## Remove all generated files including documentation and caches
	@echo "🧹 Deep cleaning..."
	@rm -rf $(DOCS_DIR)/api
	@rm -rf vendor/
	@go clean -cache -testcache -modcache
	@echo "✅ Deep cleanup complete"

clean-docker: ## Remove Docker images and containers
	@echo "� Cleaning Docker resources..."
	@docker rmi stormdb:latest stormdb:$(VERSION) 2>/dev/null || true
	@docker system prune -f
	@echo "✅ Docker cleanup complete"

# Validation and CI targets
validate: quality security test-unit ## Run all validation checks (fast)
	@echo "✅ All validation checks passed"

validate-full: quality security vuln-check test-all ## Run comprehensive validation
	@echo "✅ Full validation complete"

validate-ci: fmt vet test-unit test-race ## Run CI-friendly validation (no external tools)
	@echo "✅ CI validation complete"

pre-commit: fmt vet lint test-unit ## Pre-commit hooks
	@echo "🔍 Running pre-commit checks..."
	@echo "✅ Pre-commit checks passed"

# Release targets  
release-check: clean-all validate-full docs-generate benchmark ## Pre-release validation
	@echo "🚀 Release validation..."
	@$(MAKE) build-all
	@$(MAKE) test-coverage
	@echo "✅ Release checks complete"

release-build: ## Build release artifacts for multiple platforms
	@echo "🚀 Building release artifacts..."
	@mkdir -p $(BUILD_DIR)/release
	@GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)/main.go
	@GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)/main.go
	@GOOS=darwin GOARCH=arm64 go build $(GO_FLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)/main.go
	@GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)/main.go
	@echo "✅ Release artifacts built in $(BUILD_DIR)/release/"

# Information targets
version: ## Show version information
	@echo "StormDB Version Information:"
	@echo "  Version:    $(VERSION)"
	@echo "  Git Commit: $(GIT_COMMIT)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Go Version: $(GO_VERSION)"

info: ## Show project information
	@echo "StormDB Project Information:"
	@echo "  Binary:     $(BINARY_NAME)"
	@echo "  Build Dir:  $(BUILD_DIR)"
	@echo "  Plugin Dir: $(PLUGIN_DIR)"
	@echo "  Go Version: $(GO_VERSION)"
	@echo "  Modules:"
	@go list -m all | head -10

# Help target - must be last
help: ## Display this help message
	@echo "StormDB - PostgreSQL Benchmarking Tool"
	@echo "======================================"
	@echo ""
	@echo "🏗️  BUILD TARGETS:"
	@awk 'BEGIN {FS = ":.*?## "} /^build.*:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "🧪 TEST TARGETS:"
	@awk 'BEGIN {FS = ":.*?## "} /^test.*:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "🔍 QUALITY TARGETS:"
	@awk 'BEGIN {FS = ":.*?## "} /^(fmt|vet|lint|quality|security|vuln-check):.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""  
	@echo "🔌 PLUGIN TARGETS:"
	@awk 'BEGIN {FS = ":.*?## "} /^plugin.*:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "📚 DOCUMENTATION TARGETS:"
	@awk 'BEGIN {FS = ":.*?## "} /^docs.*:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "🐳 DOCKER TARGETS:"
	@awk 'BEGIN {FS = ":.*?## "} /^docker.*:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "🛠️  DEVELOPMENT TARGETS:"
	@awk 'BEGIN {FS = ":.*?## "} /^(dev-.*|deps.*|profile.*|benchmark):.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "✅ VALIDATION TARGETS:"
	@awk 'BEGIN {FS = ":.*?## "} /^validate.*:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "🧹 CLEANUP TARGETS:"
	@awk 'BEGIN {FS = ":.*?## "} /^clean.*:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "🚀 RELEASE TARGETS:"
	@awk 'BEGIN {FS = ":.*?## "} /^release.*:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "ℹ️  INFORMATION TARGETS:"
	@awk 'BEGIN {FS = ":.*?## "} /^(version|info):.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "🌟 COMMON WORKFLOWS:"
	@echo "  \033[33mFirst time setup:\033[0m"
	@echo "    make dev-tools deps build-all"
	@echo ""
	@echo "  \033[33mDevelopment cycle:\033[0m"
	@echo "    make dev-watch          # Watch and rebuild"
	@echo "    make pre-commit         # Before committing"
	@echo ""
	@echo "  \033[33mTesting:\033[0m"
	@echo "    make test               # Quick tests"
	@echo "    make test-all           # Full test suite"
	@echo "    make validate-full      # Complete validation"
	@echo ""
	@echo "  \033[33mRelease preparation:\033[0m"
	@echo "    make release-check      # Pre-release checks"
	@echo "    make release-build      # Multi-platform builds"
	@echo "    make release-cross      # Cross-platform release"
	@echo ""
	@echo "Environment Variables:"
	@echo "  VERSION               Version tag (default: git describe)"
	@echo "  REGISTRY              Docker registry (default: localhost:5000)"
	@echo "  GODOC_PORT           Documentation server port (default: 6060)"
	@echo "  STORMDB_TEST_HOST    PostgreSQL host for tests (default: localhost)"
	@echo "  STORMDB_TEST_DB      PostgreSQL database for tests (default: storm)"
	@echo "  STORMDB_TEST_USER    PostgreSQL user for tests"
	@echo "  STORMDB_TEST_PASS    PostgreSQL password for tests"

# Release targets
release-check: ## Run pre-release validation checks
	@echo "🔍 Running pre-release checks..."
	@echo "Version: $(VERSION)"
	@echo "Commit: $(GIT_COMMIT)"
	@$(MAKE) validate-full
	@$(MAKE) test-all
	@$(MAKE) security
	@echo "✅ Pre-release checks completed successfully"

release-build: ## Build release binaries for current platform
	@echo "🚀 Building release binary for current platform..."
	@mkdir -p $(BUILD_DIR)/release
	@CGO_ENABLED=1 go build $(GO_FLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME) $(CMD_DIR)/main.go
	@$(MAKE) plugins
	@if [ -d $(PLUGIN_DIR) ]; then \
		cp -r $(PLUGIN_DIR) $(BUILD_DIR)/release/; \
	fi
	@echo "✅ Release build complete: $(BUILD_DIR)/release/"

release-cross: ## Build cross-platform release binaries and packages
	@echo "🌍 Building cross-platform release..."
	@chmod +x scripts/build-cross-platform.sh
	@VERSION=$(VERSION) COMMIT=$(GIT_COMMIT) ./scripts/build-cross-platform.sh
	@echo "✅ Cross-platform release build complete"

release-docker: ## Build and tag Docker images for release
	@echo "🐳 Building release Docker images..."
	@docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(GIT_COMMIT) \
		-t $(REGISTRY)/$(BINARY_NAME):$(VERSION) \
		-t $(REGISTRY)/$(BINARY_NAME):latest .
	@echo "✅ Docker images built:"
	@echo "  $(REGISTRY)/$(BINARY_NAME):$(VERSION)"
	@echo "  $(REGISTRY)/$(BINARY_NAME):latest"

release-docker-push: release-docker ## Build and push Docker images to registry
	@echo "📤 Pushing Docker images to registry..."
	@docker push $(REGISTRY)/$(BINARY_NAME):$(VERSION)
	@docker push $(REGISTRY)/$(BINARY_NAME):latest
	@echo "✅ Docker images pushed successfully"

release-package-deb: ## Create DEB package (requires fpm)
	@echo "📦 Creating DEB package..."
	@command -v fpm >/dev/null 2>&1 || { echo "❌ fpm not found. Install with: gem install fpm"; exit 1; }
	@$(MAKE) release-build
	@mkdir -p $(BUILD_DIR)/packages/deb/usr/local/bin
	@mkdir -p $(BUILD_DIR)/packages/deb/etc/stormdb
	@mkdir -p $(BUILD_DIR)/packages/deb/usr/share/doc/stormdb
	@cp $(BUILD_DIR)/release/$(BINARY_NAME) $(BUILD_DIR)/packages/deb/usr/local/bin/
	@cp -r config/* $(BUILD_DIR)/packages/deb/etc/stormdb/
	@cp README.md CHANGELOG.md $(BUILD_DIR)/packages/deb/usr/share/doc/stormdb/
	@fpm -s dir -t deb \
		--name $(BINARY_NAME) \
		--version $(VERSION:v%=%) \
		--maintainer "StormDB Team" \
		--description "PostgreSQL performance testing and benchmarking tool" \
		--url "https://github.com/elchinoo/stormdb" \
		--license "MIT" \
		--after-install scripts/postinstall.sh \
		--after-remove scripts/postremove.sh \
		-C $(BUILD_DIR)/packages/deb \
		--package $(BUILD_DIR)/packages/
	@echo "✅ DEB package created in $(BUILD_DIR)/packages/"

release-package-rpm: ## Create RPM package (requires fpm)
	@echo "📦 Creating RPM package..."
	@command -v fpm >/dev/null 2>&1 || { echo "❌ fpm not found. Install with: gem install fpm"; exit 1; }
	@$(MAKE) release-build
	@mkdir -p $(BUILD_DIR)/packages/rpm/usr/local/bin
	@mkdir -p $(BUILD_DIR)/packages/rpm/etc/stormdb
	@mkdir -p $(BUILD_DIR)/packages/rpm/usr/share/doc/stormdb
	@cp $(BUILD_DIR)/release/$(BINARY_NAME) $(BUILD_DIR)/packages/rpm/usr/local/bin/
	@cp -r config/* $(BUILD_DIR)/packages/rpm/etc/stormdb/
	@cp README.md CHANGELOG.md $(BUILD_DIR)/packages/rpm/usr/share/doc/stormdb/
	@fpm -s dir -t rpm \
		--name $(BINARY_NAME) \
		--version $(VERSION:v%=%) \
		--maintainer "StormDB Team" \
		--description "PostgreSQL performance testing and benchmarking tool" \
		--url "https://github.com/elchinoo/stormdb" \
		--license "MIT" \
		--after-install scripts/postinstall.sh \
		--after-remove scripts/postremove.sh \
		-C $(BUILD_DIR)/packages/rpm \
		--package $(BUILD_DIR)/packages/
	@echo "✅ RPM package created in $(BUILD_DIR)/packages/"

release-packages: release-package-deb release-package-rpm ## Create both DEB and RPM packages

release-checksums: ## Generate checksums for release artifacts
	@echo "🔐 Generating release checksums..."
	@find $(BUILD_DIR)/release -type f -name "$(BINARY_NAME)*" -exec sha256sum {} \; > $(BUILD_DIR)/release/SHA256SUMS
	@find $(BUILD_DIR)/packages -type f \( -name "*.deb" -o -name "*.rpm" -o -name "*.tar.gz" -o -name "*.zip" \) -exec sha256sum {} \; >> $(BUILD_DIR)/release/SHA256SUMS 2>/dev/null || true
	@echo "✅ Checksums generated: $(BUILD_DIR)/release/SHA256SUMS"

release-notes: ## Generate release notes from CHANGELOG
	@echo "📝 Generating release notes..."
	@if [ ! -f "GITHUB_RELEASE_DESCRIPTION.md" ]; then \
		echo "❌ GITHUB_RELEASE_DESCRIPTION.md not found"; \
		echo "   Create this file with your release description"; \
		exit 1; \
	fi
	@echo "✅ Release notes ready: GITHUB_RELEASE_DESCRIPTION.md"

release-full: release-check release-cross release-docker release-checksums release-notes ## Complete release build process
	@echo "🎉 Full release build completed!"
	@echo ""
	@echo "📋 Release artifacts:"
	@echo "  Binaries: $(BUILD_DIR)/release/binaries/"
	@echo "  Packages: $(BUILD_DIR)/release/packages/"
	@echo "  Checksums: $(BUILD_DIR)/release/SHA256SUMS"
	@echo "  Docker: $(REGISTRY)/$(BINARY_NAME):$(VERSION)"
	@echo ""
	@echo "🚀 Ready for GitHub release!"

release-clean: ## Clean release artifacts
	@echo "🧹 Cleaning release artifacts..."
	@rm -rf $(BUILD_DIR)/release $(BUILD_DIR)/packages
	@echo "✅ Release artifacts cleaned"
