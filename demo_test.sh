#!/bin/bash

# StormDB v2 Testing Demo Script
# This script demonstrates how to test StormDB v2 with both plugins

set -e

echo "🚀 StormDB v2 Testing Demo"
echo "=========================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
STORMDB_HOST="localhost"
STORMDB_PORT="8080"
STORMDB_URL="http://${STORMDB_HOST}:${STORMDB_PORT}"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if service is running
check_service() {
    local service_name=$1
    local check_command=$2
    
    print_status "Checking if $service_name is running..."
    if eval $check_command > /dev/null 2>&1; then
        print_success "$service_name is running"
        return 0
    else
        print_error "$service_name is not running"
        return 1
    fi
}

# Function to wait for service
wait_for_service() {
    local service_name=$1
    local check_command=$2
    local max_attempts=30
    local attempt=0
    
    print_status "Waiting for $service_name to be ready..."
    
    while [ $attempt -lt $max_attempts ]; do
        if eval $check_command > /dev/null 2>&1; then
            print_success "$service_name is ready!"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done
    
    print_error "$service_name failed to start within $((max_attempts * 2)) seconds"
    return 1
}

# Function to start PostgreSQL with Docker
start_postgres() {
    print_status "Starting PostgreSQL with Docker..."
    
    # Check if container already exists
    if docker ps -a --format 'table {{.Names}}' | grep -q "postgres-stormdb-test"; then
        print_warning "PostgreSQL container already exists. Removing it..."
        docker rm -f postgres-stormdb-test > /dev/null 2>&1 || true
    fi
    
    # Start PostgreSQL
    docker run --name postgres-stormdb-test \
        -e POSTGRES_PASSWORD=postgres \
        -e POSTGRES_DB=stormdb \
        -e POSTGRES_USER=postgres \
        -p 5432:5432 \
        -d postgres:15 > /dev/null
    
    # Wait for PostgreSQL to be ready
    wait_for_service "PostgreSQL" "docker exec postgres-stormdb-test pg_isready -U postgres"
    
    # Create test database and user
    print_status "Setting up test database..."
    docker exec postgres-stormdb-test psql -U postgres -c "
        CREATE DATABASE IF NOT EXISTS stormdb_test;
        CREATE USER IF NOT EXISTS stormdb WITH PASSWORD 'stormdb_password';
        GRANT ALL PRIVILEGES ON DATABASE stormdb_test TO stormdb;
        GRANT ALL PRIVILEGES ON DATABASE stormdb TO stormdb;
    " > /dev/null 2>&1 || true
    
    print_success "PostgreSQL is ready for testing"
}

# Function to build StormDB
build_stormdb() {
    print_status "Building StormDB v2 and plugins..."
    
    if make build-all > /dev/null 2>&1; then
        print_success "Build completed successfully"
    else
        print_error "Build failed"
        exit 1
    fi
    
    # Verify plugins are built
    if [ -f "build/plugins/bulk-load.so" ] && [ -f "build/plugins/tpcc-scalability.so" ]; then
        print_success "Both plugins built successfully"
        ls -la build/plugins/
    else
        print_error "Plugin build failed"
        exit 1
    fi
}

# Function to start StormDB
start_stormdb() {
    print_status "Starting StormDB v2 server..."
    
    # Kill any existing StormDB process
    pkill -f "stormdb" > /dev/null 2>&1 || true
    
    # Start StormDB in background
    STORMDB_PLUGIN_DIR=./build/plugins ./build/stormdb > stormdb.log 2>&1 &
    STORMDB_PID=$!
    
    print_status "StormDB PID: $STORMDB_PID"
    
    # Wait for StormDB to be ready
    wait_for_service "StormDB" "curl -s $STORMDB_URL/health"
    
    print_success "StormDB v2 is running on $STORMDB_URL"
}

# Function to test basic functionality
test_basic_functionality() {
    print_status "Testing basic functionality..."
    
    # Test health endpoint
    print_status "Testing health endpoint..."
    HEALTH=$(curl -s $STORMDB_URL/health)
    if echo $HEALTH | grep -q "healthy"; then
        print_success "Health check passed"
    else
        print_error "Health check failed: $HEALTH"
        return 1
    fi
    
    # Test plugins endpoint
    print_status "Testing plugins endpoint..."
    PLUGINS=$(curl -s $STORMDB_URL/plugins)
    if echo $PLUGINS | grep -q "bulk-load" && echo $PLUGINS | grep -q "tpcc-scalability"; then
        print_success "Both plugins are loaded"
        echo "Available plugins:"
        echo $PLUGINS | jq -r '.[] | "  - " + .name + " v" + .version'
    else
        print_error "Plugins not loaded correctly: $PLUGINS"
        return 1
    fi
}

