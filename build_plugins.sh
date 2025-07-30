#!/bin/bash
# Build script for all StormDB plugins

set -e

echo "🔌 Building StormDB Plugins"
echo "=========================="

# Create build directory
echo "📁 Creating build directory..."
mkdir -p build/plugins

# Plugin directories
PLUGINS=(
    "imdb_plugin"
    "vector_plugin" 
    "ecommerce_basic_plugin"
    "ecommerce_plugin"
)

# Build each plugin
for plugin in "${PLUGINS[@]}"; do
    echo ""
    echo "🔨 Building $plugin..."
    
    if [ ! -d "plugins/$plugin" ]; then
        echo "❌ Plugin directory plugins/$plugin not found!"
        continue
    fi
    
    cd "plugins/$plugin"
    
    # Check if go.mod exists
    if [ ! -f "go.mod" ]; then
        echo "❌ go.mod not found in $plugin"
        cd ../..
        continue
    fi
    
    # Build the plugin
    if go build -buildmode=plugin -o "../../build/plugins/${plugin}.so" main.go; then
        echo "✅ Successfully built $plugin"
    else
        echo "❌ Failed to build $plugin"
        cd ../..
        continue
    fi
    
    cd ../..
done

echo ""
echo "📋 Plugin Build Summary"
echo "======================"

# List built plugins
if [ -d "build/plugins" ]; then
    plugin_count=$(ls -1 build/plugins/*.so 2>/dev/null | wc -l)
    echo "Built plugins: $plugin_count"
    
    for plugin_file in build/plugins/*.so; do
        if [ -f "$plugin_file" ]; then
            plugin_name=$(basename "$plugin_file" .so)
            file_size=$(ls -lh "$plugin_file" | awk '{print $5}')
            echo "  ✅ $plugin_name ($file_size)"
        fi
    done
else
    echo "❌ No plugins built"
fi

echo ""
echo "🎉 Plugin build completed!"
echo ""
echo "To use the plugins, ensure your configuration includes:"
echo "plugins:"
echo "  paths:"
echo "    - \"./build/plugins\""
echo "  auto_load: true"
