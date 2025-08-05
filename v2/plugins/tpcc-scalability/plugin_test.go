package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/elchinoo/stormdb/v2/core"
	_ "github.com/lib/pq"
)

// TestTPCCPlugin_Metadata tests the plugin metadata
func TestTPCCPlugin_Metadata(t *testing.T) {
	plugin := &TPCCPlugin{}
	metadata := plugin.Metadata()

	if metadata.Name != "tpcc-scalability" {
		t.Errorf("Expected name 'tpcc-scalability', got %s", metadata.Name)
	}

	if metadata.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", metadata.Version)
	}

	if len(metadata.TestTypes) == 0 {
		t.Error("Expected test types to be defined")
	}

	if metadata.ConfigSchema == "" {
		t.Error("Expected config schema to be defined")
	}
}

// TestTPCCPlugin_Validate tests configuration validation
func TestTPCCPlugin_Validate(t *testing.T) {
	plugin := &TPCCPlugin{}

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"host":             "localhost",
				"port":             5432,
				"database":         "test",
				"username":         "postgres",
				"password":         "postgres",
				"scale":            10,
				"connections":      []int{48, 96},
				"duration":         "5m",
				"new_order_pct":    45,
				"payment_pct":      43,
				"order_status_pct": 4,
				"delivery_pct":     4,
				"stock_level_pct":  4,
			},
			wantErr: false,
		},
		{
			name: "invalid connection count",
			config: map[string]interface{}{
				"host":             "localhost",
				"port":             5432,
				"database":         "test",
				"username":         "postgres",
				"password":         "postgres",
				"scale":            10,
				"connections":      []int{-1},
				"duration":         "5m",
				"new_order_pct":    45,
				"payment_pct":      43,
				"order_status_pct": 4,
				"delivery_pct":     4,
				"stock_level_pct":  4,
			},
			wantErr: true,
		},
		{
			name: "invalid transaction percentages",
			config: map[string]interface{}{
				"host":             "localhost",
				"port":             5432,
				"database":         "test",
				"username":         "postgres",
				"password":         "postgres",
				"scale":            10,
				"connections":      []int{48},
				"duration":         "5m",
				"new_order_pct":    50,
				"payment_pct":      50,
				"order_status_pct": 0,
				"delivery_pct":     0,
				"stock_level_pct":  0,
			},
			wantErr: true,
		},
		{
			name: "empty connections",
			config: map[string]interface{}{
				"host":        "localhost",
				"port":        5432,
				"database":    "test",
				"username":    "postgres",
				"password":    "postgres",
				"scale":       10,
				"connections": []int{},
				"duration":    "5m",
			},
			wantErr: true,
		},
		{
			name: "invalid scale",
			config: map[string]interface{}{
				"host":        "localhost",
				"port":        5432,
				"database":    "test",
				"username":    "postgres",
				"password":    "postgres",
				"scale":       0,
				"connections": []int{48},
				"duration":    "5m",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := plugin.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestParseConfig tests configuration parsing
func TestParseConfig(t *testing.T) {
	config := map[string]interface{}{
		"host":        "testhost",
		"port":        5433,
		"database":    "testdb",
		"scale":       20,
		"connections": []interface{}{48, 96, 192},
		"duration":    "10m",
	}

	cfg, err := parseConfig(config)
	if err != nil {
		t.Fatalf("parseConfig() error = %v", err)
	}

	if cfg.Host != "testhost" {
		t.Errorf("Expected host 'testhost', got %s", cfg.Host)
	}

	if cfg.Port != 5433 {
		t.Errorf("Expected port 5433, got %d", cfg.Port)
	}

	if cfg.Scale != 20 {
		t.Errorf("Expected scale 20, got %d", cfg.Scale)
	}

	expectedConnections := []int{48, 96, 192}
	if len(cfg.Connections) != len(expectedConnections) {
		t.Errorf("Expected %d connections, got %d", len(expectedConnections), len(cfg.Connections))
	}
}

// TestTPCCPlugin_Initialize tests plugin initialization
func TestTPCCPlugin_Initialize(t *testing.T) {
	plugin := &TPCCPlugin{}

	// Mock core services
	mockLogger := &MockLogger{}
	coreServices := &core.CoreServices{
		Logger: mockLogger,
	}

	ctx := context.Background()
	err := plugin.Initialize(ctx, coreServices)
	if err != nil {
		t.Errorf("Initialize() error = %v", err)
	}

	if plugin.core == nil {
		t.Error("Expected core services to be set")
	}

	if plugin.logger == nil {
		t.Error("Expected logger to be set")
	}

	if plugin.stopChan == nil {
		t.Error("Expected stop channel to be initialized")
	}

	if plugin.metrics == nil {
		t.Error("Expected metrics to be initialized")
	}
}

// TestSelectTransactionType tests transaction type selection
func TestSelectTransactionType(t *testing.T) {
	plugin := &TPCCPlugin{
		config: &defaultConfig,
	}

	// Test multiple selections to verify distribution
	counts := make(map[string]int)
	iterations := 10000

	for i := 0; i < iterations; i++ {
		txType := plugin.selectTransactionType()
		counts[txType]++
	}

	// Verify all transaction types are selected
	expectedTypes := []string{"new_order", "payment", "order_status", "delivery", "stock_level"}
	for _, txType := range expectedTypes {
		if counts[txType] == 0 {
			t.Errorf("Transaction type %s was never selected", txType)
		}
	}

	// Verify distribution is roughly correct (within 10% tolerance)
	tolerance := 0.1
	expectedPercent := map[string]float64{
		"new_order":    0.45,
		"payment":      0.43,
		"order_status": 0.04,
		"delivery":     0.04,
		"stock_level":  0.04,
	}

	for txType, expected := range expectedPercent {
		actual := float64(counts[txType]) / float64(iterations)
		diff := abs(actual - expected)
		if diff > tolerance {
			t.Errorf("Transaction type %s: expected ~%.2f, got %.2f (diff: %.2f)",
				txType, expected, actual, diff)
		}
	}
}

// TestMetricsOperations tests metrics operations
func TestMetricsOperations(t *testing.T) {
	plugin := &TPCCPlugin{
		metrics: &TPCCMetrics{
			MinLatency: time.Hour,
		},
	}

	// Test updateMetrics
	plugin.updateMetrics("new_order", 100*time.Millisecond, nil)
	plugin.updateMetrics("payment", 50*time.Millisecond, nil)

	if plugin.metrics.NewOrderCount != 1 {
		t.Errorf("Expected NewOrderCount 1, got %d", plugin.metrics.NewOrderCount)
	}

	if plugin.metrics.PaymentCount != 1 {
		t.Errorf("Expected PaymentCount 1, got %d", plugin.metrics.PaymentCount)
	}

	if plugin.metrics.MinLatency != 50*time.Millisecond {
		t.Errorf("Expected MinLatency 50ms, got %v", plugin.metrics.MinLatency)
	}

	if plugin.metrics.MaxLatency != 100*time.Millisecond {
		t.Errorf("Expected MaxLatency 100ms, got %v", plugin.metrics.MaxLatency)
	}

	// Test resetMetrics
	plugin.resetMetrics()
	if plugin.metrics.NewOrderCount != 0 {
		t.Errorf("Expected NewOrderCount 0 after reset, got %d", plugin.metrics.NewOrderCount)
	}

	if plugin.metrics.MinLatency != time.Hour {
		t.Errorf("Expected MinLatency reset to 1h, got %v", plugin.metrics.MinLatency)
	}
}

// TestGetMetricID tests metric ID mapping
func TestGetMetricID(t *testing.T) {
	plugin := &TPCCPlugin{}

	tests := []struct {
		txType     string
		expectedID int
	}{
		{"new_order", 1},
		{"payment", 2},
		{"order_status", 3},
		{"delivery", 4},
		{"stock_level", 5},
		{"unknown", 1}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.txType, func(t *testing.T) {
			id := plugin.getMetricID(tt.txType)
			if id != tt.expectedID {
				t.Errorf("getMetricID(%s) = %d, expected %d", tt.txType, id, tt.expectedID)
			}
		})
	}
}

// TestConfigSchema tests that the config schema is valid JSON
func TestConfigSchema(t *testing.T) {
	schema := getConfigSchema()
	if schema == "" {
		t.Error("Config schema should not be empty")
	}

	// Should be valid JSON
	var schemaMap map[string]interface{}
	if err := json.Unmarshal([]byte(schema), &schemaMap); err != nil {
		t.Errorf("Config schema is not valid JSON: %v", err)
	}

	// Should have required properties
	props, ok := schemaMap["properties"].(map[string]interface{})
	if !ok {
		t.Error("Config schema should have properties")
	}

	requiredProps := []string{"host", "port", "database", "username", "password"}
	for _, prop := range requiredProps {
		if _, exists := props[prop]; !exists {
			t.Errorf("Config schema missing required property: %s", prop)
		}
	}
}

// Benchmark tests
func BenchmarkSelectTransactionType(b *testing.B) {
	plugin := &TPCCPlugin{
		config: &defaultConfig,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin.selectTransactionType()
	}
}

func BenchmarkUpdateMetrics(b *testing.B) {
	plugin := &TPCCPlugin{
		metrics: &TPCCMetrics{
			MinLatency: time.Hour,
		},
	}

	latency := 100 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		plugin.updateMetrics("new_order", latency, nil)
	}
}

// Mock implementations for testing

type MockLogger struct {
	entries []LogEntry
}

type LogEntry struct {
	Level  string
	Msg    string
	Fields []core.Field
}

func (m *MockLogger) Debug(msg string, fields ...core.Field) {
	m.entries = append(m.entries, LogEntry{"debug", msg, fields})
}

func (m *MockLogger) Info(msg string, fields ...core.Field) {
	m.entries = append(m.entries, LogEntry{"info", msg, fields})
}

func (m *MockLogger) Warn(msg string, fields ...core.Field) {
	m.entries = append(m.entries, LogEntry{"warn", msg, fields})
}

func (m *MockLogger) Error(msg string, fields ...core.Field) {
	m.entries = append(m.entries, LogEntry{"error", msg, fields})
}

func (m *MockLogger) WithFields(fields ...core.Field) core.Logger {
	return m
}

func (m *MockLogger) WithPlugin(pluginName string) core.Logger {
	return m
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
