// Package config provides configuration management for StormDB v2
// Handles global and plugin-specific configuration with validation
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/elchinoo/stormdb/v2/core"
	"gopkg.in/yaml.v3"
)

// Manager implements the ConfigManager interface
type Manager struct {
	globalConfig  *core.CoreConfig
	pluginConfigs map[string]map[string]interface{}
	configPath    string
	mu            sync.RWMutex
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		pluginConfigs: make(map[string]map[string]interface{}),
	}
}

// LoadFromFile loads configuration from a YAML file
func (m *Manager) LoadFromFile(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.configPath = path

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", path)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config core.CoreConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate required fields
	if err := m.validateGlobalConfig(&config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	m.globalConfig = &config
	return nil
}

// LoadFromEnv loads configuration from environment variables
func (m *Manager) LoadFromEnv() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config := &core.CoreConfig{
		Database: core.DatabaseConfig{
			Host:            getEnvOrDefault("STORMDB_DB_HOST", "localhost"),
			Port:            getEnvIntOrDefault("STORMDB_DB_PORT", 5432),
			Database:        getEnvOrDefault("STORMDB_DB_NAME", "stormdb"),
			Username:        getEnvOrDefault("STORMDB_DB_USER", "postgres"),
			Password:        getEnvOrDefault("STORMDB_DB_PASSWORD", ""),
			SSLMode:         getEnvOrDefault("STORMDB_DB_SSL_MODE", "disable"),
			MaxConnections:  getEnvIntOrDefault("STORMDB_DB_MAX_CONNECTIONS", 10),
			MaxIdleConns:    getEnvIntOrDefault("STORMDB_DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvOrDefault("STORMDB_DB_CONN_MAX_LIFETIME", "1h"),
		},
		API: core.APIConfig{
			Host: getEnvOrDefault("STORMDB_API_HOST", "0.0.0.0"),
			Port: getEnvIntOrDefault("STORMDB_API_PORT", 8080),
			GRPC: struct {
				Enabled bool `yaml:"enabled" json:"enabled"`
				Port    int  `yaml:"port" json:"port"`
			}{
				Enabled: getEnvBoolOrDefault("STORMDB_GRPC_ENABLED", true),
				Port:    getEnvIntOrDefault("STORMDB_GRPC_PORT", 9090),
			},
		},
		Logging: core.LoggingConfig{
			Level:  getEnvOrDefault("STORMDB_LOG_LEVEL", "info"),
			Format: getEnvOrDefault("STORMDB_LOG_FORMAT", "json"),
			Output: getEnvOrDefault("STORMDB_LOG_OUTPUT", "stdout"),
			File:   getEnvOrDefault("STORMDB_LOG_FILE", ""),
		},
		Scheduler: core.SchedulerConfig{
			Enabled:        getEnvBoolOrDefault("STORMDB_SCHEDULER_ENABLED", true),
			WorkerPoolSize: getEnvIntOrDefault("STORMDB_SCHEDULER_WORKERS", 4),
		},
		PluginDirs: strings.Split(getEnvOrDefault("STORMDB_PLUGIN_DIRS", "./plugins"), ","),
	}

	if err := m.validateGlobalConfig(config); err != nil {
		return fmt.Errorf("invalid environment configuration: %w", err)
	}

	m.globalConfig = config
	return nil
}

// GetGlobal returns the global configuration
func (m *Manager) GetGlobal() *core.CoreConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.globalConfig
}

// GetPlugin returns plugin-specific configuration
func (m *Manager) GetPlugin(pluginName string) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if config, exists := m.pluginConfigs[pluginName]; exists {
		return config, nil
	}

	// Try to load from file if not in memory
	if m.configPath != "" {
		pluginConfigPath := m.getPluginConfigPath(pluginName)
		if _, err := os.Stat(pluginConfigPath); err == nil {
			config, err := m.loadPluginConfigFromFile(pluginConfigPath)
			if err != nil {
				return nil, err
			}
			m.pluginConfigs[pluginName] = config
			return config, nil
		}
	}

	return map[string]interface{}{}, nil
}

// SetPlugin sets plugin-specific configuration
func (m *Manager) SetPlugin(pluginName string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pluginConfigs[pluginName] = config
	return nil
}

// Reload reloads configuration from the original source
func (m *Manager) Reload() error {
	if m.configPath != "" {
		return m.LoadFromFile(m.configPath)
	}
	return m.LoadFromEnv()
}

// Validate validates configuration against a JSON schema
func (m *Manager) Validate(config map[string]interface{}, schema string) error {
	// For now, we'll do basic validation
	// In a full implementation, you'd use a JSON Schema validator
	if schema == "" {
		return nil // No schema means no validation
	}

	// Basic validation for required fields (simplified)
	if _, exists := config["name"]; !exists {
		return fmt.Errorf("missing required field: name")
	}

	return nil
}

// validateGlobalConfig validates the global configuration
func (m *Manager) validateGlobalConfig(config *core.CoreConfig) error {
	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if config.Database.Port <= 0 || config.Database.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}
	if config.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if config.Database.Username == "" {
		return fmt.Errorf("database username is required")
	}

	// Validate logging level
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[config.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	// Validate logging format
	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[config.Logging.Format] {
		return fmt.Errorf("invalid log format: %s", config.Logging.Format)
	}

	return nil
}

// getPluginConfigPath returns the expected path for a plugin's configuration file
func (m *Manager) getPluginConfigPath(pluginName string) string {
	if m.configPath == "" {
		return fmt.Sprintf("./config/%s.yaml", pluginName)
	}

	dir := filepath.Dir(m.configPath)
	return filepath.Join(dir, fmt.Sprintf("%s.yaml", pluginName))
}

// loadPluginConfigFromFile loads plugin configuration from a YAML file
func (m *Manager) loadPluginConfigFromFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin config file: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse plugin config file: %w", err)
	}

	return config, nil
}

// SavePluginConfig saves plugin configuration to file
func (m *Manager) SavePluginConfig(pluginName string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update in-memory config
	m.pluginConfigs[pluginName] = config

	// Save to file if we have a config path
	if m.configPath != "" {
		configPath := m.getPluginConfigPath(pluginName)

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Marshal to YAML
		data, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal plugin config: %w", err)
		}

		// Write to file
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write plugin config file: %w", err)
		}
	}

	return nil
}

// GetConnectionString returns a formatted database connection string
func (m *Manager) GetConnectionString() string {
	config := m.GetGlobal()
	if config == nil {
		return ""
	}

	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Database.Host,
		config.Database.Port,
		config.Database.Username,
		config.Database.Password,
		config.Database.Database,
		config.Database.SSLMode,
	)
}

// ToJSON converts configuration to JSON for API responses
func (m *Manager) ToJSON() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.globalConfig == nil {
		return nil, fmt.Errorf("no configuration loaded")
	}

	// Create a sanitized version without sensitive data
	sanitized := *m.globalConfig
	sanitized.Database.Password = "[REDACTED]"

	return json.MarshalIndent(sanitized, "", "  ")
}

// Helper functions for environment variable parsing

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := parseIntSafe(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true"
	}
	return defaultValue
}

func parseIntSafe(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
