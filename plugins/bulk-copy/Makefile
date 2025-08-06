# Makefile for StormDB bulk-copy plugin

# Plugin configuration
PLUGIN_NAME := bulk-copy
PLUGIN_VERSION := 1.0.0
PLUGIN_DIR := $(shell pwd)
BUILD_DIR := ../../build/plugins
PLUGIN_OUTPUT := $(BUILD_DIR)/$(PLUGIN_NAME).so

# Go configuration
GO := go
GOFLAGS := -buildmode=plugin
GOBUILD := $(GO) build $(GOFLAGS)
GOCLEAN := $(GO) clean
GOTEST := $(GO) test
GOGET := $(GO) get
GOMOD := $(GO) mod

# Source files
SOURCES := $(wildcard *.go)

# Default target
.PHONY: all
all: build

# Build the plugin
.PHONY: build
build: $(PLUGIN_OUTPUT)

$(PLUGIN_OUTPUT): $(SOURCES)
	@echo "Building $(PLUGIN_NAME) plugin..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(PLUGIN_OUTPUT) $(SOURCES)
	@echo "Plugin built successfully: $(PLUGIN_OUTPUT)"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning $(PLUGIN_NAME) plugin build artifacts..."
	$(GOCLEAN)
	@rm -f $(PLUGIN_OUTPUT)
	@echo "Clean completed"

# Test the plugin
.PHONY: test
test:
	@echo "Running tests for $(PLUGIN_NAME) plugin..."
	$(GOTEST) -v ./...

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Verify dependencies
.PHONY: verify
verify:
	@echo "Verifying dependencies..."
	$(GOMOD) verify

# Development build with verbose output
.PHONY: dev-build
dev-build:
	@echo "Building $(PLUGIN_NAME) plugin (development mode)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -v -x -o $(PLUGIN_OUTPUT) $(SOURCES)
	@echo "Development build completed: $(PLUGIN_OUTPUT)"

# Install to system location (if needed)
.PHONY: install
install: build
	@echo "Installing $(PLUGIN_NAME) plugin..."
	@# Installation logic would go here if needed
	@echo "Installation completed"

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Lint code (requires golangci-lint)
.PHONY: lint
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, skipping lint"; \
	fi

# Run security checks (requires gosec)
.PHONY: security
security:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found, skipping security checks"; \
	fi

# Check for vulnerabilities (requires govulncheck)
.PHONY: vulncheck
vulncheck:
	@echo "Checking for vulnerabilities..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not found, skipping vulnerability checks"; \
	fi

# Full checks (format, lint, test, security)
.PHONY: check
check: fmt lint test security vulncheck
	@echo "All checks completed"

# Show plugin info
.PHONY: info
info:
	@echo "Plugin Information:"
	@echo "  Name: $(PLUGIN_NAME)"
	@echo "  Version: $(PLUGIN_VERSION)"
	@echo "  Source Directory: $(PLUGIN_DIR)"
	@echo "  Build Directory: $(BUILD_DIR)"
	@echo "  Output File: $(PLUGIN_OUTPUT)"
	@echo "  Go Version: $(shell $(GO) version)"

# Debug build with race detection
.PHONY: debug
debug:
	@echo "Building $(PLUGIN_NAME) plugin with debug flags..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -race -gcflags="-N -l" -o $(PLUGIN_OUTPUT) $(SOURCES)
	@echo "Debug build completed: $(PLUGIN_OUTPUT)"

# Quick rebuild (for development)
.PHONY: rebuild
rebuild: clean build

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all        - Build the plugin (default)"
	@echo "  build      - Build the plugin"
	@echo "  clean      - Clean build artifacts"
	@echo "  test       - Run tests"
	@echo "  deps       - Download dependencies"
	@echo "  verify     - Verify dependencies"
	@echo "  dev-build  - Build with verbose output"
	@echo "  install    - Install plugin"
	@echo "  fmt        - Format code"
	@echo "  lint       - Lint code"
	@echo "  security   - Run security checks"
	@echo "  vulncheck  - Check for vulnerabilities"
	@echo "  check      - Run all checks"
	@echo "  info       - Show plugin information"
	@echo "  debug      - Build with debug flags"
	@echo "  rebuild    - Clean and build"
	@echo "  help       - Show this help"
