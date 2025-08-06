# StormDB v0.4-alpha Makefile

.PHONY: build clean test install deps run dev plugins build-all test-all

# Build directory
BUILD_DIR = build
PLUGINS_DIR = $(BUILD_DIR)/plugins

# Application binary
APP_NAME = stormdb
APP_BINARY = $(BUILD_DIR)/$(APP_NAME)

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod

# Main application entry point
MAIN_PATH = ./cmd/stormdb

# Plugin directories
PLUGIN_DIRS = $(wildcard plugins/*)

# Default target
all: build-all

# Build everything (core + plugins)
build-all: build plugins

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Build the application
build: deps
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(APP_BINARY) $(MAIN_PATH)
	@echo "Build complete: $(APP_BINARY)"

# Build all plugins
plugins: build
	@echo "Building plugins..."
	@mkdir -p $(PLUGINS_DIR)
	@for plugin_dir in $(PLUGIN_DIRS); do \
		echo "Building plugin: $$plugin_dir"; \
		cd $$plugin_dir && $(MAKE) plugin && cd ../..; \
	done
	@echo "All plugins built successfully"

# Build specific plugin
plugin-%:
	@echo "Building plugin: plugins/$*"
	@mkdir -p $(PLUGINS_DIR)
	@cd plugins/$* && $(MAKE) plugin

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@for plugin_dir in $(PLUGIN_DIRS); do \
		echo "Cleaning plugin: $$plugin_dir"; \
		cd $$plugin_dir && $(MAKE) clean && cd ../..; \
	done
	@echo "Clean complete"

# Run tests for core and plugins
test-all: test test-plugins

# Run core tests
test:
	@echo "Running core tests..."
	$(GOTEST) -v ./core/... ./cmd/...

# Run plugin tests
test-plugins:
	@echo "Running plugin tests..."
	@for plugin_dir in $(PLUGIN_DIRS); do \
		echo "Testing plugin: $$plugin_dir"; \
		cd $$plugin_dir && $(MAKE) test && cd ../..; \
	done

# Test specific plugin
test-plugin-%:
	@echo "Testing plugin: plugins/$*"
	@cd plugins/$* && $(MAKE) test

# Install the application and plugins
install: build-all
	@echo "Installing $(APP_NAME)..."
	cp $(APP_BINARY) /usr/local/bin/$(APP_NAME)
	@echo "Installing plugins..."
	@sudo mkdir -p /usr/local/lib/stormdb/plugins
	@sudo cp $(PLUGINS_DIR)/*.so /usr/local/lib/stormdb/plugins/ 2>/dev/null || true
	@echo "Installation complete"

# Run the application in development mode
dev: build-all
	@echo "Starting $(APP_NAME) in development mode..."
	@mkdir -p ./config
	STORMDB_CONFIG=./config/core.yaml $(APP_BINARY)

# Run the application
run: build
	@echo "Starting $(APP_NAME)..."
	$(APP_BINARY)

# Run with plugins
run-with-plugins: build-all
	@echo "Starting $(APP_NAME) with plugins..."
	STORMDB_PLUGIN_DIR=$(PLUGINS_DIR) $(APP_BINARY)

# Format code (core and plugins)
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	@for plugin_dir in $(PLUGIN_DIRS); do \
		echo "Formatting plugin: $$plugin_dir"; \
		cd $$plugin_dir && $(MAKE) fmt && cd ../..; \
	done

# Lint code (requires golangci-lint)
lint:
	@echo "Linting core code..."
	golangci-lint run
	@for plugin_dir in $(PLUGIN_DIRS); do \
		echo "Linting plugin: $$plugin_dir"; \
		cd $$plugin_dir && $(MAKE) lint && cd ../..; \
	done

# Check for security issues (requires gosec)
security:
	@echo "Checking for security issues..."
	gosec ./...
	@for plugin_dir in $(PLUGIN_DIRS); do \
		if [ -f $$plugin_dir/Makefile ]; then \
			echo "Security check for plugin: $$plugin_dir"; \
			cd $$plugin_dir && gosec . && cd ../..; \
		fi \
	done

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t stormdb:v0.4-alpha .

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 -p 5432:5432 stormdb:v0.4-alpha

# Development setup
setup-dev:
	@echo "Setting up development environment..."
	@mkdir -p config logs plugins
	@if [ ! -f config/core.yaml ]; then \
		cp config/core.yaml.example config/core.yaml 2>/dev/null || \
		echo "Warning: No example config found"; \
	fi

# List available plugins
list-plugins:
	@echo "Available plugins:"
	@for plugin_dir in $(PLUGIN_DIRS); do \
		if [ -f $$plugin_dir/plugin.go ]; then \
			echo "  - $$(basename $$plugin_dir)"; \
		fi \
	done

# Plugin status
plugins-status:
	@echo "Plugin build status:"
	@for plugin_dir in $(PLUGIN_DIRS); do \
		plugin_name=$$(basename $$plugin_dir); \
		if [ -f $(PLUGINS_DIR)/$$plugin_name.so ]; then \
			echo "  ✓ $$plugin_name - built"; \
		else \
			echo "  ✗ $$plugin_name - not built"; \
		fi \
	done

# Show help
help:
	@echo "Available targets:"
	@echo "  build-all      - Build application and all plugins"
	@echo "  build          - Build the core application"
	@echo "  plugins        - Build all plugins"
	@echo "  plugin-<name>  - Build specific plugin"
	@echo "  clean          - Clean build artifacts"
	@echo "  test-all       - Run all tests (core + plugins)"
	@echo "  test           - Run core tests"
	@echo "  test-plugins   - Run plugin tests"
	@echo "  test-plugin-<name> - Test specific plugin"
	@echo "  install        - Install application and plugins"
	@echo "  dev            - Run in development mode"
	@echo "  run            - Run the application"
	@echo "  run-with-plugins - Run with plugins loaded"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  security       - Check for security issues"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-run     - Run Docker container"
	@echo "  setup-dev      - Setup development environment"
	@echo "  list-plugins   - List available plugins"
	@echo "  plugins-status - Show plugin build status"
	@echo "  deps           - Install dependencies"
	@echo "  help           - Show this help message"
