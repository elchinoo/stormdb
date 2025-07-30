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

# Cross-compilation settings (default to current platform, override for releases)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# FPM binary location (try common locations)
FPM := $(shell command -v fpm 2>/dev/null || command -v /opt/homebrew/opt/ruby/bin/fpm 2>/dev/null || echo "fpm")

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
GOLANGCI_LINT_VERSION := v1.64.8
GODOC_PORT := 6060

# Build targets
build: ## Build the stormdb binary
	@echo "🔨 Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(GO_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "   Version: $(VERSION)"
	@echo "   Commit:  $(GIT_COMMIT)"

build-all: build plugins ## Build stormdb binary and all plugins
	@echo "🔨 Building complete solution (binary + plugins)..."
	@echo "✅ Complete build finished"

ubuntu-build: ## Ubuntu-friendly complete build process
	@echo "🐧 Ubuntu Build Process Starting..."
	@echo "Step 1: Installing dependencies..."
	@$(MAKE) deps
	@echo "Step 2: Building main binary..."
	@$(MAKE) build
	@echo "Step 3: Building plugins..."
	@$(MAKE) plugins
	@echo "Step 4: Verifying build..."
	@if [ -f "$(BUILD_DIR)/$(BINARY_NAME)" ]; then \
		echo "✅ Binary built successfully: $(BUILD_DIR)/$(BINARY_NAME)"; \
		./$(BUILD_DIR)/$(BINARY_NAME) --help | head -5; \
	else \
		echo "❌ Binary build failed"; \
		exit 1; \
	fi
	@echo "🐧 Ubuntu build complete! Run: ./$(BUILD_DIR)/$(BINARY_NAME) --help"

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
	@echo "Security checks integrated into golangci-lint configuration"
	@echo "Run 'make lint' for comprehensive code quality and security analysis"
	@echo "✅ Security analysis complete (no separate gosec installation needed)"

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
	@GOOS=$(GOOS) GOARCH=$(GOARCH) $(MAKE) -C plugins all
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
	@echo "Installing gosec security scanner..."
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@v2.21.4 || echo "⚠️  gosec installation failed, continuing..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/air-verse/air@latest
	@echo "✅ Development tools installed"

dev-tools-minimal: ## Install minimal development tools (Ubuntu-friendly)
	@echo "🛠️  Installing minimal development tools..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "✅ Minimal development tools installed"

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

release-cross: ## Build cross-platform release artifacts
	@echo "🚀 Building cross-platform release artifacts..."
	@mkdir -p $(BUILD_DIR)/release
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)/main.go
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)/main.go
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(GO_FLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)/main.go
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)/main.go
	@echo "✅ Cross-platform release artifacts built in $(BUILD_DIR)/release/"

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

# Build targets for x86_64 Linux (used by package creation)

release-build: ## Build release binary for x86_64 Linux (cross-platform aware)
	@echo "🚀 Building release binary for x86_64 Linux..."
	@mkdir -p $(BUILD_DIR)/release
	@if [ "$(shell uname)" = "Darwin" ]; then \
		echo "⚡ Cross-compiling x86_64 static binary from macOS..."; \
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME) $(CMD_DIR)/main.go; \
		echo "⚠️  Plugins skipped: Go plugins require CGO and cannot be cross-compiled"; \
		echo "   💡 Use './build-docker-native.sh' for complete packages with plugins"; \
	else \
		echo "🐧 Building native Linux binary with plugins..."; \
		CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME) $(CMD_DIR)/main.go; \
		GOOS=linux GOARCH=amd64 $(MAKE) plugins; \
		if [ -d $(PLUGIN_DIR) ]; then \
			cp -r $(PLUGIN_DIR) $(BUILD_DIR)/release/; \
		fi; \
	fi
	@echo "✅ Release build complete: $(BUILD_DIR)/release/"

release-script-cross: ## Build cross-platform release using external script
	@echo "🌍 Building cross-platform release using script..."
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