# Function to run a quick bulk load test
test_bulk_load() {
    print_status "Running quick bulk load test..."
    
    # Create test
    TEST_RESPONSE=$(curl -s -X POST "$STORMDB_URL/test-runs" \
        -H "Content-Type: application/json" \
        -d '{
            "plugin_name": "bulk-load",
            "name": "Demo Bulk Load Test",
            "description": "Quick demonstration test with small batches",
            "config": {
                "host": "localhost",
                "port": 5432,
                "database": "stormdb",
                "username": "postgres",
                "password": "postgres",
                "ssl_mode": "disable",
                "batch_sizes": [1, 100],
                "connections": 5,
                "duration": "30s",
                "warmup_time": "5s",
                "table_name": "demo_bulk_test",
                "data_columns": 3,
                "verbose": true
            }
        }')
    
    if echo $TEST_RESPONSE | grep -q "error"; then
        print_error "Failed to create bulk load test: $TEST_RESPONSE"
        return 1
    fi
    
    TEST_ID=$(echo $TEST_RESPONSE | jq -r '.id')
    print_success "Bulk load test created with ID: $TEST_ID"
    
    # Monitor test progress
    print_status "Monitoring test progress..."
    local attempts=0
    local max_attempts=20
    
    while [ $attempts -lt $max_attempts ]; do
        STATUS_RESPONSE=$(curl -s "$STORMDB_URL/test-runs/$TEST_ID")
        STATUS=$(echo $STATUS_RESPONSE | jq -r '.status')
        
        print_status "Test status: $STATUS"
        
        if [ "$STATUS" = "succeeded" ]; then
            print_success "Bulk load test completed successfully!"
            
            # Get results
            RESULTS=$(curl -s "$STORMDB_URL/test-runs/$TEST_ID/results")
            echo "Results summary:"
            echo $RESULTS | jq '{
                total_transactions: .total_transactions,
                total_rows_inserted: .total_rows_inserted,
                batch_results: [.batch_results[] | {
                    batch_size: .batch_size,
                    tps: .transactions_per_sec,
                    rows_per_sec: .rows_per_sec
                }]
            }'
            return 0
            
        elif [ "$STATUS" = "failed" ]; then
            print_error "Bulk load test failed"
            echo $STATUS_RESPONSE | jq '.'
            return 1
            
        elif [ "$STATUS" = "running" ]; then
            echo -n "."
            
        fi
        
        attempts=$((attempts + 1))
        sleep 3
    done
    
    print_warning "Test is still running after timeout"
    return 0
}

# Function to run a quick TPC-C test
test_tpcc() {
    print_status "Running quick TPC-C test..."
    
    # Create test
    TEST_RESPONSE=$(curl -s -X POST "$STORMDB_URL/test-runs" \
        -H "Content-Type: application/json" \
        -d '{
            "plugin_name": "tpcc-scalability",
            "name": "Demo TPC-C Test",
            "description": "Quick demonstration test with small scale",
            "config": {
                "host": "localhost",
                "port": 5432,
                "database": "stormdb",
                "username": "postgres",
                "password": "postgres",
                "ssl_mode": "disable",
                "scale": 2,
                "connections": [5, 10],
                "duration": "30s",
                "warmup_time": "5s"
            }
        }')
    
    if echo $TEST_RESPONSE | grep -q "error"; then
        print_error "Failed to create TPC-C test: $TEST_RESPONSE"
        return 1
    fi
    
    TEST_ID=$(echo $TEST_RESPONSE | jq -r '.id')
    print_success "TPC-C test created with ID: $TEST_ID"
    
    # Monitor test progress
    print_status "Monitoring test progress..."
    local attempts=0
    local max_attempts=25
    
    while [ $attempts -lt $max_attempts ]; do
        STATUS_RESPONSE=$(curl -s "$STORMDB_URL/test-runs/$TEST_ID")
        STATUS=$(echo $STATUS_RESPONSE | jq -r '.status')
        
        print_status "Test status: $STATUS"
        
        if [ "$STATUS" = "succeeded" ]; then
            print_success "TPC-C test completed successfully!"
            
            # Get results
            RESULTS=$(curl -s "$STORMDB_URL/test-runs/$TEST_ID/results")
            echo "Results summary:"
            echo $RESULTS | jq '{
                scale: .scale,
                total_transactions: .total_transactions,
                connection_results: [.connection_results[] | {
                    connections: .connections,
                    tps: .tps,
                    avg_latency_ms: .avg_latency_ms
                }]
            }'
            return 0
            
        elif [ "$STATUS" = "failed" ]; then
            print_error "TPC-C test failed"
            echo $STATUS_RESPONSE | jq '.'
            return 1
            
        elif [ "$STATUS" = "running" ]; then
            echo -n "."
            
        fi
        
        attempts=$((attempts + 1))
        sleep 3
    done
    
    print_warning "Test is still running after timeout"
    return 0
}

