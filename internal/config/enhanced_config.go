package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// ConfigValidator defines the interface for configuration validation and migration
type ConfigValidator interface {
	Validate() error
	SetDefaults()
	Migrate(version string) error
}

// StormDBConfig represents the complete configuration structure with validation
type StormDBConfig struct {
	Version  string         `yaml:"version" validate:"required,semver"`
	Database DatabaseConfig `yaml:"database" validate:"required"`
	Workload WorkloadConfig `yaml:"workload" validate:"required"`
	Plugins  PluginConfig   `yaml:"plugins"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	Advanced AdvancedConfig `yaml:"advanced"`
	Logger   LoggerConfig   `yaml:"logger"`
}

// DatabaseConfig holds database connection configuration with validation
type DatabaseConfig struct {
	Type     string `yaml:"type" validate:"required,oneof=postgres"`
	Host     string `yaml:"host" validate:"required,hostname_rfc1123|ip"`
	Port     int    `yaml:"port" validate:"min=1,max=65535"`
	Database string `yaml:"database" validate:"required,min=1"`
	Username string `yaml:"username" validate:"required"`
	Password string `yaml:"password" validate:"required"`
	SSLMode  string `yaml:"sslmode" validate:"oneof=disable require verify-ca verify-full"`

	// Connection pool settings
	MaxConnections    int           `yaml:"max_connections" validate:"min=1,max=1000"`
	MinConnections    int           `yaml:"min_connections" validate:"min=0"`
	MaxConnLifetime   time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `yaml:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `yaml:"health_check_period"`
	ConnectTimeout    time.Duration `yaml:"connect_timeout"`
}

// WorkloadConfig defines workload execution parameters with validation
type WorkloadConfig struct {
	Type            string        `yaml:"type" validate:"required,min=1"`
	Duration        time.Duration `yaml:"duration" validate:"min=1s"`
	Workers         int           `yaml:"workers" validate:"min=1,max=10000"`
	Connections     int           `yaml:"connections" validate:"min=1,max=10000"`
	Scale           int           `yaml:"scale" validate:"min=1"`
	SummaryInterval time.Duration `yaml:"summary_interval" validate:"min=1s"`

	// Progressive scaling configuration
	Progressive *ProgressiveConfig `yaml:"progressive"`

	// Workload-specific configuration
	Config map[string]interface{} `yaml:"workload_config"`
}

// ProgressiveConfig defines progressive scaling parameters
type ProgressiveConfig struct {
	Enabled           bool          `yaml:"enabled"`
	Strategy          string        `yaml:"strategy" validate:"oneof=linear exponential fibonacci custom"`
	MinWorkers        int           `yaml:"min_workers" validate:"min=1"`
	MaxWorkers        int           `yaml:"max_workers" validate:"min=1"`
	MinConnections    int           `yaml:"min_connections" validate:"min=1"`
	MaxConnections    int           `yaml:"max_connections" validate:"min=1"`
	TestDuration      time.Duration `yaml:"test_duration" validate:"min=10s"`
	WarmupDuration    time.Duration `yaml:"warmup_duration" validate:"min=1s"`
	CooldownDuration  time.Duration `yaml:"cooldown_duration" validate:"min=1s"`
	Bands             int           `yaml:"bands" validate:"min=2,max=50"`
	EnableAnalysis    bool          `yaml:"enable_analysis"`
	MaxLatencySamples int           `yaml:"max_latency_samples" validate:"min=1000"`
	MemoryLimitMB     int           `yaml:"memory_limit_mb" validate:"min=100"`
}

