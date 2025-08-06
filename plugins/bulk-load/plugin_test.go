package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/elchinoo/stormdb/core"
)

// MockLogger implements the core.Logger interface for testing
type MockLogger struct{}

func (m *MockLogger) Debug(msg string, fields ...core.Field)              {}
func (m *MockLogger) Info(msg string, fields ...core.Field)               {}
func (m *MockLogger) Warn(msg string, fields ...core.Field)               {}
func (m *MockLogger) Error(msg string, fields ...core.Field)              {}
func (m *MockLogger) WithFields(fields ...core.Field) core.Logger         { return m }
func (m *MockLogger) WithPlugin(pluginName string) core.Logger            { return m }
func (m *MockLogger) WithStorage(storage core.StorageManager) core.Logger { return m }

// MockCoreServices provides mock implementations for testing
type MockCoreServices struct {
	Logger core.Logger
}

func TestBulkLoadPlugin_Metadata(t *testing.T) {
	plugin := &BulkLoadPlugin{}
	metadata := plugin.Metadata()

	if metadata.Name != "bulk-load" {
		t.Errorf("Expected plugin name 'bulk-load', got '%s'", metadata.Name)
	}

	if metadata.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", metadata.Version)
	}

	if len(metadata.TestTypes) == 0 {
		t.Error("Expected test types to be defined")
	}

	expectedTestTypes := []string{"bulk_insert", "batch_performance", "load_testing"}
	for i, expected := range expectedTestTypes {
		if i >= len(metadata.TestTypes) || metadata.TestTypes[i] != expected {
			t.Errorf("Expected test type '%s' at index %d", expected, i)
		}
	}

	// Validate JSON schema
	var schema map[string]interface{}
	if err := json.Unmarshal([]byte(metadata.ConfigSchema), &schema); err != nil {
		t.Errorf("Invalid JSON schema: %v", err)
	}
}

func TestBulkLoadPlugin_Initialize(t *testing.T) {
	plugin := &BulkLoadPlugin{}
	coreServices := &MockCoreServices{
		Logger: &MockLogger{},
	}

	ctx := context.Background()
	err := plugin.Initialize(ctx, &core.CoreServices{
		Logger: coreServices.Logger,
	})

	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}

	if plugin.core == nil {
		t.Error("Core services not set")
	}

	if plugin.logger == nil {
		t.Error("Logger not set")
	}

	if plugin.metrics == nil {
		t.Error("Metrics not initialized")
	}
}

func TestBulkLoadPlugin_Validate(t *testing.T) {
	plugin := &BulkLoadPlugin{}
	coreServices := &MockCoreServices{
		Logger: &MockLogger{},
	}

	ctx := context.Background()
	plugin.Initialize(ctx, &core.CoreServices{
		Logger: coreServices.Logger,
	})

	tests := []struct {
		name      string
		config    map[string]interface{}
		expectErr bool
	}{
		{
			name: "valid_minimal_config",
			config: map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"database": "testdb",
				"username": "testuser",
				"password": "testpass",
			},
			expectErr: false,
		},
		{
			name: "valid_full_config",
			config: map[string]interface{}{
				"host":          "localhost",
				"port":          5432,
				"database":      "testdb",
				"username":      "testuser",
				"password":      "testpass",
				"ssl_mode":      "disable",
				"batch_sizes":   []interface{}{1, 100, 1000},
				"connections":   10,
				"duration":      "2m",
				"warmup_time":   "10s",
				"think_time":    "5ms",
				"table_name":    "test_table",
				"drop_table":    true,
				"generate_data": true,
				"data_columns":  5,
				"verbose":       true,
			},
			expectErr: false,
		},
		{
			name: "missing_host",
			config: map[string]interface{}{
				"port":     5432,
				"database": "testdb",
				"username": "testuser",
				"password": "testpass",
			},
			expectErr: true,
		},
		{
			name: "invalid_port",
			config: map[string]interface{}{
				"host":     "localhost",
				"port":     70000,
				"database": "testdb",
				"username": "testuser",
				"password": "testpass",
			},
			expectErr: true,
		},
		{
			name: "missing_database",
			config: map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"username": "testuser",
				"password": "testpass",
			},
			expectErr: true,
		},
		{
			name: "invalid_connections_too_low",
			config: map[string]interface{}{
				"host":        "localhost",
				"port":        5432,
				"database":    "testdb",
				"username":    "testuser",
				"password":    "testpass",
				"connections": 0,
			},
			expectErr: true,
		},
		{
			name: "invalid_connections_too_high",
			config: map[string]interface{}{
				"host":        "localhost",
				"port":        5432,
				"database":    "testdb",
				"username":    "testuser",
				"password":    "testpass",
				"connections": 1001,
			},
			expectErr: true,
		},
		{
			name: "invalid_data_columns_too_low",
			config: map[string]interface{}{
				"host":         "localhost",
				"port":         5432,
				"database":     "testdb",
				"username":     "testuser",
				"password":     "testpass",
				"data_columns": 0,
			},
			expectErr: true,
		},
		{
			name: "invalid_data_columns_too_high",
			config: map[string]interface{}{
				"host":         "localhost",
				"port":         5432,
				"database":     "testdb",
				"username":     "testuser",
				"password":     "testpass",
				"data_columns": 51,
			},
			expectErr: true,
		},
		{
			name: "invalid_batch_size",
			config: map[string]interface{}{
				"host":        "localhost",
				"port":        5432,
				"database":    "testdb",
				"username":    "testuser",
				"password":    "testpass",
				"batch_sizes": []interface{}{1, 0, 1000},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugin.Validate(tt.config)
			if tt.expectErr && err == nil {
				t.Errorf("Expected error for test case '%s', but got none", tt.name)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error for test case '%s': %v", tt.name, err)
			}
		})
	}
}