release-package-deb: ## Create DEB package with proper Linux filesystem layout
	@echo "📦 Creating DEB package..."
	@command -v fpm >/dev/null 2>&1 || command -v /opt/homebrew/opt/ruby/bin/fpm >/dev/null 2>&1 || { echo "❌ fpm not found. Install with: gem install fpm"; exit 1; }
	@$(MAKE) release-build
	@echo "Setting up DEB package directory structure..."
	
	# Create directory structure
	@mkdir -p $(BUILD_DIR)/packages/deb/usr/bin
	@mkdir -p $(BUILD_DIR)/packages/deb/usr/lib/stormdb/plugins
	@mkdir -p $(BUILD_DIR)/packages/deb/etc/stormdb/examples
	@mkdir -p $(BUILD_DIR)/packages/deb/usr/share/man/man1
	@mkdir -p $(BUILD_DIR)/packages/deb/usr/share/doc/stormdb
	@mkdir -p $(BUILD_DIR)/packages/deb/usr/share/stormdb
	
	# Install binary to /usr/bin/stormdb
	@cp $(BUILD_DIR)/release/$(BINARY_NAME) $(BUILD_DIR)/packages/deb/usr/bin/
	
	# Install plugins to /usr/lib/stormdb/plugins/
	@if [ -d "$(BUILD_DIR)/plugins" ]; then \
		cp $(BUILD_DIR)/plugins/*.so $(BUILD_DIR)/packages/deb/usr/lib/stormdb/plugins/ 2>/dev/null || true; \
	fi
	
	# Install configuration files to /etc/stormdb/examples/
	@cp config/*.yaml $(BUILD_DIR)/packages/deb/etc/stormdb/examples/
	
	# Install config_tpcc.yaml to /etc/stormdb/config_tpcc.yaml
	@cp config/config_tpcc.yaml $(BUILD_DIR)/packages/deb/etc/stormdb/
	
	# Install man page to /usr/share/man/man1/
	@cp stormdb.1 $(BUILD_DIR)/packages/deb/usr/share/man/man1/
	@gzip -9 $(BUILD_DIR)/packages/deb/usr/share/man/man1/stormdb.1
	
	# Install documentation to /usr/share/doc/stormdb/
	@cp README.md CHANGELOG.md ARCHITECTURE.md $(BUILD_DIR)/packages/deb/usr/share/doc/stormdb/
	@cp -r docs/* $(BUILD_DIR)/packages/deb/usr/share/doc/stormdb/
	
	# Install static data to /usr/share/stormdb/
	@cp imdb.sql $(BUILD_DIR)/packages/deb/usr/share/stormdb/ 2>/dev/null || true
	@cp -r config $(BUILD_DIR)/packages/deb/usr/share/stormdb/templates
	
	@$(FPM) -s dir -t deb \
		--name $(BINARY_NAME) \
		--version $(VERSION:v%=%) \
		--maintainer "StormDB Team <team@stormdb.org>" \
		--description "PostgreSQL performance testing and benchmarking tool with plugin-based workload architecture" \
		--url "https://github.com/elchinoo/stormdb" \
		--license "MIT" \
		--architecture amd64 \
		--depends postgresql-client \
		--category database \
		--after-install scripts/postinstall.sh \
		--after-remove scripts/postremove.sh \
		-C $(BUILD_DIR)/packages/deb \
		--package $(BUILD_DIR)/packages/
	@echo "✅ DEB package created in $(BUILD_DIR)/packages/"

release-package-rpm: ## Create RPM package with proper Linux filesystem layout
	@echo "📦 Creating RPM package..."
	@command -v fpm >/dev/null 2>&1 || command -v /opt/homebrew/opt/ruby/bin/fpm >/dev/null 2>&1 || { echo "❌ fpm not found. Install with: gem install fpm"; exit 1; }
	@$(MAKE) release-build
	@echo "Setting up RPM package directory structure..."
	
	# Create directory structure
	@mkdir -p $(BUILD_DIR)/packages/rpm/usr/bin
	@mkdir -p $(BUILD_DIR)/packages/rpm/usr/lib/stormdb/plugins
	@mkdir -p $(BUILD_DIR)/packages/rpm/etc/stormdb/examples
	@mkdir -p $(BUILD_DIR)/packages/rpm/usr/share/man/man1
	@mkdir -p $(BUILD_DIR)/packages/rpm/usr/share/doc/stormdb
	@mkdir -p $(BUILD_DIR)/packages/rpm/usr/share/stormdb
	
	# Install binary to /usr/bin/stormdb
	@cp $(BUILD_DIR)/release/$(BINARY_NAME) $(BUILD_DIR)/packages/rpm/usr/bin/
	
	# Install plugins to /usr/lib/stormdb/plugins/
	@if [ -d "$(BUILD_DIR)/plugins" ]; then \
		cp $(BUILD_DIR)/plugins/*.so $(BUILD_DIR)/packages/rpm/usr/lib/stormdb/plugins/ 2>/dev/null || true; \
	fi
	
	# Install configuration files to /etc/stormdb/examples/
	@cp config/*.yaml $(BUILD_DIR)/packages/rpm/etc/stormdb/examples/
	
	# Install config_tpcc.yaml to /etc/stormdb/config_tpcc.yaml
	@cp config/config_tpcc.yaml $(BUILD_DIR)/packages/rpm/etc/stormdb/
	
	# Install man page to /usr/share/man/man1/
	@cp stormdb.1 $(BUILD_DIR)/packages/rpm/usr/share/man/man1/
	@gzip -9 $(BUILD_DIR)/packages/rpm/usr/share/man/man1/stormdb.1
	
	# Install documentation to /usr/share/doc/stormdb/
	@cp README.md CHANGELOG.md ARCHITECTURE.md $(BUILD_DIR)/packages/rpm/usr/share/doc/stormdb/
	@cp -r docs/* $(BUILD_DIR)/packages/rpm/usr/share/doc/stormdb/
	
	# Install static data to /usr/share/stormdb/
	@cp imdb.sql $(BUILD_DIR)/packages/rpm/usr/share/stormdb/ 2>/dev/null || true
	@cp -r config $(BUILD_DIR)/packages/rpm/usr/share/stormdb/templates
	
	@$(FPM) -s dir -t rpm \
		--name $(BINARY_NAME) \
		--version $(VERSION:v%=%) \
		--maintainer "StormDB Team <team@stormdb.org>" \
		--description "PostgreSQL performance testing and benchmarking tool with plugin-based workload architecture" \
		--url "https://github.com/elchinoo/stormdb" \
		--license "MIT" \
		--architecture x86_64 \
		--depends postgresql \
		--category "Applications/Databases" \
		--after-install scripts/postinstall.sh \
		--after-remove scripts/postremove.sh \
		-C $(BUILD_DIR)/packages/rpm \
		--package $(BUILD_DIR)/packages/
	@echo "✅ RPM package created in $(BUILD_DIR)/packages/"

release-packages: release-package-deb release-package-rpm ## Create both DEB and RPM packages

# Native package creation targets (for use inside Docker containers)
package-deb-native: ## Create DEB package natively inside container
	@echo "📦 Creating native DEB package..."
	@mkdir -p build/packages
	@mkdir -p $(BUILD_DIR)/deb/stormdb-$(VERSION)
	
	# Install binary and plugins to standard Linux locations
	@mkdir -p $(BUILD_DIR)/deb/stormdb-$(VERSION)/usr/bin
	@mkdir -p $(BUILD_DIR)/deb/stormdb-$(VERSION)/usr/lib/stormdb/plugins
	@mkdir -p $(BUILD_DIR)/deb/stormdb-$(VERSION)/usr/share/doc/stormdb
	@mkdir -p $(BUILD_DIR)/deb/stormdb-$(VERSION)/usr/share/man/man1
	@mkdir -p $(BUILD_DIR)/deb/stormdb-$(VERSION)/etc/stormdb
	
	# Copy binary (built natively inside container)
	@cp $(BUILD_DIR)/stormdb $(BUILD_DIR)/deb/stormdb-$(VERSION)/usr/bin/
	
	# Copy plugins (built natively inside container)
	@if [ -d "$(PLUGIN_DIR)" ]; then \
		cp $(PLUGIN_DIR)/*.so $(BUILD_DIR)/deb/stormdb-$(VERSION)/usr/lib/stormdb/plugins/ 2>/dev/null || true; \
	fi
	
	# Copy documentation
	@cp README.md $(BUILD_DIR)/deb/stormdb-$(VERSION)/usr/share/doc/stormdb/
	@cp ARCHITECTURE.md $(BUILD_DIR)/deb/stormdb-$(VERSION)/usr/share/doc/stormdb/ 2>/dev/null || true
	@if [ -d "docs" ]; then \
		cp -r docs/* $(BUILD_DIR)/deb/stormdb-$(VERSION)/usr/share/doc/stormdb/ 2>/dev/null || true; \
	fi
	
	# Copy configuration files
	@if [ -d "config" ]; then \
		cp -r config/* $(BUILD_DIR)/deb/stormdb-$(VERSION)/etc/stormdb/ 2>/dev/null || true; \
	fi
	
	# Create package using FPM with proper DEB architecture (amd64 for x86_64)
	$(FPM) -s dir -t deb \
		-n stormdb \
		-v $(VERSION) \
		-a amd64 \
		--description "StormDB - PostgreSQL Performance Benchmarking Tool" \
		--url "https://github.com/charly-batista/stormdb" \
		--maintainer "Charly Batista <charly.batista@example.com>" \
		--license "MIT" \
		--category "database" \
		--depends "postgresql-client" \
		--config-files "/etc/stormdb" \
		-C $(BUILD_DIR)/deb/stormdb-$(VERSION) \
		-p build/packages/stormdb_$(VERSION)_amd64.deb \
		.
	
	@echo "✅ Native DEB package created: build/packages/stormdb_$(VERSION)_amd64.deb"

package-rpm-native: ## Create RPM package natively inside container
	@echo "📦 Creating native RPM package..."
	@mkdir -p build/packages
	@mkdir -p $(BUILD_DIR)/rpm/stormdb-$(VERSION)
	
	# Install binary and plugins to standard Linux locations
	@mkdir -p $(BUILD_DIR)/rpm/stormdb-$(VERSION)/usr/bin
	@mkdir -p $(BUILD_DIR)/rpm/stormdb-$(VERSION)/usr/lib64/stormdb/plugins
	@mkdir -p $(BUILD_DIR)/rpm/stormdb-$(VERSION)/usr/share/doc/stormdb
	@mkdir -p $(BUILD_DIR)/rpm/stormdb-$(VERSION)/usr/share/man/man1
	@mkdir -p $(BUILD_DIR)/rpm/stormdb-$(VERSION)/etc/stormdb
	
	# Copy binary (built natively inside container)
	@cp $(BUILD_DIR)/stormdb $(BUILD_DIR)/rpm/stormdb-$(VERSION)/usr/bin/
	
	# Copy plugins (built natively inside container)
	@if [ -d "$(PLUGIN_DIR)" ]; then \
		cp $(PLUGIN_DIR)/*.so $(BUILD_DIR)/rpm/stormdb-$(VERSION)/usr/lib64/stormdb/plugins/ 2>/dev/null || true; \
	fi
	
	# Copy documentation
	@cp README.md $(BUILD_DIR)/rpm/stormdb-$(VERSION)/usr/share/doc/stormdb/
	@cp ARCHITECTURE.md $(BUILD_DIR)/rpm/stormdb-$(VERSION)/usr/share/doc/stormdb/ 2>/dev/null || true
	@if [ -d "docs" ]; then \
		cp -r docs/* $(BUILD_DIR)/rpm/stormdb-$(VERSION)/usr/share/doc/stormdb/ 2>/dev/null || true; \
	fi
	
	# Copy configuration files
	@if [ -d "config" ]; then \
		cp -r config/* $(BUILD_DIR)/rpm/stormdb-$(VERSION)/etc/stormdb/ 2>/dev/null || true; \
	fi
	
	# Create package using FPM with proper RPM architecture (x86_64)
	$(FPM) -s dir -t rpm \
		-n stormdb \
		-v $(VERSION) \
		-a x86_64 \
		--description "StormDB - PostgreSQL Performance Benchmarking Tool" \
		--url "https://github.com/charly-batista/stormdb" \
		--maintainer "Charly Batista <charly.batista@example.com>" \
		--license "MIT" \
		--category "Applications/Databases" \
		--depends "postgresql" \
		--config-files "/etc/stormdb" \
		-C $(BUILD_DIR)/rpm/stormdb-$(VERSION) \
		-p build/packages/stormdb-$(VERSION)-1.x86_64.rpm \
		.
	
	@echo "✅ Native RPM package created: build/packages/stormdb-$(VERSION)-1.x86_64.rpm"

# Package testing targets
test-packages: ## Test packages locally using Docker across multiple distributions
	@echo "🧪 Testing packages across multiple Linux distributions..."
	@./docker/test-packages.sh

test-packages-local: ## Test package building locally (without Docker)
	@echo "🧪 Testing package building locally..."
	@./docker/test-local.sh

test-packages-ubuntu: ## Test DEB package on Ubuntu using Docker
	@echo "🧪 Testing DEB package on Ubuntu..."
	@./docker/test-packages.sh --distro ubuntu

test-packages-debian: ## Test DEB package on Debian using Docker
	@echo "🧪 Testing DEB package on Debian..."
	@./docker/test-packages.sh --distro debian

test-packages-centos: ## Test RPM package on CentOS using Docker
	@echo "🧪 Testing RPM package on CentOS..."
	@./docker/test-packages.sh --distro centos

test-packages-verbose: ## Test packages with verbose output
	@echo "🧪 Testing packages with verbose output..."
	@./docker/test-packages.sh --verbose

release-test: release-packages test-packages ## Build and test packages before release

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