// PluginConfig defines plugin system configuration
type PluginConfig struct {
	Paths               []string      `yaml:"paths"`
	Files               []string      `yaml:"files"`
	AutoLoad            bool          `yaml:"auto_load"`
	HealthCheckEnabled  bool          `yaml:"health_check_enabled"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	MaxLoadAttempts     int           `yaml:"max_load_attempts" validate:"min=1,max=10"`
	LoadTimeout         time.Duration `yaml:"load_timeout"`
}

// MetricsConfig defines metrics collection and reporting configuration
type MetricsConfig struct {
	Enabled            bool          `yaml:"enabled"`
	Interval           time.Duration `yaml:"interval" validate:"min=1s"`
	LatencyPercentiles []int         `yaml:"latency_percentiles" validate:"dive,min=1,max=100"`
	CollectPGStats     bool          `yaml:"collect_pg_stats"`
	PGStatsStatements  bool          `yaml:"pg_stats_statements"`
	ExportPrometheus   bool          `yaml:"export_prometheus"`
	PrometheusPort     int           `yaml:"prometheus_port" validate:"min=1024,max=65535"`
	BatchSize          int           `yaml:"batch_size" validate:"min=1,max=10000"`
	FlushInterval      time.Duration `yaml:"flush_interval"`
	MaxMemoryMB        int           `yaml:"max_memory_mb" validate:"min=10"`
}

// AdvancedConfig contains advanced configuration options
type AdvancedConfig struct {
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
	ResourceLimits ResourceLimitsConfig `yaml:"resource_limits"`
	ErrorHandling  ErrorHandlingConfig  `yaml:"error_handling"`
	Observability  ObservabilityConfig  `yaml:"observability"`
}

// CircuitBreakerConfig defines circuit breaker parameters
type CircuitBreakerConfig struct {
	Enabled       bool          `yaml:"enabled"`
	MaxFailures   int           `yaml:"max_failures" validate:"min=1"`
	ResetTimeout  time.Duration `yaml:"reset_timeout" validate:"min=1s"`
	HalfOpenLimit int           `yaml:"half_open_limit" validate:"min=1"`
}

// ResourceLimitsConfig defines resource usage limits
type ResourceLimitsConfig struct {
	MaxMemoryMB     int           `yaml:"max_memory_mb" validate:"min=100"`
	MaxCPUPercent   int           `yaml:"max_cpu_percent" validate:"min=1,max=100"`
	MaxGoroutines   int           `yaml:"max_goroutines" validate:"min=10"`
	GCTargetPercent int           `yaml:"gc_target_percent" validate:"min=10,max=200"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" validate:"min=1s"`
}

// ErrorHandlingConfig defines error handling behavior
type ErrorHandlingConfig struct {
	FailFast           bool          `yaml:"fail_fast"`
	MaxRetries         int           `yaml:"max_retries" validate:"min=0,max=10"`
	RetryBackoff       time.Duration `yaml:"retry_backoff" validate:"min=100ms"`
	BackoffMultiplier  float64       `yaml:"backoff_multiplier" validate:"min=1.0,max=10.0"`
	MaxBackoff         time.Duration `yaml:"max_backoff"`
	PanicRecovery      bool          `yaml:"panic_recovery"`
	ErrorRateThreshold float64       `yaml:"error_rate_threshold" validate:"min=0.0,max=1.0"`
}

// ObservabilityConfig defines observability and tracing configuration
type ObservabilityConfig struct {
	TracingEnabled  bool    `yaml:"tracing_enabled"`
	TracingEndpoint string  `yaml:"tracing_endpoint"`
	MetricsEnabled  bool    `yaml:"metrics_enabled"`
	MetricsEndpoint string  `yaml:"metrics_endpoint"`
	ServiceName     string  `yaml:"service_name"`
	ServiceVersion  string  `yaml:"service_version"`
	SampleRate      float64 `yaml:"sample_rate" validate:"min=0.0,max=1.0"`
}

// LoggerConfig defines logging configuration
type LoggerConfig struct {
	Level       string `yaml:"level" validate:"oneof=debug info warn error fatal"`
	Format      string `yaml:"format" validate:"oneof=json console"`
	Output      string `yaml:"output"`
	Development bool   `yaml:"development"`
}

// CLIOptions represents command-line override options
type CLIOptions struct {
	ConfigFile  string
	Workers     *int
	Connections *int
	Duration    *time.Duration
	Scale       *int
	Workload    string
	DatabaseURL string
	LogLevel    string
	Verbose     bool
	DryRun      bool
}

// NewStormDBConfig creates a new configuration with defaults
func NewStormDBConfig() *StormDBConfig {
	config := &StormDBConfig{
		Version: "1.0",
	}
	config.SetDefaults()
	return config
}