# Function to cleanup
cleanup() {
    print_status "Cleaning up..."
    
    # Stop StormDB
    if [ -n "$STORMDB_PID" ]; then
        print_status "Stopping StormDB (PID: $STORMDB_PID)..."
        kill $STORMDB_PID > /dev/null 2>&1 || true
    fi
    
    # Stop PostgreSQL container
    if docker ps --format 'table {{.Names}}' | grep -q "postgres-stormdb-test"; then
        print_status "Stopping PostgreSQL container..."
        docker stop postgres-stormdb-test > /dev/null 2>&1 || true
        docker rm postgres-stormdb-test > /dev/null 2>&1 || true
    fi
    
    print_success "Cleanup completed"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --skip-postgres    Skip starting PostgreSQL (assume it's running)"
    echo "  --skip-build       Skip building StormDB"
    echo "  --skip-tests       Skip running demo tests"
    echo "  --cleanup-only     Only perform cleanup"
    echo "  --help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Full demo (start everything, run tests)"
    echo "  $0 --skip-postgres    # Demo with existing PostgreSQL"
    echo "  $0 --cleanup-only     # Just cleanup previous runs"
}

# Parse command line arguments
SKIP_POSTGRES=false
SKIP_BUILD=false
SKIP_TESTS=false
CLEANUP_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-postgres)
            SKIP_POSTGRES=true
            shift
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        --cleanup-only)
            CLEANUP_ONLY=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Trap to ensure cleanup on exit
trap cleanup EXIT

# Main execution
main() {
    if [ "$CLEANUP_ONLY" = true ]; then
        cleanup
        exit 0
    fi
    
    echo ""
    print_status "Starting StormDB v2 demo..."
    
    # Check dependencies
    print_status "Checking dependencies..."
    
    if ! command -v curl > /dev/null; then
        print_error "curl is required but not installed"
        exit 1
    fi
    
    if ! command -v jq > /dev/null; then
        print_error "jq is required but not installed"
        print_status "Install with: brew install jq (macOS) or apt-get install jq (Ubuntu)"
        exit 1
    fi
    
    if [ "$SKIP_POSTGRES" = false ]; then
        if ! command -v docker > /dev/null; then
            print_error "Docker is required but not installed"
            exit 1
        fi
        
        if ! docker info > /dev/null 2>&1; then
            print_error "Docker daemon is not running"
            exit 1
        fi
        
        start_postgres
    else
        print_warning "Skipping PostgreSQL setup - assuming it's running"
    fi
    
    if [ "$SKIP_BUILD" = false ]; then
        build_stormdb
    else
        print_warning "Skipping build - using existing binaries"
    fi
    
    start_stormdb
    
    test_basic_functionality
    
    if [ "$SKIP_TESTS" = false ]; then
        echo ""
        print_status "Running demonstration tests..."
        
        test_bulk_load
        echo ""
        test_tpcc
    else
        print_warning "Skipping demonstration tests"
    fi
    
    echo ""
    print_success "StormDB v2 demo completed successfully!"
    print_status "StormDB is still running at $STORMDB_URL"
    print_status "You can now run additional tests manually using the API"
    print_status "Press Ctrl+C to stop all services"
    
    # Keep running until interrupted
    wait $STORMDB_PID
}

# Run main function
main "$@"
