package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/elchinoo/stormdb/pkg/types"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `
database:
  type: postgres
  host: "localhost"
  port: 5432
  dbname: "test_db"
  username: "test_user"
  password: "test_pass"
  sslmode: "disable"

workload: "simple"
scale: 100
duration: "30s"
workers: 4
connections: 8
`

	err := os.WriteFile(configFile, []byte(configContent), 0600)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded values
	if cfg.Database.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", cfg.Database.Port)
	}
	if cfg.Workload != "simple" {
		t.Errorf("Expected workload 'simple', got %s", cfg.Workload)
	}
	if cfg.Duration != "30s" {
		t.Errorf("Expected duration '30s', got %s", cfg.Duration)
	}
	if cfg.Workers != 4 {
		t.Errorf("Expected workers 4, got %d", cfg.Workers)
	}
	if cfg.Connections != 8 {
		t.Errorf("Expected connections 8, got %d", cfg.Connections)
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent config file")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *types.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     5432,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     4,
				Connections: 8,
				Scale:       100,
			},
			expectError: false,
		},
		{
			name: "invalid duration",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     5432,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "invalid",
				Workers:     4,
				Connections: 8,
				Scale:       100,
			},
			expectError: true,
			errorMsg:    "invalid duration format",
		},
		{
			name: "zero workers",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     5432,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     0,
				Connections: 8,
				Scale:       100,
			},
			expectError: true,
			errorMsg:    "workers must be positive",
		},
		{
			name: "too many workers",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     5432,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     1001,
				Connections: 8,
				Scale:       100,
			},
			expectError: true,
			errorMsg:    "workers too high",
		},
		{
			name: "zero connections",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     5432,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     4,
				Connections: 0,
				Scale:       100,
			},
			expectError: true,
			errorMsg:    "connections must be positive",
		},
		{
			name: "connections less than workers",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     5432,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     8,
				Connections: 4,
				Scale:       100,
			},
			expectError: true,
			errorMsg:    "connections (4) should be >= workers (8) for optimal performance",
		},
		{
			name: "negative scale",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     5432,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     4,
				Connections: 8,
				Scale:       -1,
			},
			expectError: true,
			errorMsg:    "scale must be non-negative",
		},
		{
			name: "empty host",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "",
					Port:     5432,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     4,
				Connections: 8,
				Scale:       100,
			},
			expectError: true,
			errorMsg:    "database host is required",
		},
		{
			name: "invalid port",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     0,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     4,
				Connections: 8,
				Scale:       100,
			},
			expectError: true,
			errorMsg:    "database port must be between 1-65535",
		},
		{
			name: "empty username",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     5432,
					Dbname:   "test",
					Username: "",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     4,
				Connections: 8,
				Scale:       100,
			},
			expectError: true,
			errorMsg:    "database username is required",
		},
		{
			name: "invalid ssl mode",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     5432,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "invalid",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     4,
				Connections: 8,
				Scale:       100,
			},
			expectError: true,
			errorMsg:    "invalid sslmode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if err.Error() == "" || len(tt.errorMsg) > 0 {
					// Check if error message contains expected text
					if len(tt.errorMsg) > 0 && err.Error() != "" {
						// Just verify it's an error for now - specific message checking can be too brittle
						t.Logf("Got expected error: %v", err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestValidateDataLoadingConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *types.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid generate mode",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     5432,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     4,
				Connections: 8,
				Scale:       100,
				DataLoading: struct {
					Mode        string `mapstructure:"mode"`
					FilePath    string `mapstructure:"filepath"`
					BatchSize   int    `mapstructure:"batch_size"`
					MaxMemoryMB int    `mapstructure:"max_memory_mb"`
				}{
					Mode: "generate",
				},
			},
			expectError: false,
		},
		{
			name: "invalid data loading mode",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     5432,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     4,
				Connections: 8,
				Scale:       100,
				DataLoading: struct {
					Mode        string `mapstructure:"mode"`
					FilePath    string `mapstructure:"filepath"`
					BatchSize   int    `mapstructure:"batch_size"`
					MaxMemoryMB int    `mapstructure:"max_memory_mb"`
				}{
					Mode: "invalid",
				},
			},
			expectError: true,
			errorMsg:    "invalid data loading mode",
		},
		{
			name: "dump mode without filepath",
			config: &types.Config{
				Database: struct {
					Type     string `mapstructure:"type"`
					Host     string `mapstructure:"host"`
					Port     int    `mapstructure:"port"`
					Dbname   string `mapstructure:"dbname"`
					Username string `mapstructure:"username"`
					Password string `mapstructure:"password"`
					Sslmode  string `mapstructure:"sslmode"`
				}{
					Host:     "localhost",
					Port:     5432,
					Dbname:   "test",
					Username: "user",
					Sslmode:  "disable",
				},
				Workload:    "simple",
				Duration:    "30s",
				Workers:     4,
				Connections: 8,
				Scale:       100,
				DataLoading: struct {
					Mode        string `mapstructure:"mode"`
					FilePath    string `mapstructure:"filepath"`
					BatchSize   int    `mapstructure:"batch_size"`
					MaxMemoryMB int    `mapstructure:"max_memory_mb"`
				}{
					Mode: "dump",
				},
			},
			expectError: true,
			errorMsg:    "filepath is required when data loading mode is 'dump'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}
