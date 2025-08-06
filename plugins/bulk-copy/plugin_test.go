package main

import (
	"context"
	"testing"
	"time"

	"github.com/elchinoo/stormdb/core"
)

// MockLogger implements the core.Logger interface for testing
type MockLogger struct {
	logs []string
}

func (m *MockLogger) Info(msg string, fields ...core.Field) {
	m.logs = append(m.logs, "INFO: "+msg)
}

func (m *MockLogger) Error(msg string, fields ...core.Field) {
	m.logs = append(m.logs, "ERROR: "+msg)
}

func (m *MockLogger) Warn(msg string, fields ...core.Field) {
	m.logs = append(m.logs, "WARN: "+msg)
}

func (m *MockLogger) Debug(msg string, fields ...core.Field) {
	m.logs = append(m.logs, "DEBUG: "+msg)
}

func (m *MockLogger) WithPlugin(pluginName string) core.Logger {
	return m
}

func (m *MockLogger) WithFields(fields ...core.Field) core.Logger {
	return m
}

func (m *MockLogger) WithStorage(storage core.StorageManager) core.Logger {
	return m
}

// MockStorage implements basic storage interface for testing
type MockStorage struct{}

func (m *MockStorage) CreateTestRun(ctx context.Context, run *core.TestRun) (int64, error) {
	return 1, nil
}

func (m *MockStorage) UpdateTestRunStatus(ctx context.Context, runID int64, status core.ServiceStatus) error {
	return nil
}

func (m *MockStorage) GetTestRun(ctx context.Context, runID int64) (*core.TestRun, error) {
	return &core.TestRun{ID: runID}, nil
}

func (m *MockStorage) ListTestRuns(ctx context.Context, limit, offset int) ([]core.TestRun, error) {
	return []core.TestRun{}, nil
}

func (m *MockStorage) StoreResults(ctx context.Context, results []core.TestResult) error {
	return nil
}

func (m *MockStorage) GetResults(ctx context.Context, runID int64) ([]core.TestResult, error) {
	return []core.TestResult{}, nil
}

func (m *MockStorage) GetResultsByMetric(ctx context.Context, metricCode string, limit int) ([]core.TestResult, error) {
	return []core.TestResult{}, nil
}

func (m *MockStorage) RegisterTestType(ctx context.Context, code, name, description string) (int, error) {
	return 1, nil
}

func (m *MockStorage) GetTestType(ctx context.Context, code string) (*core.TestType, error) {
	return &core.TestType{ID: 1, Code: code}, nil
}

func (m *MockStorage) RegisterPlugin(ctx context.Context, metadata core.PluginMetadata) (int, error) {
	return 1, nil
}

func (m *MockStorage) GetPlugin(ctx context.Context, name, version string) (*core.PluginMetadata, error) {
	return &core.PluginMetadata{ID: 1, Name: name, Version: version}, nil
}

func (m *MockStorage) RegisterMetric(ctx context.Context, code, description, unit string) (int, error) {
	return 1, nil
}

func (m *MockStorage) GetMetric(ctx context.Context, code string) (*core.TestMetric, error) {
	return &core.TestMetric{
		ID:   1,
		Code: code,
	}, nil
}

func (m *MockStorage) StoreLog(ctx context.Context, entry core.LogEntry) error {
	return nil
}

func (m *MockStorage) GetTestRunLogs(ctx context.Context, testRunID int64, limit int) ([]core.LogEntry, error) {
	return []core.LogEntry{}, nil
}

func (m *MockStorage) FixStuckTests(ctx context.Context) (int64, error) {
	return 0, nil
}

// MockCoreServices provides mock core services for testing
type MockCoreServices struct {
	Logger  core.Logger
	Storage core.StorageManager
}

func newMockCoreServices() *MockCoreServices {
	return &MockCoreServices{
		Logger:  &MockLogger{},
		Storage: &MockStorage{},
	}
}

func TestBulkCopyPlugin_Metadata(t *testing.T) {
	plugin := &BulkCopyPlugin{}
	metadata := plugin.Metadata()

	if metadata.Name != "bulk-copy" {
		t.Errorf("Expected plugin name 'bulk-copy', got '%s'", metadata.Name)
	}

	if metadata.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", metadata.Version)
	}

	if len(metadata.TestTypes) == 0 {
		t.Error("Expected test types to be defined")
	}

	expectedTestTypes := []string{"bulk_copy", "copy_performance", "load_testing"}
	for i, expected := range expectedTestTypes {
		if i >= len(metadata.TestTypes) || metadata.TestTypes[i] != expected {
			t.Errorf("Expected test type '%s' at index %d", expected, i)
		}
	}
}

func TestBulkCopyPlugin_Initialize(t *testing.T) {
	plugin := &BulkCopyPlugin{}
	coreServices := &core.CoreServices{
		Logger:  &MockLogger{},
		Storage: &MockStorage{},
	}

	ctx := context.Background()
	err := plugin.Initialize(ctx, coreServices)

	if err != nil {
		t.Errorf("Initialize should not return error, got: %v", err)
	}

	if plugin.core == nil {
		t.Error("Core services should be set after initialization")
	}

	if plugin.logger == nil {
		t.Error("Logger should be set after initialization")
	}

	if plugin.stopChan == nil {
		t.Error("Stop channel should be initialized")
	}

	if plugin.metrics == nil {
		t.Error("Metrics should be initialized")
	}
}

