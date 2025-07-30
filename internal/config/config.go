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

	// Validate progressive scaling configuration if enabled
	if cfg.Progressive.Enabled {
		if err := validateProgressiveConfig(&cfg.Progressive); err != nil {
			return fmt.Errorf("progressive scaling configuration error: %w", err)
		}
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
		// File existence will be validated when actually used during setup
		// to avoid requiring the file to exist during config validation
	}

	return nil
}

// validateProgressiveConfig validates progressive scaling configuration
func validateProgressiveConfig(p *struct {
	Enabled          bool   `mapstructure:"enabled"`
	Strategy         string `mapstructure:"strategy"`
	MinWorkers       int    `mapstructure:"min_workers"`
	MaxWorkers       int    `mapstructure:"max_workers"`
	MinConns         int    `mapstructure:"min_connections"`
	MaxConns         int    `mapstructure:"max_connections"`
	TestDuration     string `mapstructure:"test_duration"`
	WarmupDuration   string `mapstructure:"warmup_duration"`
	CooldownDuration string `mapstructure:"cooldown_duration"`
	Bands            int    `mapstructure:"bands"`
	ExportCSV        bool   `mapstructure:"export_csv"`
	ExportJSON       bool   `mapstructure:"export_json"`
	EnableAnalysis   bool   `mapstructure:"enable_analysis"`
	// Legacy fields for backward compatibility
	StepWorkers  int    `mapstructure:"step_workers"`
	StepConns    int    `mapstructure:"step_connections"`
	BandDuration string `mapstructure:"band_duration"`
	WarmupTime   string `mapstructure:"warmup_time"`
	CooldownTime string `mapstructure:"cooldown_time"`
	ExportFormat string `mapstructure:"export_format"`
	ExportPath   string `mapstructure:"export_path"`
}) error {
	if p.MinWorkers <= 0 {
		return fmt.Errorf("min_workers must be positive, got: %d", p.MinWorkers)
	}
	if p.MaxWorkers <= 0 {
		return fmt.Errorf("max_workers must be positive, got: %d", p.MaxWorkers)
	}
	if p.MinWorkers > p.MaxWorkers {
		return fmt.Errorf("min_workers (%d) must be <= max_workers (%d)", p.MinWorkers, p.MaxWorkers)
	}

	if p.MinConns <= 0 {
		return fmt.Errorf("min_connections must be positive, got: %d", p.MinConns)
	}
	if p.MaxConns <= 0 {
		return fmt.Errorf("max_connections must be positive, got: %d", p.MaxConns)
	}
	if p.MinConns > p.MaxConns {
		return fmt.Errorf("min_connections (%d) must be <= max_connections (%d)", p.MinConns, p.MaxConns)
	}

	// Check format - prefer v0.2 format
	usingV2Format := p.Bands > 0 || p.TestDuration != ""

	if !usingV2Format {
		// Legacy format validation
		if p.StepWorkers <= 0 {
			return fmt.Errorf("step_workers must be positive, got: %d", p.StepWorkers)
		}
		if p.StepConns <= 0 {
			return fmt.Errorf("step_connections must be positive, got: %d", p.StepConns)
		}
		if p.BandDuration == "" {
			return fmt.Errorf("band_duration is required")
		}
	}

	// Basic validation only - detailed validation happens in progressive engine
	return nil
}