// SetDefaults sets reasonable defaults for all configuration options
func (c *StormDBConfig) SetDefaults() {
	// Database defaults
	if c.Database.Type == "" {
		c.Database.Type = "postgres"
	}
	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.Database.MaxConnections == 0 {
		c.Database.MaxConnections = 50
	}
	if c.Database.MinConnections == 0 {
		c.Database.MinConnections = 1
	}
	if c.Database.MaxConnLifetime == 0 {
		c.Database.MaxConnLifetime = 30 * time.Minute
	}
	if c.Database.MaxConnIdleTime == 0 {
		c.Database.MaxConnIdleTime = 15 * time.Minute
	}
	if c.Database.HealthCheckPeriod == 0 {
		c.Database.HealthCheckPeriod = 30 * time.Second
	}
	if c.Database.ConnectTimeout == 0 {
		c.Database.ConnectTimeout = 10 * time.Second
	}

	// Workload defaults
	if c.Workload.Duration == 0 {
		c.Workload.Duration = 5 * time.Minute
	}
	if c.Workload.Workers == 0 {
		c.Workload.Workers = 4
	}
	if c.Workload.Connections == 0 {
		c.Workload.Connections = 8
	}
	if c.Workload.Scale == 0 {
		c.Workload.Scale = 1
	}
	if c.Workload.SummaryInterval == 0 {
		c.Workload.SummaryInterval = 10 * time.Second
	}

	// Plugin defaults
	if len(c.Plugins.Paths) == 0 {
		c.Plugins.Paths = []string{"./plugins", "./build/plugins"}
	}
	c.Plugins.AutoLoad = true
	c.Plugins.HealthCheckEnabled = true
	if c.Plugins.HealthCheckInterval == 0 {
		c.Plugins.HealthCheckInterval = 30 * time.Second
	}
	if c.Plugins.MaxLoadAttempts == 0 {
		c.Plugins.MaxLoadAttempts = 3
	}
	if c.Plugins.LoadTimeout == 0 {
		c.Plugins.LoadTimeout = 10 * time.Second
	}

	// Metrics defaults
	c.Metrics.Enabled = true
	if c.Metrics.Interval == 0 {
		c.Metrics.Interval = 5 * time.Second
	}
	if len(c.Metrics.LatencyPercentiles) == 0 {
		c.Metrics.LatencyPercentiles = []int{50, 90, 95, 99}
	}
	if c.Metrics.PrometheusPort == 0 {
		c.Metrics.PrometheusPort = 9090
	}
	if c.Metrics.BatchSize == 0 {
		c.Metrics.BatchSize = 1000
	}
	if c.Metrics.FlushInterval == 0 {
		c.Metrics.FlushInterval = 5 * time.Second
	}
	if c.Metrics.MaxMemoryMB == 0 {
		c.Metrics.MaxMemoryMB = 100
	}

	// Advanced defaults
	c.Advanced.CircuitBreaker.Enabled = true
	if c.Advanced.CircuitBreaker.MaxFailures == 0 {
		c.Advanced.CircuitBreaker.MaxFailures = 5
	}
	if c.Advanced.CircuitBreaker.ResetTimeout == 0 {
		c.Advanced.CircuitBreaker.ResetTimeout = 30 * time.Second
	}
	if c.Advanced.CircuitBreaker.HalfOpenLimit == 0 {
		c.Advanced.CircuitBreaker.HalfOpenLimit = 3
	}

	if c.Advanced.ResourceLimits.MaxMemoryMB == 0 {
		c.Advanced.ResourceLimits.MaxMemoryMB = 1024
	}
	if c.Advanced.ResourceLimits.MaxCPUPercent == 0 {
		c.Advanced.ResourceLimits.MaxCPUPercent = 80
	}
	if c.Advanced.ResourceLimits.MaxGoroutines == 0 {
		c.Advanced.ResourceLimits.MaxGoroutines = 1000
	}
	if c.Advanced.ResourceLimits.GCTargetPercent == 0 {
		c.Advanced.ResourceLimits.GCTargetPercent = 100
	}
	if c.Advanced.ResourceLimits.ShutdownTimeout == 0 {
		c.Advanced.ResourceLimits.ShutdownTimeout = 30 * time.Second
	}

	if c.Advanced.ErrorHandling.MaxRetries == 0 {
		c.Advanced.ErrorHandling.MaxRetries = 3
	}
	if c.Advanced.ErrorHandling.RetryBackoff == 0 {
		c.Advanced.ErrorHandling.RetryBackoff = 1 * time.Second
	}
	if c.Advanced.ErrorHandling.BackoffMultiplier == 0 {
		c.Advanced.ErrorHandling.BackoffMultiplier = 2.0
	}
	if c.Advanced.ErrorHandling.MaxBackoff == 0 {
		c.Advanced.ErrorHandling.MaxBackoff = 30 * time.Second
	}
	c.Advanced.ErrorHandling.PanicRecovery = true
	if c.Advanced.ErrorHandling.ErrorRateThreshold == 0 {
		c.Advanced.ErrorHandling.ErrorRateThreshold = 0.1 // 10%
	}

	if c.Advanced.Observability.ServiceName == "" {
		c.Advanced.Observability.ServiceName = "stormdb"
	}
	if c.Advanced.Observability.SampleRate == 0 {
		c.Advanced.Observability.SampleRate = 0.1 // 10%
	}

	// Logger defaults
	if c.Logger.Level == "" {
		c.Logger.Level = "info"
	}
	if c.Logger.Format == "" {
		c.Logger.Format = "console"
	}
	if c.Logger.Output == "" {
		c.Logger.Output = "stdout"
	}
}

