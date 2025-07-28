#!/bin/bash

# StormDB Example Usage Script
# This script demonstrates various ways to use StormDB for PostgreSQL performance testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper function to print colored output
print_step() {
    echo -e "${BLUE}==== $1 ====${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Check if stormdb binary exists
if [ ! -f "./stormdb" ]; then
    print_error "stormdb binary not found. Please build it first:"
    echo "  go build -o stormdb cmd/stormdb/main.go"
    exit 1
fi

# Example 1: Quick Performance Test
run_example_1() {
    print_step "Example 1: Quick Performance Test (TPC-C)"
    echo "This runs a quick TPC-C benchmark to get baseline performance metrics."
    echo "Duration: 30 seconds, 4 workers, basic monitoring"
    echo ""
    
    ./stormdb --config config/config_tpcc.yaml \
              --duration 30s \
              --workers 4 \
              --connections 8 \
              --collect-pg-stats \
              --summary-interval 10s
    
    print_success "Quick performance test completed"
    echo ""
}

# Example 2: Comprehensive IMDB Analysis
run_example_2() {
    print_step "Example 2: Comprehensive IMDB Analysis with Full Monitoring"
    echo "This runs an IMDB mixed workload with full PostgreSQL monitoring enabled."
    echo "Includes pg_stat_statements for query analysis."
    echo ""
    
    # First setup the schema
    print_warning "Setting up IMDB schema..."
    ./stormdb --config config/config_imdb_mixed.yaml --setup
    
    # Run the test with full monitoring
    ./stormdb --config config/config_imdb_mixed.yaml \
              --collect-pg-stats \
              --pg-stat-statements \
              --duration 60s \
              --summary-interval 10s
    
    print_success "IMDB analysis completed"
    echo ""
}

# Example 3: Connection Overhead Analysis
run_example_3() {
    print_step "Example 3: Connection Overhead Analysis"
    echo "This compares persistent vs transient connection performance."
    echo "Shows the overhead of creating new connections for each operation."
    echo ""
    
    ./stormdb --config config/config_connection_overhead.yaml \
              --duration 45s \
              --workers 6 \
              --collect-pg-stats
    
    print_success "Connection overhead analysis completed"
    echo ""
}

# Example 4: Vector Similarity Performance
run_example_4() {
    print_step "Example 4: Vector Similarity Performance Testing"
    echo "Tests high-dimensional vector operations with different similarity metrics."
    echo ""
    
    # Test L2 distance
    print_warning "Testing L2 distance similarity..."
    ./stormdb --workload vector_1024 \
              --scale 5000 \
              --duration 30s \
              --workers 4 \
              --collect-pg-stats
    
    echo ""
    
    # Test cosine similarity
    print_warning "Testing cosine similarity..."
    ./stormdb --workload vector_1024_cosine \
              --scale 5000 \
              --duration 30s \
              --workers 4 \
              --collect-pg-stats
    
    print_success "Vector similarity tests completed"
    echo ""
}

# Example 5: Scale Testing
run_example_5() {
    print_step "Example 5: Scale Testing - Performance vs Load"
    echo "Tests how performance changes with different scales and worker counts."
    echo ""
    
    scales=(1 2 5)
    workers=(2 4 8)
    
    for scale in "${scales[@]}"; do
        for worker in "${workers[@]}"; do
            print_warning "Testing scale=$scale, workers=$worker"
            ./stormdb --config config/config_tpcc.yaml \
                      --scale $scale \
                      --workers $worker \
                      --duration 20s \
                      --no-summary \
                      --collect-pg-stats
            echo ""
        done
    done
    
    print_success "Scale testing completed"
    echo ""
}

# Function to display menu
show_menu() {
    echo -e "${YELLOW}StormDB Example Usage Menu${NC}"
    echo "=================================="
    echo "1. Quick Performance Test (TPC-C)"
    echo "2. Comprehensive IMDB Analysis"
    echo "3. Connection Overhead Analysis"
    echo "4. Vector Similarity Performance"
    echo "5. Scale Testing (Multiple Configurations)"
    echo "6. Run All Examples"
    echo "0. Exit"
    echo ""
}

# Main execution
main() {
    echo -e "${GREEN}StormDB PostgreSQL Performance Testing Examples${NC}"
    echo "=============================================="
    echo ""
    
    # Check database connectivity first
    print_step "Checking Database Connectivity"
    if ./stormdb --config config/config_tpcc.yaml --duration 1s --workers 1 --no-summary > /dev/null 2>&1; then
        print_success "Database connection successful"
    else
        print_error "Database connection failed. Please check your configuration files."
        echo "Make sure PostgreSQL is running and credentials are correct."
        exit 1
    fi
    echo ""
    
    # If arguments provided, run specific examples
    if [ $# -gt 0 ]; then
        case $1 in
            1) run_example_1 ;;
            2) run_example_2 ;;
            3) run_example_3 ;;
            4) run_example_4 ;;
            5) run_example_5 ;;
            all) 
                run_example_1
                sleep 2
                run_example_2
                sleep 2
                run_example_3
                sleep 2
                run_example_4
                sleep 2
                run_example_5
                ;;
            *) 
                print_error "Invalid example number: $1"
                echo "Valid options: 1, 2, 3, 4, 5, all"
                exit 1
                ;;
        esac
        exit 0
    fi
    
    # Interactive menu
    while true; do
        show_menu
        echo -n "Select an example (0-6): "
        read -r choice
        echo ""
        
        case $choice in
            1) run_example_1 ;;
            2) run_example_2 ;;
            3) run_example_3 ;;
            4) run_example_4 ;;
            5) run_example_5 ;;
            6)
                print_warning "Running all examples. This will take several minutes..."
                run_example_1
                sleep 3
                run_example_2
                sleep 3
                run_example_3
                sleep 3
                run_example_4
                sleep 3
                run_example_5
                print_success "All examples completed!"
                ;;
            0)
                print_success "Goodbye!"
                exit 0
                ;;
            *)
                print_error "Invalid choice. Please select 0-6."
                ;;
        esac
        
        echo ""
        echo -n "Press Enter to continue..."
        read
        echo ""
    done
}

# Usage information
usage() {
    echo "Usage: $0 [example_number|all]"
    echo ""
    echo "Examples:"
    echo "  $0          # Interactive menu"
    echo "  $0 1        # Run example 1 only"
    echo "  $0 all      # Run all examples"
    echo ""
    echo "Available examples:"
    echo "  1 - Quick Performance Test (TPC-C)"
    echo "  2 - Comprehensive IMDB Analysis"
    echo "  3 - Connection Overhead Analysis"
    echo "  4 - Vector Similarity Performance"
    echo "  5 - Scale Testing"
}

# Check for help flag
if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    usage
    exit 0
fi

# Run main function
main "$@"
