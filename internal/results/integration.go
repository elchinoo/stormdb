// Integration utilities for StormDB Database Backend
// This package provides helper functions to integrate test results storage

package results

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"
)

// StoreTestResults is a convenience function to store test results
// This should be called at the end of a test run
func StoreTestResults(ctx context.Context, backend *Backend, cfg *types.Config, metrics *types.Metrics, startTime, endTime time.Time) error {
	if backend == nil {
		return nil // Backend not configured
	}

	// Create test run record
	testRun := &TestRun{
		TestName:       getTestName(cfg),
		Workload:       cfg.Workload,
		Configuration:  buildConfigMap(cfg),
		StartTime:      startTime,
		EndTime:        endTime,
		Duration:       endTime.Sub(startTime),
		Workers:        cfg.Workers,
		Connections:    cfg.Connections,
		Scale:          cfg.Scale,
		TestMode:       getTestMode(cfg),
		Environment:    getEnvironment(cfg),
		DatabaseTarget: getDatabaseTarget(cfg),
		Version:        "1.0.0", // This should come from build info
		Status:         "completed",
		Notes:          getNotes(cfg),
		Tags:           getTags(cfg),
	}

	// Store in database
	return backend.StoreTestRun(ctx, testRun, metrics)
}

// getTestName extracts or generates a test name
func getTestName(cfg *types.Config) string {
	// Check if test metadata is available in config
	if testName, ok := cfg.TestMetadata["test_name"].(string); ok && testName != "" {
		return testName
	}

	// Generate a descriptive name
	return fmt.Sprintf("%s_scale_%d_workers_%d", cfg.Workload, cfg.Scale, cfg.Workers)
}

// buildConfigMap creates a map of configuration for storage
func buildConfigMap(cfg *types.Config) map[string]interface{} {
	configMap := map[string]interface{}{
		"workload":            cfg.Workload,
		"scale":               cfg.Scale,
		"workers":             cfg.Workers,
		"connections":         cfg.Connections,
		"duration":            cfg.Duration,
		"summary_interval":    cfg.SummaryInterval,
		"collect_pg_stats":    cfg.CollectPgStats,
		"pg_stats_statements": cfg.PgStatsStatements,
	}

	// Add progressive scaling config if enabled
	if cfg.Progressive.Enabled {
		configMap["progressive"] = map[string]interface{}{
			"enabled":         cfg.Progressive.Enabled,
			"strategy":        cfg.Progressive.Strategy,
			"min_workers":     cfg.Progressive.MinWorkers,
			"max_workers":     cfg.Progressive.MaxWorkers,
			"min_connections": cfg.Progressive.MinConns,
			"max_connections": cfg.Progressive.MaxConns,
			"bands":           cfg.Progressive.Bands,
			"test_duration":   cfg.Progressive.TestDuration,
		}
	}

	return configMap
}

// getTestMode determines the test mode
func getTestMode(cfg *types.Config) string {
	if cfg.Progressive.Enabled {
		return "progressive"
	}
	return "standard"
}

// getEnvironment extracts environment from config or defaults
func getEnvironment(cfg *types.Config) string {
	if env, ok := cfg.TestMetadata["environment"].(string); ok && env != "" {
		return env
	}
	return "unknown"
}

// getDatabaseTarget creates a description of the target database
func getDatabaseTarget(cfg *types.Config) string {
	if target, ok := cfg.TestMetadata["database_target"].(string); ok && target != "" {
		return target
	}
	return fmt.Sprintf("%s://%s:%d/%s", cfg.Database.Type, cfg.Database.Host, cfg.Database.Port, cfg.Database.Dbname)
}

// getNotes extracts notes from config
func getNotes(cfg *types.Config) string {
	if notes, ok := cfg.TestMetadata["notes"].(string); ok {
		return notes
	}
	return ""
}

// getTags extracts tags from config
func getTags(cfg *types.Config) []string {
	if tags, ok := cfg.TestMetadata["tags"].([]interface{}); ok {
		stringTags := make([]string, len(tags))
		for i, tag := range tags {
			if str, ok := tag.(string); ok {
				stringTags[i] = str
			}
		}
		return stringTags
	}
	return []string{}
}

