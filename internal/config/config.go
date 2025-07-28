// internal/config/config.go
package config

import (
	"fmt"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"

	"github.com/spf13/viper"
)

func Load(configFile string) (*types.Config, error) {
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg types.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

func validateConfig(cfg *types.Config) error {
	// Validate duration
	if _, err := time.ParseDuration(cfg.Duration); err != nil {
		return fmt.Errorf("invalid duration format: %s", cfg.Duration)
	}

	// Validate workers
	if cfg.Workers <= 0 {
		return fmt.Errorf("workers must be positive, got: %d", cfg.Workers)
	}
	if cfg.Workers > 1000 {
		return fmt.Errorf("workers too high (max 1000), got: %d", cfg.Workers)
	}

	// Validate connections
	if cfg.Connections <= 0 {
		return fmt.Errorf("connections must be positive, got: %d", cfg.Connections)
	}
	if cfg.Connections > 10000 {
		return fmt.Errorf("connections too high (max 10000), got: %d", cfg.Connections)
	}

	// Validate connections >= workers (good practice)
	if cfg.Connections < cfg.Workers {
		return fmt.Errorf("connections (%d) should be >= workers (%d) for optimal performance", cfg.Connections, cfg.Workers)
	}

	// Validate scale
	if cfg.Scale < 0 {
		return fmt.Errorf("scale must be non-negative, got: %d", cfg.Scale)
	}

	// Note: Workload validation is deferred to the workload factory
	// to support plugins that may provide additional workload types

	// Validate database configuration
	if cfg.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if cfg.Database.Port <= 0 || cfg.Database.Port > 65535 {
		return fmt.Errorf("database port must be between 1-65535, got: %d", cfg.Database.Port)
	}
	if cfg.Database.Dbname == "" {
		return fmt.Errorf("database name is required")
	}
	if cfg.Database.Username == "" {
		return fmt.Errorf("database username is required")
	}

	// Validate SSL mode
	validSSLModes := map[string]bool{
		"disable": true, "require": true, "verify-ca": true, "verify-full": true,
	}
	if cfg.Database.Sslmode != "" && !validSSLModes[cfg.Database.Sslmode] {
		return fmt.Errorf("invalid sslmode: %s (valid: disable, require, verify-ca, verify-full)", cfg.Database.Sslmode)
	}

	// Validate data loading configuration (if specified)
	if cfg.DataLoading.Mode != "" {
		validModes := map[string]bool{
			"generate": true,
			"dump":     true,
			"sql":      true,
		}
		if !validModes[cfg.DataLoading.Mode] {
			return fmt.Errorf("invalid data loading mode: %s (valid: generate, dump, sql)", cfg.DataLoading.Mode)
		}

		// Validate file path is provided for dump and sql modes
		if (cfg.DataLoading.Mode == "dump" || cfg.DataLoading.Mode == "sql") && cfg.DataLoading.FilePath == "" {
			return fmt.Errorf("filepath is required when data loading mode is '%s'", cfg.DataLoading.Mode)
		}

		// Validate file exists for dump and sql modes
		if cfg.DataLoading.Mode == "dump" || cfg.DataLoading.Mode == "sql" {
			// File existence will be validated when actually used during setup
			// to avoid requiring the file to exist during config validation
		}
	}

	return nil
}
