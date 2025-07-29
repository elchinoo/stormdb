# Load Testing Scripts Usage Guide

## Available Scripts

### Individual Environment Scripts:
- `hcp/epas16.sh` - Tests EPAS v16 environment
- `hcp/epas17.sh` - Tests EPAS v17 environment  
- `hcp/pge16.sh` - Tests PGE v16 environment
- `hcp/pge17.sh` - Tests PGE v17 environment

### Master Script:
- `hcp/run_all_tests.sh` - Runs all environment tests sequentially

## Test Configuration

### Workloads Tested:
- `ecommerce_mixed` - E-commerce platform simulation
- `imdb_mixed` - Movie database workload
- `tpcc` - TPC-C benchmark (OLTP)
- `realworld` - Social media platform simulation

### Worker Counts:
- 16 workers
- 36 workers  
- 64 workers
- 128 workers

### Test Duration:
- 60 minutes per test
- 60 second pause between worker configurations
- 120 second pause between workloads
- 300 second pause between environments (master script)

## Usage Examples

### Run single environment:
```bash
# Test EPAS v17 only
./hcp/epas17.sh

# Test PGE v16 only  
./hcp/pge16.sh
```

### Run all environments:
```bash
# Run complete test suite (will take ~16+ hours)
./hcp/run_all_tests.sh
```

### Run specific test manually:
```bash
# Test ecommerce workload with 64 workers on EPAS v17
./stormdb -c hcp/config_ecommerce_mixed_epas17.yaml --workers=64 --duration=60m
```

## Results Structure

### Individual Environment Results:
- `results_epas16/` - EPAS v16 test results
- `results_epas17/` - EPAS v17 test results  
- `results_pge16/` - PGE v16 test results
- `results_pge17/` - PGE v17 test results

### Result Files per Environment:
- `{workload}_{workers}workers_{timestamp}.log` - Individual test logs
- `test_summary.log` - Success/failure summary
- `final_summary.txt` - Complete environment summary

### Master Results:
- `results_master/master_log.txt` - Master script execution log
- `results_master/master_summary.txt` - Complete test suite summary

## Estimated Runtime

### Single Environment:
- 4 workloads × 4 worker configs × 60min = ~4 hours per environment
- Plus pause time: ~4.5 hours total per environment

### All Environments:
- 4 environments × 4.5 hours = ~18 hours total
- Sequential execution with extended pauses between environments

## Database Servers

- **EPAS v16**: p-rzjaqzb0yn-rw-external-31683a08ddd89515.elb.us-east-1.amazonaws.com
- **EPAS v17**: p-5z2mcdtt4g-rw-external-4742739c702557c6.elb.us-east-1.amazonaws.com  
- **PGE v16**: p-ys0nl9245c-rw-external-d6e5d894e2a130a6.elb.us-east-1.amazonaws.com
- **PGE v17**: p-kgd54g3gg7-rw-external-2e5e606c35f06e33.elb.us-east-1.amazonaws.com

All tests use credentials: edb_admin/mattdemo123!/edb_admin
