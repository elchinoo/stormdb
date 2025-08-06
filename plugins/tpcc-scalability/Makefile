# TPC-C Scalability Plugin Makefile

.PHONY: build clean test install plugin

# Plugin configuration
PLUGIN_NAME = tpcc-scalability
PLUGIN_SO = $(PLUGIN_NAME).so
PLUGIN_DIR = ../../build/plugins

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
GOMOD = $(GOCMD) mod
GOCLEAN = $(GOCMD) clean

# Default target
all: plugin

# Install dependencies
deps:
	@echo "Installing plugin dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Build the plugin as a shared library
plugin: deps
	@echo "Building $(PLUGIN_NAME) plugin..."
	@mkdir -p $(PLUGIN_DIR)
	$(GOBUILD) -buildmode=plugin -o $(PLUGIN_DIR)/$(PLUGIN_SO) .
	@echo "Plugin built: $(PLUGIN_DIR)/$(PLUGIN_SO)"

# Run tests
test:
	@echo "Running plugin tests..."
	$(GOTEST) -v .

# Run tests with coverage
test-coverage:
	@echo "Running plugin tests with coverage..."
	$(GOTEST) -v -cover .

# Run benchmarks
bench:
	@echo "Running plugin benchmarks..."
	$(GOTEST) -bench=. -benchmem .

# Clean build artifacts
clean:
	@echo "Cleaning plugin artifacts..."
	@rm -f $(PLUGIN_DIR)/$(PLUGIN_SO)
	$(GOCLEAN)

# Install plugin to system location
install: plugin
	@echo "Installing $(PLUGIN_NAME) plugin..."
	@sudo mkdir -p /usr/local/lib/stormdb/plugins
	@sudo cp $(PLUGIN_DIR)/$(PLUGIN_SO) /usr/local/lib/stormdb/plugins/
	@echo "Plugin installed to /usr/local/lib/stormdb/plugins/"

# Format code
fmt:
	@echo "Formatting plugin code..."
	$(GOCMD) fmt .

# Lint code (requires golangci-lint)
lint:
	@echo "Linting plugin code..."
	golangci-lint run

# Validate plugin configuration
validate-config:
	@echo "Validating plugin configuration..."
	@if [ -f config-example.yaml ]; then \
		echo "Configuration example found"; \
	else \
		echo "Warning: config-example.yaml not found"; \
	fi

# Show plugin info
info:
	@echo "Plugin: $(PLUGIN_NAME)"
	@echo "Output: $(PLUGIN_DIR)/$(PLUGIN_SO)"
	@echo "Configuration: config-example.yaml"

# Help
help:
	@echo "Available targets:"
	@echo "  plugin         - Build the plugin shared library"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  bench          - Run benchmarks"
	@echo "  clean          - Clean build artifacts"
	@echo "  install        - Install plugin to system location"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  validate-config- Validate configuration"
	@echo "  info           - Show plugin information"
	@echo "  deps           - Install dependencies"
	@echo "  help           - Show this help message"