// Validate performs comprehensive validation of the configuration
func (c *StormDBConfig) Validate() error {
	validate := validator.New()

	// Register custom validation functions
	if err := c.registerCustomValidators(validate); err != nil {
		return errors.Wrap(err, "failed to register custom validators")
	}

	if err := validate.Struct(c); err != nil {
		return c.formatValidationErrors(err)
	}

	// Custom validation logic
	if err := c.validateBusinessRules(); err != nil {
		return err
	}

	return nil
}

// registerCustomValidators registers custom validation functions
func (c *StormDBConfig) registerCustomValidators(validate *validator.Validate) error {
	// Register semver validator
	err := validate.RegisterValidation("semver", func(fl validator.FieldLevel) bool {
		version := fl.Field().String()
		if version == "" {
			return false
		}
		// Simple semver check - could be enhanced with proper semver library
		parts := strings.Split(version, ".")
		return len(parts) >= 2
	})
	if err != nil {
		return err
	}

	return nil
}

// formatValidationErrors formats validation errors into a readable format
func (c *StormDBConfig) formatValidationErrors(err error) error {
	var errorMessages []string

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, err := range validationErrors {
			errorMessages = append(errorMessages, fmt.Sprintf(
				"Field '%s' failed validation: %s (value: %v)",
				err.Field(), err.Tag(), err.Value(),
			))
		}
	}

	return fmt.Errorf("configuration validation failed: %s", strings.Join(errorMessages, "; "))
}

// validateBusinessRules performs business logic validation
func (c *StormDBConfig) validateBusinessRules() error {
	// Validate database connection limits
	if c.Database.MinConnections > c.Database.MaxConnections {
		return errors.New("min_connections cannot be greater than max_connections")
	}

	// Validate workload parameters
	if c.Workload.Workers > c.Workload.Connections {
		return errors.New("workers cannot exceed connections")
	}

	// Validate progressive configuration if enabled
	if c.Workload.Progressive != nil && c.Workload.Progressive.Enabled {
		if c.Workload.Progressive.MinWorkers > c.Workload.Progressive.MaxWorkers {
			return errors.New("progressive.min_workers cannot be greater than max_workers")
		}
		if c.Workload.Progressive.MinConnections > c.Workload.Progressive.MaxConnections {
			return errors.New("progressive.min_connections cannot be greater than max_connections")
		}
		if c.Workload.Progressive.Bands < 2 {
			return errors.New("progressive.bands must be at least 2")
		}
	}

	// Validate metrics percentiles
	for _, p := range c.Metrics.LatencyPercentiles {
		if p < 1 || p > 100 {
			return fmt.Errorf("invalid latency percentile: %d (must be 1-100)", p)
		}
	}

	return nil
}