// CreateBackendFromConfig creates a database backend from configuration
func CreateBackendFromConfig(cfg *types.Config) (*Backend, error) {
	if !cfg.ResultsBackend.Enabled {
		return nil, nil // Backend not configured
	}

	// Set defaults
	tablePrefix := cfg.ResultsBackend.TablePrefix
	if tablePrefix == "" {
		tablePrefix = "stormdb_"
	}

	metricsBatchSize := cfg.ResultsBackend.MetricsBatchSize
	if metricsBatchSize == 0 {
		metricsBatchSize = 1000
	}

	retentionDays := cfg.ResultsBackend.RetentionDays
	if retentionDays == 0 {
		retentionDays = 30 // Default 30 days retention
	}

	backendConfig := &BackendConfig{
		Host:             cfg.ResultsBackend.Host,
		Port:             cfg.ResultsBackend.Port,
		Database:         cfg.ResultsBackend.Database,
		Username:         cfg.ResultsBackend.Username,
		Password:         cfg.ResultsBackend.Password,
		SSLMode:          cfg.ResultsBackend.SSLMode,
		Enabled:          cfg.ResultsBackend.Enabled,
		RetentionDays:    retentionDays,
		StoreRawMetrics:  cfg.ResultsBackend.StoreRawMetrics,
		StorePgStats:     cfg.ResultsBackend.StorePgStats,
		MetricsBatchSize: metricsBatchSize,
		TablePrefix:      tablePrefix,
	}

	backend, err := NewBackend(backendConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create results backend: %w", err)
	}

	log.Printf("📊 Database backend enabled for test results storage")
	return backend, nil
}

// PerformMaintenance runs maintenance tasks on the backend
func PerformMaintenance(ctx context.Context, backend *Backend) error {
	if backend == nil {
		return nil
	}

	log.Printf("🧹 Performing database backend maintenance...")

	// Clean up old results
	if err := backend.CleanupOldResults(ctx); err != nil {
		return fmt.Errorf("failed to cleanup old results: %w", err)
	}

	// Additional maintenance tasks could be added here:
	// - VACUUM tables
	// - Update statistics
	// - Compress old data
	// - Generate reports

	return nil
}

// GetRecentTestRuns retrieves recent test runs for analysis
func GetRecentTestRuns(ctx context.Context, backend *Backend, workload string, limit int) ([]*TestRun, error) {
	if backend == nil {
		return nil, fmt.Errorf("backend not configured")
	}

	filters := map[string]interface{}{
		"limit": limit,
	}

	if workload != "" {
		filters["workload"] = workload
	}

	return backend.GetTestRuns(ctx, filters)
}

// CompareTestPerformance provides a simple comparison between test runs
func CompareTestPerformance(ctx context.Context, backend *Backend, testRunID1, testRunID2 int64) (map[string]interface{}, error) {
	if backend == nil {
		return nil, fmt.Errorf("backend not configured")
	}

	// Get results for both test runs
	results1, err := backend.GetTestResults(ctx, testRunID1)
	if err != nil {
		return nil, fmt.Errorf("failed to get results for test run %d: %w", testRunID1, err)
	}

	results2, err := backend.GetTestResults(ctx, testRunID2)
	if err != nil {
		return nil, fmt.Errorf("failed to get results for test run %d: %w", testRunID2, err)
	}

	// Calculate performance differences
	comparison := map[string]interface{}{
		"test_run_1":          testRunID1,
		"test_run_2":          testRunID2,
		"tps_improvement":     ((results2.TPS - results1.TPS) / results1.TPS) * 100,
		"latency_p95_change":  ((results2.P95LatencyMs - results1.P95LatencyMs) / results1.P95LatencyMs) * 100,
		"success_rate_change": results2.SuccessRate - results1.SuccessRate,
		"results_1":           results1,
		"results_2":           results2,
	}

	return comparison, nil
}
