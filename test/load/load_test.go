package load_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/elchinoo/stormdb/internal/database"
	"github.com/elchinoo/stormdb/internal/workload"
	"github.com/elchinoo/stormdb/pkg/types"
)

// TestConcurrentWorkloads tests multiple workloads running concurrently
func TestConcurrentWorkloads(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cfg := getLoadTestConfig(t)
	cfg.Duration = "5s"
	cfg.Workers = 4
	cfg.Connections = 8

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	results := make(chan testResult, 3)

	workloads := []string{"simple", "tpcc"}

	for _, workloadType := range workloads {
		wg.Add(1)
		go func(wt string) {
			defer wg.Done()

			testCfg := *cfg
			testCfg.Workload = wt
			if wt == "tpcc" {
				testCfg.Scale = 1 // Small scale for load test
			} else {
				testCfg.Scale = 100
			}

			result := runWorkloadLoadTest(ctx, t, &testCfg)
			results <- testResult{
				workload: wt,
				metrics:  result,
			}
		}(workloadType)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var totalTPS int64
	var totalErrors int64
	var workloadsRun int
	for result := range results {
		t.Logf("Workload %s: TPS=%d, Errors=%d",
			result.workload, result.metrics.TPS, result.metrics.Errors)
		totalTPS += result.metrics.TPS
		totalErrors += result.metrics.Errors
		if result.metrics.TPS > 0 || result.metrics.Errors > 0 {
			workloadsRun++
		}
	}

	// Skip test if no workloads could run (likely no database connection)
	if workloadsRun == 0 {
		t.Skip("No workloads could run - likely no database connection available")
	}

	t.Logf("Total combined TPS: %d", totalTPS)
}

// TestHighConcurrency tests workload under high concurrency
func TestHighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cfg := getLoadTestConfig(t)
	cfg.Workload = "simple"
	cfg.Scale = 1000
	cfg.Duration = "10s"
	cfg.Workers = 20
	cfg.Connections = 30

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	metrics := runWorkloadLoadTest(ctx, t, cfg)

	// Skip test if no database connection is available
	if metrics.TPS == 0 && metrics.Errors == 0 {
		t.Skip("No database connection available for load testing")
	}

	// Validate high concurrency results
	if metrics.TPS < 100 {
		t.Errorf("Expected TPS > 100 under high concurrency, got %d", metrics.TPS)
	}

	if metrics.Errors > metrics.TPS/10 {
		t.Errorf("Too many errors: %d errors vs %d total transactions",
			metrics.Errors, metrics.TPS)
	}

	t.Logf("High concurrency test: TPS=%d, Errors=%d, Workers=%d",
		metrics.TPS, metrics.Errors, cfg.Workers)
}

// TestStressTest runs a longer stress test
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// Only run if explicitly requested
	if os.Getenv("STORMDB_STRESS_TEST") == "" {
		t.Skip("Set STORMDB_STRESS_TEST=1 to run stress tests")
	}

	cfg := getLoadTestConfig(t)
	cfg.Workload = "simple"
	cfg.Scale = 10000
	cfg.Duration = "60s"
	cfg.Workers = 10
	cfg.Connections = 20

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	startTime := time.Now()
	metrics := runWorkloadLoadTest(ctx, t, cfg)
	duration := time.Since(startTime)

	// Validate stress test results
	expectedMinTPS := int64(50) // Minimum acceptable TPS
	if metrics.TPS < expectedMinTPS {
		t.Errorf("Stress test TPS too low: got %d, expected >= %d",
			metrics.TPS, expectedMinTPS)
	}

	// Error rate should be reasonable
	errorRate := float64(metrics.Errors) / float64(metrics.TPS) * 100
	if errorRate > 5.0 {
		t.Errorf("Stress test error rate too high: %.2f%%", errorRate)
	}

	t.Logf("Stress test completed in %v: TPS=%d, Errors=%d (%.2f%% error rate)",
		duration, metrics.TPS, metrics.Errors, errorRate)
}

// TestMemoryUsage tests for memory leaks during extended execution
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	if os.Getenv("STORMDB_MEMORY_TEST") == "" {
		t.Skip("Set STORMDB_MEMORY_TEST=1 to run memory tests")
	}

	cfg := getLoadTestConfig(t)
	cfg.Workload = "vector_1024"
	cfg.Scale = 1000
	cfg.Duration = "30s"
	cfg.Workers = 4
	cfg.Connections = 8

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Check if pgvector is available
	db, err := database.NewPostgres(cfg)
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}
	defer db.Close()

	var extName string
	err = db.Pool.QueryRow(ctx, "SELECT extname FROM pg_extension WHERE extname = 'vector'").Scan(&extName)
	if err != nil {
		t.Skip("pgvector extension not available, skipping memory test")
	}

	metrics := runWorkloadLoadTest(ctx, t, cfg)

	// Basic validation - mainly checking we don't crash
	if metrics.TPS == 0 && metrics.Errors == 0 {
		t.Error("Expected some activity in memory test")
	}

	t.Logf("Memory test completed: TPS=%d, Errors=%d", metrics.TPS, metrics.Errors)
}

type testResult struct {
	workload string
	metrics  *types.Metrics
}

// Helper function to run a workload load test
func runWorkloadLoadTest(ctx context.Context, t *testing.T, cfg *types.Config) *types.Metrics {
	db, err := database.NewPostgres(cfg)
	if err != nil {
		t.Logf("Could not connect to test database: %v", err)
		// Return empty metrics instead of skipping
		return &types.Metrics{
			ErrorTypes: make(map[string]int64),
		}
	}
	defer db.Close()

	// Create workload factory
	factory, err := workload.NewFactory(cfg)
	if err != nil {
		t.Fatalf("Failed to create workload factory: %v", err)
	}
	defer func() {
		_ = factory.Cleanup()
	}()

	if err := factory.Initialize(); err != nil {
		t.Fatalf("Failed to initialize workload factory: %v", err)
	}

	w, err := factory.Get(cfg.Workload)
	if err != nil {
		t.Fatalf("Failed to get workload: %v", err)
	}

	// Setup
	err = w.Cleanup(ctx, db.Pool, cfg)
	if err != nil {
		t.Fatalf("Failed to cleanup workload: %v", err)
	}

	// Run test
	metrics := &types.Metrics{
		ErrorTypes: make(map[string]int64),
	}

	err = w.Run(ctx, db.Pool, cfg, metrics)
	if err != nil {
		t.Fatalf("Failed to run workload: %v", err)
	}

	return metrics
}

// Helper function to get load test configuration
func getLoadTestConfig(_ *testing.T) *types.Config {
	cfg := &types.Config{
		Database: struct {
			Type     string `mapstructure:"type"`
			Host     string `mapstructure:"host"`
			Port     int    `mapstructure:"port"`
			Dbname   string `mapstructure:"dbname"`
			Username string `mapstructure:"username"`
			Password string `mapstructure:"password"`
			Sslmode  string `mapstructure:"sslmode"`
		}{
			Type:     "postgres",
			Host:     getEnvOrDefault("STORMDB_TEST_HOST", "localhost"),
			Port:     5432,
			Dbname:   getEnvOrDefault("STORMDB_TEST_DB", "postgres"),
			Username: getEnvOrDefault("STORMDB_TEST_USER", "postgres"),
			Password: getEnvOrDefault("STORMDB_TEST_PASSWORD", ""),
			Sslmode:  "disable",
		},
		Workload:    "simple",
		Scale:       100,
		Duration:    "5s",
		Workers:     2,
		Connections: 4,
	}

	return cfg
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