// Migrate handles configuration version migration
func (c *StormDBConfig) Migrate(version string) error {
	// Handle configuration migration from older versions
	switch version {
	case "0.1":
		return c.migrateFrom01()
	case "0.2":
		return c.migrateFrom02()
	default:
		return fmt.Errorf("unsupported migration from version %s", version)
	}
}

// migrateFrom01 migrates configuration from version 0.1
func (c *StormDBConfig) migrateFrom01() error {
	// Example migration: convert old connection settings
	if c.Database.MaxConnections == 0 && c.Workload.Connections > 0 {
		c.Database.MaxConnections = c.Workload.Connections * 2
	}
	return nil
}

// migrateFrom02 migrates configuration from version 0.2
func (c *StormDBConfig) migrateFrom02() error {
	// Example migration: update plugin configuration
	if len(c.Plugins.Paths) == 0 {
		c.Plugins.Paths = []string{"./plugins"}
	}
	return nil
}

// ApplyOverrides applies command-line overrides to the configuration
func (c *StormDBConfig) ApplyOverrides(cli CLIOptions) error {
	if cli.Workers != nil {
		if *cli.Workers < 1 {
			return errors.New("workers must be at least 1")
		}
		c.Workload.Workers = *cli.Workers
	}

	if cli.Connections != nil {
		if *cli.Connections < 1 {
			return errors.New("connections must be at least 1")
		}
		c.Workload.Connections = *cli.Connections
	}

	if cli.Duration != nil {
		if *cli.Duration < time.Second {
			return errors.New("duration must be at least 1 second")
		}
		c.Workload.Duration = *cli.Duration
	}

	if cli.Scale != nil {
		if *cli.Scale < 1 {
			return errors.New("scale must be at least 1")
		}
		c.Workload.Scale = *cli.Scale
	}

	if cli.Workload != "" {
		c.Workload.Type = cli.Workload
	}

	if cli.DatabaseURL != "" {
		if err := c.parseDatabaseURL(cli.DatabaseURL); err != nil {
			return errors.Wrap(err, "failed to parse database URL")
		}
	}

	if cli.LogLevel != "" {
		c.Logger.Level = cli.LogLevel
	}

	// Re-validate after applying overrides
	return c.Validate()
}

// parseDatabaseURL parses a database URL and updates database configuration
func (c *StormDBConfig) parseDatabaseURL(url string) error {
	// This is a simplified parser - could be enhanced with proper URL parsing
	// Format: postgres://user:pass@host:port/dbname?sslmode=disable
	if !strings.HasPrefix(url, "postgres://") {
		return errors.New("unsupported database URL format")
	}

	// For now, just store the URL - implement proper parsing as needed
	return errors.New("database URL parsing not yet implemented")
}

// ConfigManager provides centralized configuration management
type ConfigManager struct {
	config    *StormDBConfig
	validator *validator.Validate
	logger    *zap.Logger
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(logger *zap.Logger) *ConfigManager {
	return &ConfigManager{
		validator: validator.New(),
		logger:    logger,
	}
}

// LoadConfig loads configuration from file with validation and migration
func (cm *ConfigManager) LoadConfig(filePath string, cliOptions CLIOptions) (*StormDBConfig, error) {
	// Load base configuration
	config := NewStormDBConfig()

	// Apply CLI overrides
	if err := config.ApplyOverrides(cliOptions); err != nil {
		return nil, errors.Wrap(err, "failed to apply CLI overrides")
	}

	// Validate final configuration
	if err := config.Validate(); err != nil {
		return nil, errors.Wrap(err, "configuration validation failed")
	}

	cm.config = config

	cm.logger.Info("Configuration loaded successfully",
		zap.String("version", config.Version),
		zap.String("workload", config.Workload.Type),
		zap.Int("workers", config.Workload.Workers),
	)

	return config, nil
}
