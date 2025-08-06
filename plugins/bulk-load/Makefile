# Makefile for Bulk Load Plugin

PLUGIN_NAME = bulk-load
PLUGIN_SO = ../../build/plugins/$(PLUGIN_NAME).so
GO_FILES = $(wildcard *.go)

# Build settings
CGO_ENABLED = 1
GOOS = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

.PHONY: all plugin test clean install deps lint format vet benchmark coverage help

# Default target
all: plugin

# Build plugin as shared library
plugin: $(PLUGIN_SO)

$(PLUGIN_SO): $(GO_FILES) go.mod go.sum
	@echo "Building bulk load plugin..."
	@mkdir -p ../../build/plugins
	CGO_ENABLED=$(CGO_ENABLED) go build -buildmode=plugin -o $(PLUGIN_SO) .
	@echo "Plugin built successfully: $(PLUGIN_SO)"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running unit tests..."
	go test -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Format code
format:
	@echo "Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, skipping..."; \
	fi

# Install plugin to system directory
install: plugin
	@echo "Installing plugin..."
	@if [ -z "$(PLUGIN_DIR)" ]; then \
		echo "PLUGIN_DIR not set, installing to ./build/"; \
		mkdir -p ./build/plugins; \
		cp $(PLUGIN_SO) ./build/plugins/; \
	else \
		mkdir -p $(PLUGIN_DIR); \
		cp $(PLUGIN_SO) $(PLUGIN_DIR)/; \
	fi
	@echo "Plugin installed successfully"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(PLUGIN_SO)
	rm -f coverage.out coverage.html
	rm -rf ./build/
	go clean -cache -modcache -testcache

# Development helpers
dev-setup: deps
	@echo "Setting up development environment..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi

# Quick quality check
check: format vet lint test
	@echo "All quality checks passed!"

# Build and test pipeline
ci: deps check plugin
	@echo "CI pipeline completed successfully!"

# Help target
help:
	@echo "Available targets:"
	@echo "  plugin     - Build the plugin as a shared library"
	@echo "  test       - Run unit tests"
	@echo "  coverage   - Run tests with coverage report"
	@echo "  benchmark  - Run performance benchmarks"
	@echo "  format     - Format Go code"
	@echo "  vet        - Run Go vet"
	@echo "  lint       - Run golangci-lint (if available)"
	@echo "  install    - Install plugin to PLUGIN_DIR or ./build/"
	@echo "  clean      - Remove build artifacts"
	@echo "  deps       - Download and tidy dependencies"
	@echo "  dev-setup  - Set up development environment"
	@echo "  check      - Run format, vet, lint, and test"
	@echo "  ci         - Full CI pipeline (deps, check, plugin)"
	@echo "  help       - Show this help message"

# Targets that don't create files
.NOTPARALLEL:
