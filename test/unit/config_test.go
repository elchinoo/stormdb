package unit_test

import (
	"path/filepath"
	"testing"

	"github.com/elchinoo/stormdb/internal/config"
)

func TestLoadValidConfig(t *testing.T) {
	configPath := filepath.Join("..", "..", "test", "fixtures", "valid_config.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Expected valid config to load successfully, got error: %v", err)
	}

	// Validate expected values
	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", cfg.Database.Port)
	}
	if cfg.Workload != "simple" {
		t.Errorf("Expected workload 'simple', got '%s'", cfg.Workload)
	}
	if cfg.Workers != 4 {
		t.Errorf("Expected workers 4, got %d", cfg.Workers)
	}
	if cfg.Connections != 8 {
		t.Errorf("Expected connections 8, got %d", cfg.Connections)
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	configPath := filepath.Join("..", "..", "test", "fixtures", "invalid_duration.yaml")

	_, err := config.Load(configPath)
	if err == nil {
		t.Fatal("Expected error for invalid duration, but got none")
	}

	expectedError := "invalid duration format"
	if !containsError(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestLoadInvalidWorkers(t *testing.T) {
	configPath := filepath.Join("..", "..", "test", "fixtures", "invalid_workers.yaml")

	_, err := config.Load(configPath)
	if err == nil {
		t.Fatal("Expected error for invalid workers, but got none")
	}

	expectedError := "workers must be positive"
	if !containsError(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestLoadInvalidConnections(t *testing.T) {
	configPath := filepath.Join("..", "..", "test", "fixtures", "invalid_connections.yaml")

	_, err := config.Load(configPath)
	if err == nil {
		t.Fatal("Expected error for invalid connections, but got none")
	}

	expectedError := "connections (5) should be >= workers (10)"
	if !containsError(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestLoadInvalidWorkload(t *testing.T) {
	configPath := filepath.Join("..", "..", "test", "fixtures", "invalid_workload.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Unexpected error loading config: %v", err)
	}

	// The config should load successfully since workload validation
	// is now deferred to the workload factory (to support plugins)
	if cfg.Workload != "invalid_workload" {
		t.Errorf("Expected workload to be 'invalid_workload', got: %s", cfg.Workload)
	}

	// Note: Actual workload validation happens when creating the workload
	// through the factory, not during config loading
}

func TestLoadInvalidHost(t *testing.T) {
	configPath := filepath.Join("..", "..", "test", "fixtures", "invalid_host.yaml")

	_, err := config.Load(configPath)
	if err == nil {
		t.Fatal("Expected error for invalid host, but got none")
	}

	expectedError := "database host is required"
	if !containsError(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	_, err := config.Load("non_existent_file.yaml")
	if err == nil {
		t.Fatal("Expected error for non-existent config file, but got none")
	}
}

// Helper function to check if error message contains expected substring
func containsError(actual, expected string) bool {
	return len(actual) >= len(expected) &&
		actual[:len(expected)] == expected ||
		actual[len(actual)-len(expected):] == expected ||
		findSubstring(actual, expected)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
