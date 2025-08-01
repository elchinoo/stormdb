# Plugin Build Makefile
# This makefile builds all StormDB workload plugins

.PHONY: all clean imdb vector ecommerce_basic ecommerce tpcc simple connection bulk_insert test build-dir deps-check install

# Default plugin output directory (can be overridden)
PLUGIN_DIR ?= ../build/plugins

# Go build flags for plugins
GO_PLUGIN_FLAGS := -buildmode=plugin -ldflags="-s -w"

# macOS-specific linker flags to suppress dylib warnings
ifeq ($(shell uname),Darwin)
    GO_PLUGIN_FLAGS := -buildmode=plugin -ldflags="-s -w -extldflags=-Wl,-w"
    BUILD_CMD = go build $(GO_PLUGIN_FLAGS)
else
    BUILD_CMD = go build $(GO_PLUGIN_FLAGS)
endif

# Build all plugins
all: build-dir imdb vector ecommerce_basic ecommerce tpcc simple connection bulk_insert

# Create build directory
build-dir:
	@mkdir -p $(PLUGIN_DIR)

# Build IMDB plugin
imdb: build-dir
	@echo "🔌 Building IMDB plugin..."
	@cd imdb_plugin && $(BUILD_CMD) -o $(PLUGIN_DIR)/imdb_plugin.so *.go
	@echo "✅ IMDB plugin built: $(PLUGIN_DIR)/imdb_plugin.so"

# Build Vector plugin  
vector: build-dir
	@echo "🔌 Building Vector plugin..."
	@cd vector_plugin && $(BUILD_CMD) -o $(PLUGIN_DIR)/vector_plugin.so *.go
	@echo "✅ Vector plugin built: $(PLUGIN_DIR)/vector_plugin.so"

# Build E-commerce Basic plugin
ecommerce_basic: build-dir
	@echo "🔌 Building E-commerce Basic plugin..."
	@cd ecommerce_basic_plugin && $(BUILD_CMD) -o $(PLUGIN_DIR)/ecommerce_basic_plugin.so *.go
	@echo "✅ E-commerce Basic plugin built: $(PLUGIN_DIR)/ecommerce_basic_plugin.so"

# Build E-commerce plugin
ecommerce: build-dir
	@echo "🔌 Building E-commerce plugin..."
	@cd ecommerce_plugin && $(BUILD_CMD) -o $(PLUGIN_DIR)/ecommerce_plugin.so *.go
	@echo "✅ E-commerce plugin built: $(PLUGIN_DIR)/ecommerce_plugin.so"

# Build TPC-C plugin
tpcc: build-dir
	@echo "🔌 Building TPC-C plugin..."
	@cd tpcc_plugin && $(BUILD_CMD) -o ../$(PLUGIN_DIR)/tpcc_plugin.so *.go
	@echo "✅ TPC-C plugin built: $(PLUGIN_DIR)/tpcc_plugin.so"

# Build Simple plugin
simple: build-dir
	@echo "🔌 Building Simple plugin..."
	@cd simple_plugin && $(BUILD_CMD) -o ../$(PLUGIN_DIR)/simple_plugin.so *.go
	@echo "✅ Simple plugin built: $(PLUGIN_DIR)/simple_plugin.so"

# Build Connection plugin
connection: build-dir
	@echo "🔌 Building Connection plugin..."
	@cd connection_plugin && $(BUILD_CMD) -o ../$(PLUGIN_DIR)/connection_plugin.so *.go
	@echo "✅ Connection plugin built: $(PLUGIN_DIR)/connection_plugin.so"

# Build Bulk Insert plugin
bulk_insert: build-dir
	@echo "🔌 Building Bulk Insert plugin..."
	@cd bulk_insert_plugin && $(BUILD_CMD) -o ../$(PLUGIN_DIR)/bulk_insert_plugin.so *.go
	@echo "✅ Bulk Insert plugin built: $(PLUGIN_DIR)/bulk_insert_plugin.so"

# Test all plugins
test:
	@echo "🧪 Testing plugins..."
	@cd imdb_plugin && go test -v ./...
	@cd vector_plugin && go test -v ./...
	@cd ecommerce_basic_plugin && go test -v ./...
	@cd ecommerce_plugin && go test -v ./...
	@cd tpcc_plugin && go test -v ./...
	@cd simple_plugin && go test -v ./...
	@cd connection_plugin && go test -v ./...
	@cd bulk_insert_plugin && go test -v ./...
	@echo "✅ Plugin tests completed"

# Check plugin dependencies
deps-check:
	@echo "📦 Checking plugin dependencies..."
	@cd imdb_plugin && go mod verify && go mod tidy
	@cd vector_plugin && go mod verify && go mod tidy
	@cd ecommerce_basic_plugin && go mod verify && go mod tidy
	@cd ecommerce_plugin && go mod verify && go mod tidy
	@cd tpcc_plugin && go mod verify && go mod tidy
	@cd simple_plugin && go mod verify && go mod tidy
	@cd connection_plugin && go mod verify && go mod tidy
	@cd bulk_insert_plugin && go mod verify && go mod tidy
	@echo "✅ Plugin dependencies verified"

# Clean built plugins
clean:
	@echo "🧹 Cleaning plugins..."
	@rm -f $(PLUGIN_DIR)/*.so
	@echo "✅ Plugin cleanup complete"

# Install plugins to system directory (requires sudo)
install: all
	@echo "📦 Installing plugins to /usr/local/lib/stormdb/plugins..."
	@sudo mkdir -p /usr/local/lib/stormdb/plugins
	@sudo cp $(PLUGIN_DIR)/*.so /usr/local/lib/stormdb/plugins/
	@echo "✅ Plugins installed to system directory"

# Development helpers
fmt:
	@echo "🎨 Formatting plugin code..."
	@cd imdb_plugin && go fmt ./...
	@cd vector_plugin && go fmt ./...
	@cd ecommerce_basic_plugin && go fmt ./...
	@cd ecommerce_plugin && go fmt ./...
	@cd tpcc_plugin && go fmt ./...
	@cd simple_plugin && go fmt ./...
	@cd connection_plugin && go fmt ./...
	@cd bulk_insert_plugin && go fmt ./...
	@echo "✅ Plugin code formatted"

vet:
	@echo "🔍 Running go vet on plugins..."
	@cd imdb_plugin && go vet ./...
	@cd vector_plugin && go vet ./...
	@cd ecommerce_basic_plugin && go vet ./...
	@cd ecommerce_plugin && go vet ./...
	@cd tpcc_plugin && go vet ./...
	@cd simple_plugin && go vet ./...
	@cd connection_plugin && go vet ./...
	@cd bulk_insert_plugin && go vet ./...
	@echo "✅ Plugin static analysis complete"
