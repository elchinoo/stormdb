#!/bin/bash
# progress_demo.sh - Demonstrates the new progress tracking features in StormDB
# Shows progress bars during data seeding operations for different workloads

set -e

BINARY="./build/stormdb"
CONFIGS=(
    "config/config_progress_demo_tpcc.yaml"
    "config/config_progress_demo_ecommerce.yaml" 
    "config/config_progress_demo_imdb.yaml"
)

WORKLOAD_NAMES=(
    "TPC-C (Warehouses → Districts → Customers)"
    "E-commerce (Vendors → Users → Products → Orders → Reviews)"
    "IMDB (Actors → Movies → Comments → Logs → Voting)"
)

echo "=========================================="
echo "StormDB Progress Tracking Demo"
echo "=========================================="
echo "This demo showcases the new progress tracking features"
echo "during data seeding operations (--setup and --rebuild)."
echo ""
echo "Features demonstrated:"
echo "• Real-time progress bars with completion percentages"
echo "• Insertion rates (items per second)"
echo "• ETA (estimated time to completion)"
echo "• Multi-stage progress for complex workloads"
echo ""

# Check if binary exists
if [[ ! -f "$BINARY" ]]; then
    echo "❌ StormDB binary not found at $BINARY"
    echo "Please run 'make build-all' first to build the binary and plugins."
    exit 1
fi

# Function to run a demo
run_demo() {
    local config=$1
    local workload_name=$2
    
    echo "=========================================="
    echo "Demo: $workload_name"
    echo "=========================================="
    echo "Configuration: $config"
    echo ""
    echo "🔧 Setting up schema and loading data with progress tracking..."
    echo ""
    
    # Run with --setup to show progress bars during data seeding
    if $BINARY -c "$config" --setup; then
        echo ""
        echo "✅ Setup completed successfully!"
        echo ""
        echo "📊 Running quick workload test (10 seconds)..."
        echo ""
        
        # Run a short test to show the workload is functional
        $BINARY -c "$config" --duration 10s
        
        echo ""
        echo "✅ Workload test completed!"
    else
        echo ""
        echo "❌ Setup failed for $config"
        return 1
    fi
    
    echo ""
    echo "Press Enter to continue to next demo, or Ctrl+C to exit..."
    read
}

echo "Starting demos..."
echo ""

# Run each demo
for i in "${!CONFIGS[@]}"; do
    config="${CONFIGS[$i]}"
    workload_name="${WORKLOAD_NAMES[$i]}"
    
    if [[ -f "$config" ]]; then
        run_demo "$config" "$workload_name"
    else
        echo "⚠️  Skipping $config (file not found)"
    fi
done

echo "=========================================="
echo "Demo Complete!"
echo "=========================================="
echo ""
echo "Summary of what you saw:"
echo "• Progress bars during data seeding for all major workloads"
echo "• Real-time completion percentages and insertion rates"
echo "• ETA calculations based on current progress"
echo "• Multi-stage progress tracking (e.g., warehouses → districts → customers)"
echo ""
echo "Try these commands to explore further:"
echo "• $BINARY -c config/config_progress_demo_tpcc.yaml --rebuild"
echo "• $BINARY -c config/config_progress_demo_ecommerce.yaml --setup"
echo "• $BINARY -c config/config_progress_demo_imdb.yaml --setup"
echo ""
echo "For larger datasets with longer progress bars, increase the 'scale' parameter"
echo "in the configuration files."
echo ""
echo "Happy testing! 🚀"