func TestBulkLoadPlugin_ConfigDefaults(t *testing.T) {
	plugin := &BulkLoadPlugin{}
	coreServices := &MockCoreServices{
		Logger: &MockLogger{},
	}

	ctx := context.Background()
	plugin.Initialize(ctx, &core.CoreServices{
		Logger: coreServices.Logger,
	})

	config := map[string]interface{}{
		"host":     "localhost",
		"port":     5432,
		"database": "testdb",
		"username": "testuser",
		"password": "testpass",
	}

	err := plugin.Validate(config)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	// Check defaults
	if len(plugin.config.BatchSizes) != 4 {
		t.Errorf("Expected 4 default batch sizes, got %d", len(plugin.config.BatchSizes))
	}

	expectedBatchSizes := []int{1, 1000, 10000, 50000}
	for i, expected := range expectedBatchSizes {
		if plugin.config.BatchSizes[i] != expected {
			t.Errorf("Expected batch size %d at index %d, got %d", expected, i, plugin.config.BatchSizes[i])
		}
	}

	if plugin.config.Connections != 20 {
		t.Errorf("Expected default connections 20, got %d", plugin.config.Connections)
	}

	if plugin.config.Duration != 5*time.Minute {
		t.Errorf("Expected default duration 5m, got %v", plugin.config.Duration)
	}

	if plugin.config.WarmupTime != 30*time.Second {
		t.Errorf("Expected default warmup time 30s, got %v", plugin.config.WarmupTime)
	}

	if plugin.config.ThinkTime != 10*time.Millisecond {
		t.Errorf("Expected default think time 10ms, got %v", plugin.config.ThinkTime)
	}

	if plugin.config.TableName != "bulk_test_data" {
		t.Errorf("Expected default table name 'bulk_test_data', got '%s'", plugin.config.TableName)
	}

	if plugin.config.DataColumns != 10 {
		t.Errorf("Expected default data columns 10, got %d", plugin.config.DataColumns)
	}

	if plugin.config.SSLMode != "disable" {
		t.Errorf("Expected default SSL mode 'disable', got '%s'", plugin.config.SSLMode)
	}
}

func TestBulkLoadPlugin_Cleanup(t *testing.T) {
	plugin := &BulkLoadPlugin{}
	coreServices := &MockCoreServices{
		Logger: &MockLogger{},
	}

	ctx := context.Background()
	plugin.Initialize(ctx, &core.CoreServices{
		Logger: coreServices.Logger,
	})

	err := plugin.Cleanup(ctx)
	if err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}
}

func TestWorkerStats(t *testing.T) {
	stats := &WorkerStats{
		WorkerID:   1,
		MinLatency: time.Hour, // Initialize with high value
	}

	// Test initial state
	if stats.Transactions != 0 {
		t.Error("Expected initial transactions to be 0")
	}

	if stats.RowsInserted != 0 {
		t.Error("Expected initial rows inserted to be 0")
	}

	if stats.Errors != 0 {
		t.Error("Expected initial errors to be 0")
	}

	if stats.MinLatency != time.Hour {
		t.Error("Expected MinLatency to be initialized to 1 hour")
	}
}

func TestBatchResult(t *testing.T) {
	result := &BatchResult{
		BatchSize:         1000,
		Connections:       20,
		TotalTransactions: 100,
		TotalRowsInserted: 100000,
		DurationSeconds:   60.0,
	}

	// Calculate derived metrics
	result.TransactionsPerSec = float64(result.TotalTransactions) / result.DurationSeconds
	result.RowsPerSec = float64(result.TotalRowsInserted) / result.DurationSeconds

	expectedTPS := 100.0 / 60.0
	if result.TransactionsPerSec != expectedTPS {
		t.Errorf("Expected TPS %.2f, got %.2f", expectedTPS, result.TransactionsPerSec)
	}

	expectedRowsPerSec := 100000.0 / 60.0
	if result.RowsPerSec != expectedRowsPerSec {
		t.Errorf("Expected rows per second %.2f, got %.2f", expectedRowsPerSec, result.RowsPerSec)
	}
}

// Benchmark tests
func BenchmarkConfigValidation(b *testing.B) {
	plugin := &BulkLoadPlugin{}
	coreServices := &MockCoreServices{
		Logger: &MockLogger{},
	}

	ctx := context.Background()
	plugin.Initialize(ctx, &core.CoreServices{
		Logger: coreServices.Logger,
	})

	config := map[string]interface{}{
		"host":        "localhost",
		"port":        5432,
		"database":    "testdb",
		"username":    "testuser",
		"password":    "testpass",
		"batch_sizes": []int{1, 1000, 10000, 50000},
		"connections": 20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin.Validate(config)
	}
}

func BenchmarkMetricsUpdate(b *testing.B) {
	metrics := &BulkLoadMetrics{
		BatchResults: make([]BatchResult, 0),
	}

	result := BatchResult{
		BatchSize:         1000,
		TotalTransactions: 100,
		TotalRowsInserted: 100000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.mu.Lock()
		metrics.BatchResults = append(metrics.BatchResults, result)
		metrics.TotalTransactions += result.TotalTransactions
		metrics.TotalRowsInserted += result.TotalRowsInserted
		metrics.mu.Unlock()
	}
}
