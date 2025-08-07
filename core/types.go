// Package core defines the fundamental types and interfaces for StormDB v0.4-alpha
// This package provides the core infrastructure for the plugin-based database performance testing framework
package core

import (
	"context"
	"database/sql"
	"time"
)

// Service status enumeration
type ServiceStatus string

const (
	StatusPending   ServiceStatus = "pending"
	StatusRunning   ServiceStatus = "running"
	StatusSucceeded ServiceStatus = "succeeded"
	StatusFailed    ServiceStatus = "failed"
	StatusAborted   ServiceStatus = "aborted"
)

// Standard test metric codes (must be registered in database)
const (
	MetricRowInsert = "ROW_INSERT"
	MetricRowSelect = "ROW_SELECT"
	MetricRowUpdate = "ROW_UPDATE"
	MetricRowDelete = "ROW_DELETE"
	MetricRowCopy   = "ROW_COPY"
)

// CoreConfig represents the global configuration for the core services
type CoreConfig struct {
	Database   DatabaseConfig  `yaml:"database" json:"database"`
	API        APIConfig       `yaml:"api" json:"api"`
	Logging    LoggingConfig   `yaml:"logging" json:"logging"`
	Scheduler  SchedulerConfig `yaml:"scheduler" json:"scheduler"`
	PluginDirs []string        `yaml:"plugin_dirs" json:"plugin_dirs"`
}

// DatabaseConfig holds database connection and pool settings
type DatabaseConfig struct {
	Host            string `yaml:"host" json:"host"`
	Port            int    `yaml:"port" json:"port"`
	Database        string `yaml:"database" json:"database"`
	Username        string `yaml:"username" json:"username"`
	Password        string `yaml:"password" json:"password"`
	SSLMode         string `yaml:"ssl_mode" json:"ssl_mode"`
	MaxConnections  int    `yaml:"max_connections" json:"max_connections"`
	MaxIdleConns    int    `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime string `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
}

// APIConfig configures both REST and gRPC API servers
type APIConfig struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
	GRPC struct {
		Enabled bool `yaml:"enabled" json:"enabled"`
		Port    int  `yaml:"port" json:"port"`
	} `yaml:"grpc" json:"grpc"`
}

// LoggingConfig defines structured logging settings
type LoggingConfig struct {
	Level  string `yaml:"level" json:"level"`   // debug, info, warn, error
	Format string `yaml:"format" json:"format"` // json, text
	Output string `yaml:"output" json:"output"` // stdout, file, syslog
	File   string `yaml:"file" json:"file"`     // if output=file
}

// SchedulerConfig controls task scheduling and worker pool
type SchedulerConfig struct {
	Enabled        bool `yaml:"enabled" json:"enabled"`
	WorkerPoolSize int  `yaml:"worker_pool_size" json:"worker_pool_size"`
}

// PluginMetadata contains plugin registration information
type PluginMetadata struct {
	ID           int               `json:"id,omitempty"`
	Name         string            `yaml:"name" json:"name"`
	Version      string            `yaml:"version" json:"version"`
	Description  string            `yaml:"description" json:"description"`
	Author       string            `yaml:"author" json:"author"`
	License      string            `yaml:"license" json:"license"`
	TestTypes    []string          `yaml:"test_types" json:"test_types"`
	ConfigSchema string            `yaml:"config_schema" json:"config_schema"` // JSON Schema
	Dependencies map[string]string `yaml:"dependencies" json:"dependencies"`
	SHA256       string            `yaml:"sha256" json:"sha256"` // Build integrity hash
}

// TestRun represents a test execution instance
type TestRun struct {
	ID           int64                  `json:"id"`
	TestTypeID   int                    `json:"test_type_id"`
	PluginID     int                    `json:"plugin_id"`
	Host         string                 `json:"host"`
	Port         int                    `json:"port"`
	DBName       string                 `json:"db_name"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Status       ServiceStatus          `json:"status"`
	Config       map[string]interface{} `json:"config"`
	CreatedAt    time.Time              `json:"created_at"`
	StartTime    *time.Time             `json:"start_time,omitempty"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	ErrorMessage *string                `json:"error_message,omitempty"`
	ErrorDetails map[string]interface{} `json:"error_details,omitempty"`
	LogsURL      *string                `json:"logs_url,omitempty"`
}

// TestResult represents a single measurement from a test execution
type TestResult struct {
	ID                int64                  `json:"id"`
	TestRunID         int64                  `json:"test_run_id"`
	MetricID          int                    `json:"metric_id"`
	StartTime         time.Time              `json:"start_time"`
	EndTime           time.Time              `json:"end_time"`
	Value             float64                `json:"value"`
	Tags              map[string]interface{} `json:"tags,omitempty"`               // Flexible metadata, stored as JSONB
	ActiveConnections *int                   `json:"active_connections,omitempty"` // Number of active database connections
	ActiveWorkers     *int                   `json:"active_workers,omitempty"`     // Number of active worker threads/processes
}

