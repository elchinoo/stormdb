#!/bin/bash
# run_all_tests.sh - Master script to run all progressive load tests
# Executes tests for all database environments

set -e

SCRIPT_DIR="hcp"
SCRIPTS=("epas16.sh" "epas17.sh" "pge16.sh" "pge17.sh")
MASTER_RESULTS_DIR="results_master"

# Create master results directory
mkdir -p "$MASTER_RESULTS_DIR"

echo "==========================================="
echo "Master Progressive Load Testing Started"
echo "Date: $(date)"
echo "Testing all environments: EPAS v16/v17, PGE v16/v17"
echo "==========================================="

# Log start time
echo "Master test started at: $(date)" > "${MASTER_RESULTS_DIR}/master_log.txt"

for script in "${SCRIPTS[@]}"; do
    script_path="${SCRIPT_DIR}/${script}"
    
    if [[ -f "$script_path" ]]; then
        echo "=========================================="
        echo "Starting test script: $script"
        echo "=========================================="
        
        # Make script executable
        chmod +x "$script_path"
        
        # Run the script and log results
        if ./"$script_path"; then
            echo "✅ $script completed successfully"
            echo "$(date): $script - SUCCESS" >> "${MASTER_RESULTS_DIR}/master_log.txt"
        else
            echo "❌ $script failed"
            echo "$(date): $script - FAILED" >> "${MASTER_RESULTS_DIR}/master_log.txt"
        fi
        
        echo "Extended pause between environments..."
        sleep 300  # 5 minute pause between environments
        echo ""
    else
        echo "❌ Script not found: $script_path"
        echo "$(date): $script - NOT FOUND" >> "${MASTER_RESULTS_DIR}/master_log.txt"
    fi
done

echo "==========================================="
echo "Master Progressive Load Testing Complete"
echo "Total duration: $(date)"
echo "==========================================="

# Generate master summary
echo "Master Test Summary:" > "${MASTER_RESULTS_DIR}/master_summary.txt"
echo "Start time: $(head -n1 ${MASTER_RESULTS_DIR}/master_log.txt)" >> "${MASTER_RESULTS_DIR}/master_summary.txt"
echo "End time: $(date)" >> "${MASTER_RESULTS_DIR}/master_summary.txt"
echo "Environments tested: EPAS v16, EPAS v17, PGE v16, PGE v17" >> "${MASTER_RESULTS_DIR}/master_summary.txt"
echo "Workloads per environment: ecommerce_mixed, imdb_mixed, tpcc, realworld" >> "${MASTER_RESULTS_DIR}/master_summary.txt"
echo "Worker configurations: 16, 36, 64, 128" >> "${MASTER_RESULTS_DIR}/master_summary.txt"
echo "Duration per test: 60m" >> "${MASTER_RESULTS_DIR}/master_summary.txt"
echo "" >> "${MASTER_RESULTS_DIR}/master_summary.txt"
echo "Script execution results:" >> "${MASTER_RESULTS_DIR}/master_summary.txt"
cat "${MASTER_RESULTS_DIR}/master_log.txt" >> "${MASTER_RESULTS_DIR}/master_summary.txt"

echo "Master summary saved to: ${MASTER_RESULTS_DIR}/master_summary.txt"
echo "Individual results available in: results_epas16/, results_epas17/, results_pge16/, results_pge17/"