func TestBulkCopyPlugin_Validate(t *testing.T) {
	plugin := &BulkCopyPlugin{}

	tests := []struct {
		name        string
		config      map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid basic config",
			config: map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"database": "test",
				"username": "user",
				"password": "pass",
			},
			expectError: false,
		},
		{
			name: "missing host",
			config: map[string]interface{}{
				"port":     5432,
				"database": "test",
				"username": "user",
				"password": "pass",
			},
			expectError: true,
			errorMsg:    "host is required",
		},
		{
			name: "invalid port",
			config: map[string]interface{}{
				"host":     "localhost",
				"port":     -1,
				"database": "test",
				"username": "user",
				"password": "pass",
			},
			expectError: true,
			errorMsg:    "port must be between 1 and 65535",
		},
		{
			name: "missing database",
			config: map[string]interface{}{
				"host":     "localhost",
				"port":     5432,
				"username": "user",
				"password": "pass",
			},
			expectError: true,
			errorMsg:    "database is required",
		},
		{
			name: "invalid connections",
			config: map[string]interface{}{
				"host":        "localhost",
				"port":        5432,
				"database":    "test",
				"username":    "user",
				"password":    "pass",
				"connections": 0,
			},
			expectError: true,
			errorMsg:    "connections must be between 1 and 1000",
		},
		{
			name: "invalid copy format",
			config: map[string]interface{}{
				"host":        "localhost",
				"port":        5432,
				"database":    "test",
				"username":    "user",
				"password":    "pass",
				"copy_format": "INVALID",
			},
			expectError: true,
			errorMsg:    "copy_format must be one of: CSV, TEXT, BINARY",
		},
		{
			name: "valid config with all options",
			config: map[string]interface{}{
				"host":           "localhost",
				"port":           5432,
				"database":       "test",
				"username":       "user",
				"password":       "pass",
				"ssl_mode":       "disable",
				"batch_sizes":    []interface{}{1000, 5000, 10000},
				"connections":    10,
				"duration":       "5m",
				"warmup_time":    "30s",
				"think_time":     "10ms",
				"table_name":     "test_table",
				"drop_table":     true,
				"generate_data":  true,
				"data_columns":   15,
				"index_columns":  []interface{}{"col1", "col2"},
				"verbose":        true,
				"rebuild":        false,
				"copy_format":    "CSV",
				"copy_header":    false,
				"copy_delimiter": ",",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugin.Validate(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got no error", tt.errorMsg)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("Expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestBulkCopyPlugin_DefaultValues(t *testing.T) {
	plugin := &BulkCopyPlugin{}

	config := map[string]interface{}{
		"host":     "localhost",
		"port":     5432,
		"database": "test",
		"username": "user",
		"password": "pass",
	}

	err := plugin.Validate(config)
	if err != nil {
		t.Errorf("Validation should succeed with minimal config, got: %v", err)
	}

	// Check default values
	if len(plugin.config.BatchSizes) == 0 {
		t.Error("Default batch sizes should be set")
	}

	expectedDefaults := []int{1000, 10000, 50000, 100000}
	for i, expected := range expectedDefaults {
		if i >= len(plugin.config.BatchSizes) || plugin.config.BatchSizes[i] != expected {
			t.Errorf("Expected default batch size %d at index %d, got %v", expected, i, plugin.config.BatchSizes)
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

	if plugin.config.TableName != "bulk_copy_test_data" {
		t.Errorf("Expected default table name 'bulk_copy_test_data', got '%s'", plugin.config.TableName)
	}

	if plugin.config.DataColumns != 10 {
		t.Errorf("Expected default data columns 10, got %d", plugin.config.DataColumns)
	}

	if plugin.config.SSLMode != "disable" {
		t.Errorf("Expected default SSL mode 'disable', got '%s'", plugin.config.SSLMode)
	}

	if plugin.config.CopyFormat != "CSV" {
		t.Errorf("Expected default copy format 'CSV', got '%s'", plugin.config.CopyFormat)
	}

	if plugin.config.CopyDelimiter != "," {
		t.Errorf("Expected default copy delimiter ',', got '%s'", plugin.config.CopyDelimiter)
	}
}

func TestBulkCopyPlugin_NewPlugin(t *testing.T) {
	plugin := NewPlugin()

	if plugin == nil {
		t.Error("NewPlugin should return a plugin instance")
	}

	bulkCopyPlugin, ok := plugin.(*BulkCopyPlugin)
	if !ok {
		t.Error("NewPlugin should return a BulkCopyPlugin instance")
	}

	if bulkCopyPlugin == nil {
		t.Error("Plugin should be properly initialized")
	}
}

func TestBulkCopyPlugin_Cleanup(t *testing.T) {
	plugin := &BulkCopyPlugin{
		stopChan: make(chan struct{}),
	}

	ctx := context.Background()
	err := plugin.Cleanup(ctx)

	if err != nil {
		t.Errorf("Cleanup should not return error, got: %v", err)
	}

	// Verify that stopChan is closed by trying to receive from it
	select {
	case _, ok := <-plugin.stopChan:
		if ok {
			t.Error("stopChan should be closed after cleanup")
		}
	default:
		t.Error("stopChan should be closed and readable")
	}
}

// Benchmark tests for performance validation
func BenchmarkBulkCopyPlugin_Validate(b *testing.B) {
	plugin := &BulkCopyPlugin{}
	config := map[string]interface{}{
		"host":     "localhost",
		"port":     5432,
		"database": "test",
		"username": "user",
		"password": "pass",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = plugin.Validate(config)
	}
}

func BenchmarkBulkCopyPlugin_Metadata(b *testing.B) {
	plugin := &BulkCopyPlugin{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = plugin.Metadata()
	}
}