// TestType represents a category of tests (e.g., bulk_insert, read_latency)
type TestType struct {
	ID          int       `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsActive    bool      `json:"is_active"`
}

// TestMetric defines a measurable aspect of test performance
type TestMetric struct {
	ID          int       `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Unit        string    `json:"unit"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsActive    bool      `json:"is_active"`
}

// Task represents a schedulable unit of work
type Task interface {
	ID() string
	Type() string
	Execute(ctx context.Context) error
}

// SchedulerStatus provides runtime information about the scheduler
type SchedulerStatus struct {
	Running            bool `json:"running"`
	WorkerCount        int  `json:"worker_count"`
	QueueLength        int  `json:"queue_length"`
	ScheduledTaskCount int  `json:"scheduled_task_count"`
}

// Plugin interface that all test plugins must implement
type Plugin interface {
	// Plugin metadata and identification
	Metadata() PluginMetadata

	// Plugin lifecycle management
	Initialize(ctx context.Context, core *CoreServices) error
	Validate(config map[string]interface{}) error
	Execute(ctx context.Context, config map[string]interface{}) error
	Cleanup(ctx context.Context) error
}

// CoreServices provides all core infrastructure to plugins
type CoreServices struct {
	Database  DatabaseManager
	Logger    Logger
	Storage   StorageManager
	Config    ConfigManager
	Scheduler SchedulerManager
	Plugin    PluginManager
}

// LogEntry represents a single log record for a test run
type LogEntry struct {
	TestRunID int64                  `json:"test_run_id"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// DatabaseManager provides connection pooling and database operations
type DatabaseManager interface {
	// Connection lifecycle
	Connect(ctx context.Context) error
	GetConnection(ctx context.Context) (*sql.DB, error)
	GetConnectionPool() *sql.DB
	Health(ctx context.Context) error
	Close() error

	// Schema management
	Migrate(ctx context.Context) error

	// Transaction support for plugins
	BeginTransaction(ctx context.Context) (*sql.Tx, error)
}

// Logger provides structured logging for core and plugins
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	WithFields(fields ...Field) Logger
	WithPlugin(pluginName string) Logger
	// WithStorage provides a StorageManager to enable DB log persistence
	WithStorage(storage StorageManager) Logger
}

// Field represents a structured log field
type Field struct {
	Key   string
	Value interface{}
}

// StorageManager handles test results and metadata persistence
type StorageManager interface {
	// Test run lifecycle
	CreateTestRun(ctx context.Context, run *TestRun) (int64, error)
	UpdateTestRunStatus(ctx context.Context, runID int64, status ServiceStatus) error
	UpdateTestRunWithError(ctx context.Context, runID int64, status ServiceStatus, errorMessage string, errorDetails map[string]interface{}) error
	GetTestRun(ctx context.Context, runID int64) (*TestRun, error)
	ListTestRuns(ctx context.Context, limit, offset int) ([]TestRun, error)

	// Result storage and retrieval
	StoreResults(ctx context.Context, results []TestResult) error
	GetResults(ctx context.Context, runID int64) ([]TestResult, error)
	GetResultsByMetric(ctx context.Context, metricCode string, limit int) ([]TestResult, error)

	// Metadata management
	RegisterTestType(ctx context.Context, code, name, description string) (int, error)
	GetTestType(ctx context.Context, code string) (*TestType, error)
	RegisterPlugin(ctx context.Context, metadata PluginMetadata) (int, error)
	GetPlugin(ctx context.Context, name, version string) (*PluginMetadata, error)
	RegisterMetric(ctx context.Context, code, description, unit string) (int, error)
	GetMetric(ctx context.Context, code string) (*TestMetric, error)
	// Log persistence
	StoreLog(ctx context.Context, entry LogEntry) error
	GetTestRunLogs(ctx context.Context, testRunID int64, limit int) ([]LogEntry, error)
	// Fix stuck tests on startup: set status=failed for runs still running or pending
	FixStuckTests(ctx context.Context) (int64, error)
}

// ConfigManager handles global and plugin-specific configuration
type ConfigManager interface {
	// Global configuration
	GetGlobal() *CoreConfig
	Reload() error

	// Plugin configuration
	GetPlugin(pluginName string) (map[string]interface{}, error)
	SetPlugin(pluginName string, config map[string]interface{}) error

	// Configuration validation
	Validate(config map[string]interface{}, schema string) error
}

// SchedulerManager orchestrates test execution and task scheduling
type SchedulerManager interface {
	// Lifecycle management
	Start() error
	Stop() error
	IsRunning() bool
	GetStatus() SchedulerStatus

	// Task management
	SubmitTask(task Task) error
	ScheduleTask(taskID string, task Task, interval time.Duration) error
	CancelTask(taskID string) error

	// Test execution orchestration
	ScheduleTest(ctx context.Context, plugin Plugin, config map[string]interface{}) (int64, error)
	CancelTest(ctx context.Context, runID int64) error
	GetRunStatus(ctx context.Context, runID int64) (ServiceStatus, error)
	ListActiveRuns(ctx context.Context) ([]TestRun, error)
}

// PluginManager handles dynamic plugin loading and lifecycle
type PluginManager interface {
	// Plugin loading and discovery
	LoadPlugin(path string) (Plugin, error)
	LoadPlugins() error
	UnloadPlugins() error

	// Plugin registry
	GetPlugin(name string, version string) (Plugin, error)
	GetLoadedPlugins() []Plugin
	ListPlugins() []PluginMetadata

	// Plugin validation and security
	ValidatePlugin(plugin Plugin) error
	ValidatePluginFile(path string) error
	RegisterPlugin(plugin Plugin) error
	UnregisterPlugin(name string) error
}
