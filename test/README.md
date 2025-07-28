# stormdb Test Suite

This directory contains a comprehensive test suite for the stormdb database load testing tool.

## Test Structure

```
test/
├── unit/                 # Unit tests (no external dependencies)
│   ├── config_test.go    # Configuration validation tests
│   ├── workload_test.go  # Workload factory tests
│   └── metrics_test.go   # Metrics calculation tests
├── integration/          # Integration tests (require database)
│   └── workload_integration_test.go  # End-to-end workload tests
├── load/                 # Load and performance tests
│   └── load_test.go      # Concurrency and stress tests
├── fixtures/             # Test configuration files
│   ├── valid_config.yaml
│   ├── invalid_*.yaml
│   └── ...
├── run_tests.sh         # Test runner script
└── README.md           # This file
```

## Test Categories

### 1. Unit Tests (`test/unit/`)

**Purpose**: Test individual components in isolation without external dependencies.

**Tests Include**:
- **Configuration Validation**: Ensures all configuration validation rules work correctly
- **Workload Factory**: Tests workload creation and type checking
- **Metrics Calculations**: Validates percentile and statistics calculations

**Run Command**:
```bash
./test/run_tests.sh unit
```

### 2. Integration Tests (`test/integration/`)

**Purpose**: Test complete workflows with real database connections.

**Requirements**: 
- Running PostgreSQL instance
- Proper database credentials

**Tests Include**:
- **Database Connectivity**: Basic connection and query validation
- **Workload Schema Setup**: Ensures each workload can create its required schema
- **Short Workload Execution**: Validates each workload can execute successfully

**Run Command**:
```bash
./test/run_tests.sh integration
```

### 3. Load Tests (`test/load/`)

**Purpose**: Test system behavior under various load conditions.

**Tests Include**:
- **Concurrent Workloads**: Multiple workloads running simultaneously
- **High Concurrency**: Many workers and connections
- **Stress Testing**: Extended duration tests (optional)
- **Memory Usage**: Tests for memory leaks (optional)

**Run Commands**:
```bash
./test/run_tests.sh load      # Basic load tests
./test/run_tests.sh stress    # Extended stress tests
```

## Running Tests

### Quick Start
```bash
# Run all standard tests
./test/run_tests.sh all

# Run only unit tests (fastest)
./test/run_tests.sh unit
```

### Database Setup for Integration/Load Tests

The integration and load tests require a PostgreSQL database. Configure using environment variables:

```bash
export STORMDB_TEST_HOST="localhost"
export STORMDB_TEST_DB="postgres"
export STORMDB_TEST_USER="postgres"
export STORMDB_TEST_PASSWORD="your_password"

./test/run_tests.sh integration
```

### Individual Test Commands

```bash
# Unit tests only
go test -v ./test/unit/...

# Integration tests only  
go test -v ./test/integration/...

# Load tests only
go test -v ./test/load/...

# With short mode (skips slow tests)
go test -short -v ./test/...
```

### Coverage Analysis

```bash
./test/run_tests.sh coverage
```

This generates:
- Console coverage summary
- `coverage.html` - Detailed HTML coverage report

## Test Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `STORMDB_TEST_HOST` | `localhost` | PostgreSQL host |
| `STORMDB_TEST_DB` | `postgres` | Database name |
| `STORMDB_TEST_USER` | `postgres` | Database user |
| `STORMDB_TEST_PASSWORD` | `` | Database password |
| `STORMDB_STRESS_TEST` | `` | Set to `1` to enable stress tests |
| `STORMDB_MEMORY_TEST` | `` | Set to `1` to enable memory tests |

### Test Fixtures

Test configuration files in `fixtures/` directory:

- `valid_config.yaml` - Working configuration for tests
- `invalid_*.yaml` - Various invalid configurations for validation testing

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    - name: Run tests
      env:
        STORMDB_TEST_HOST: localhost
        STORMDB_TEST_USER: postgres  
        STORMDB_TEST_PASSWORD: postgres
      run: ./test/run_tests.sh all
```

## Test Development Guidelines

### Adding New Tests

1. **Unit Tests**: Add to appropriate `test/unit/*.go` file
2. **Integration Tests**: Add to `test/integration/*.go`
3. **Load Tests**: Add to `test/load/*.go`

### Test Naming Conventions

- Test functions: `TestFunctionName`
- Test files: `*_test.go`
- Helper functions: `testHelperName` or `helperName`

### Best Practices

1. **Isolation**: Each test should be independent
2. **Cleanup**: Always cleanup database resources in integration tests
3. **Error Messages**: Provide clear, actionable error messages
4. **Skip Conditions**: Use `t.Skip()` when dependencies aren't available
5. **Timeouts**: Use context timeouts for long-running tests

### Example Test Structure

```go
func TestFeatureName(t *testing.T) {
    // Setup
    cfg := getTestConfig(t)
    
    // Test execution
    result, err := functionUnderTest(cfg)
    
    // Validation
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }
    
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
    
    // Cleanup (if needed)
    cleanup()
}
```

## Troubleshooting

### Common Issues

1. **Database Connection Failures**
   - Verify PostgreSQL is running
   - Check connection parameters
   - Ensure database exists and user has permissions

2. **pgvector Extension Missing**
   - Install pgvector extension: `CREATE EXTENSION vector;`
   - Some vector tests will be skipped if extension is unavailable

3. **Test Timeouts**
   - Increase timeout for slow environments
   - Use `-timeout` flag: `go test -timeout 60s`

4. **Permission Errors**
   - Ensure test user has CREATE/DROP table permissions
   - Consider using a dedicated test database

### Debug Mode

Enable verbose output:
```bash
go test -v -run TestSpecificTest ./test/unit/
```

Add debug logging in tests:
```go
t.Logf("Debug info: %v", value)
```

## Future Test Enhancements

Planned additions to the test suite:

1. **Benchmark Tests**: Performance regression detection
2. **Chaos Testing**: Database failure scenarios  
3. **Multi-Database Tests**: Different PostgreSQL versions
4. **Security Tests**: SQL injection, connection security
5. **Configuration Fuzzing**: Random configuration testing
6. **End-to-End CLI Tests**: Full CLI integration testing
