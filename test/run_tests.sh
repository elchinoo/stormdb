#!/bin/bash

# Test runner script for stormdb
# This script runs different test suites based on the provided argument

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default test database settings
export STORMDB_TEST_HOST=${STORMDB_TEST_HOST:-"localhost"}
export STORMDB_TEST_DB=${STORMDB_TEST_DB:-"postgres"}
export STORMDB_TEST_USER=${STORMDB_TEST_USER:-"postgres"}
export STORMDB_TEST_PASSWORD=${STORMDB_TEST_PASSWORD:-""}

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

run_unit_tests() {
    print_header "Running Unit Tests"
    
    echo "Running configuration tests..."
    go test -v ./test/unit/config_test.go || {
        print_error "Configuration tests failed"
        return 1
    }
    
    echo "Running workload factory tests..."
    go test -v ./test/unit/workload_test.go || {
        print_error "Workload factory tests failed"
        return 1
    }
    
    echo "Running metrics tests..."
    go test -v ./test/unit/metrics_test.go || {
        print_error "Metrics tests failed"
        return 1
    }
    
    print_success "All unit tests passed"
}

run_integration_tests() {
    print_header "Running Integration Tests"
    
    # Check database connectivity first
    echo "Testing database connectivity..."
    if ! pg_isready -h "$STORMDB_TEST_HOST" -U "$STORMDB_TEST_USER" -d "$STORMDB_TEST_DB" >/dev/null 2>&1; then
        print_warning "Database not available, skipping integration tests"
        print_warning "To run integration tests, ensure PostgreSQL is running and accessible"
        return 0
    fi
    
    go test -v ./test/integration/... || {
        print_error "Integration tests failed"
        return 1
    }
    
    print_success "All integration tests passed"
}

run_load_tests() {
    print_header "Running Load Tests"
    
    # Check database connectivity first
    if ! pg_isready -h "$STORMDB_TEST_HOST" -U "$STORMDB_TEST_USER" -d "$STORMDB_TEST_DB" >/dev/null 2>&1; then
        print_warning "Database not available, skipping load tests"
        return 0
    fi
    
    echo "Running basic load tests..."
    go test -v ./test/load/... || {
        print_error "Load tests failed"
        return 1
    }
    
    print_success "All load tests passed"
}

run_stress_tests() {
    print_header "Running Stress Tests"
    
    if ! pg_isready -h "$STORMDB_TEST_HOST" -U "$STORMDB_TEST_USER" -d "$STORMDB_TEST_DB" >/dev/null 2>&1; then
        print_warning "Database not available, skipping stress tests"
        return 0
    fi
    
    export STORMDB_STRESS_TEST=1
    export STORMDB_MEMORY_TEST=1
    
    echo "Running stress and memory tests (this may take several minutes)..."
    go test -v -timeout=300s ./test/load/... || {
        print_error "Stress tests failed"
        return 1
    }
    
    print_success "All stress tests passed"
}

run_coverage() {
    print_header "Running Test Coverage Analysis"
    
    echo "Generating coverage report..."
    go test -coverprofile=coverage.out ./test/unit/... ./internal/... || {
        print_error "Coverage analysis failed"
        return 1
    }
    
    echo "Coverage summary:"
    go tool cover -func=coverage.out
    
    echo "Generating HTML coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    
    print_success "Coverage report generated: coverage.html"
}

run_all_tests() {
    print_header "Running All Tests"
    
    run_unit_tests || return 1
    run_integration_tests || return 1
    run_load_tests || return 1
    
    print_success "All tests completed successfully"
}

show_help() {
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  unit        Run unit tests only"
    echo "  integration Run integration tests (requires database)"
    echo "  load        Run load tests (requires database)"
    echo "  stress      Run stress tests (requires database, takes time)"
    echo "  coverage    Run tests with coverage analysis"
    echo "  all         Run unit, integration, and load tests"
    echo "  help        Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  STORMDB_TEST_HOST     PostgreSQL host (default: localhost)"
    echo "  STORMDB_TEST_DB       PostgreSQL database (default: postgres)"
    echo "  STORMDB_TEST_USER     PostgreSQL user (default: postgres)"
    echo "  STORMDB_TEST_PASSWORD PostgreSQL password (default: empty)"
    echo ""
    echo "Examples:"
    echo "  $0 unit                    # Run only unit tests"
    echo "  $0 all                     # Run all standard tests"
    echo "  STORMDB_TEST_HOST=db $0 integration  # Run integration tests against 'db' host"
}

# Main script logic
case "${1:-all}" in
    unit)
        run_unit_tests
        ;;
    integration)
        run_integration_tests
        ;;
    load)
        run_load_tests
        ;;
    stress)
        run_stress_tests
        ;;
    coverage)
        run_coverage
        ;;
    all)
        run_all_tests
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
