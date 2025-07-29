#!/bin/bash
# pge16.sh - Progressive load testing for PGE v16
# Tests all workloads with increasing worker counts

set -e

# Configuration
WORKERS=(16 36 64 128)
WORKLOADS=("ecommerce_mixed" "imdb_mixed" "tpcc" "realworld")
DURATION="60m"
RESULTS_DIR="results_pge16"
SERVER="pge16"

# Create results directory
mkdir -p "$RESULTS_DIR"

echo "=========================================="
echo "PGE v16 Progressive Load Testing Started"
echo "Server: p-ys0nl9245c-rw-external-d6e5d894e2a130a6.elb.us-east-1.amazonaws.com"
echo "Date: $(date)"
echo "=========================================="

# Function to run a single test
run_test() {
    local workload=$1
    local workers=$2
    local config_file="hcp/config_${workload}_pge16.yaml"
    local result_file="${RESULTS_DIR}/${workload}_${workers}workers_$(date +%Y%m%d_%H%M%S).log"
    
    echo "Testing $workload with $workers workers..."
    echo "Config: $config_file"
    echo "Output: $result_file"
    
    # Run the test
    if ./stormdb -c "$config_file" --workers="$workers" --duration="$DURATION" > "$result_file" 2>&1; then
        echo "✅ $workload with $workers workers completed successfully"
        # Extract key metrics and append to summary
        echo "$(date): $workload - $workers workers - SUCCESS" >> "${RESULTS_DIR}/test_summary.log"
    else
        echo "❌ $workload with $workers workers failed"
        echo "$(date): $workload - $workers workers - FAILED" >> "${RESULTS_DIR}/final_summary.txt"
    fi
    
    echo "Pausing for analysis..."
    sleep 60
    echo ""
}

# Main testing loop
for workload in "${WORKLOADS[@]}"; do
    echo "===================="
    echo "Testing workload: $workload"
    echo "===================="
    
    for workers in "${WORKERS[@]}"; do
        run_test "$workload" "$workers"
    done
    
    echo "Completed all worker configurations for $workload"
    echo "Extended pause before next workload..."
    sleep 120
    echo ""
done

echo "=========================================="
echo "PGE v16 Progressive Load Testing Complete"
echo "Results saved in: $RESULTS_DIR/"
echo "Summary: ${RESULTS_DIR}/test_summary.log"
echo "=========================================="

# Generate final summary
echo "Test Summary for PGE v16:" > "${RESULTS_DIR}/final_summary.txt"
echo "Server: p-ys0nl9245c-rw-external-d6e5d894e2a130a6.elb.us-east-1.amazonaws.com" >> "${RESULTS_DIR}/final_summary.txt"
echo "Date: $(date)" >> "${RESULTS_DIR}/final_summary.txt"
echo "Duration per test: $DURATION" >> "${RESULTS_DIR}/final_summary.txt"
echo "Worker configurations: ${WORKERS[*]}" >> "${RESULTS_DIR}/final_summary.txt"
echo "Workloads tested: ${WORKLOADS[*]}" >> "${RESULTS_DIR}/final_summary.txt"
echo "" >> "${RESULTS_DIR}/final_summary.txt"
echo "Detailed results:" >> "${RESULTS_DIR}/final_summary.txt"
cat "${RESULTS_DIR}/test_summary.log" >> "${RESULTS_DIR}/final_summary.txt"

echo "Final summary saved to: ${RESULTS_DIR}/final_summary.txt"
