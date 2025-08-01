package logging

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// StormDBLogger provides structured logging interface for StormDB
type StormDBLogger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, err error, fields ...zap.Field)
	Fatal(msg string, err error, fields ...zap.Field)
	With(fields ...zap.Field) StormDBLogger
	Sync() error
}

// Logger implements StormDBLogger using zap
type Logger struct {
	logger *zap.Logger
}

// LoggerConfig defines logger configuration
type LoggerConfig struct {
	Level       string `yaml:"level"`
	Format      string `yaml:"format"`
	Output      string `yaml:"output"`
	Development bool   `yaml:"development"`
}

// NewLogger creates a new structured logger based on configuration
func NewLogger(config LoggerConfig) (StormDBLogger, error) {
	// Parse log level
	level, err := parseLogLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}

	// Configure encoder
	var encoderConfig zapcore.EncoderConfig
	if config.Development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	}

	// Choose encoder format
	var encoder zapcore.Encoder
	switch strings.ToLower(config.Format) {
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case "console", "":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		return nil, fmt.Errorf("unsupported log format: %s", config.Format)
	}

	// Configure output
	var writeSyncer zapcore.WriteSyncer
	switch strings.ToLower(config.Output) {
	case "stdout", "":
		writeSyncer = zapcore.AddSync(os.Stdout)
	case "stderr":
		writeSyncer = zapcore.AddSync(os.Stderr)
	default:
		// File output
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		writeSyncer = zapcore.AddSync(file)
	}

	// Create core and logger
	core := zapcore.NewCore(encoder, writeSyncer, level)
	
	// Add caller info and stack traces for errors in development
	var options []zap.Option
	if config.Development {
		options = append(options, zap.Development(), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	} else {
		options = append(options, zap.AddCaller())
	}

	logger := zap.New(core, options...)
	
	return &Logger{logger: logger}, nil
}

// NewDefaultLogger creates a logger with sensible defaults for development
func NewDefaultLogger() StormDBLogger {
	config := LoggerConfig{
		Level:       "info",
		Format:      "console",
		Output:      "stdout",
		Development: true,
	}
	
	logger, err := NewLogger(config)
	if err != nil {
		// Fallback to basic zap logger
		zapLogger, _ := zap.NewDevelopment()
		return &Logger{logger: zapLogger}
	}
	
	return logger
}

// Debug logs a debug message with optional fields
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.logger.Debug(msg, fields...)
}

// Info logs an info message with optional fields
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

// Warn logs a warning message with optional fields
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

// Error logs an error message with error and optional fields
func (l *Logger) Error(msg string, err error, fields ...zap.Field) {
	allFields := make([]zap.Field, 0, len(fields)+1)
	if err != nil {
		allFields = append(allFields, zap.Error(err))
	}
	allFields = append(allFields, fields...)
	l.logger.Error(msg, allFields...)
}

// Fatal logs a fatal message with error and optional fields, then calls os.Exit(1)
func (l *Logger) Fatal(msg string, err error, fields ...zap.Field) {
	allFields := make([]zap.Field, 0, len(fields)+1)
	if err != nil {
		allFields = append(allFields, zap.Error(err))
	}
	allFields = append(allFields, fields...)
	l.logger.Fatal(msg, allFields...)
}

// With creates a child logger with additional fields
func (l *Logger) With(fields ...zap.Field) StormDBLogger {
	return &Logger{logger: l.logger.With(fields...)}
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.logger.Sync()
}

// parseLogLevel converts string level to zapcore.Level
func parseLogLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info", "":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown log level: %s", level)
	}
}

// LoggerFields provides common field constructors for structured logging
type LoggerFields struct{}

// Fields provides convenient field constructors
var Fields LoggerFields

// String creates a string field
func (LoggerFields) String(key, value string) zap.Field {
	return zap.String(key, value)
}

// Int creates an int field
func (LoggerFields) Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

// Int64 creates an int64 field
func (LoggerFields) Int64(key string, value int64) zap.Field {
	return zap.Int64(key, value)
}

// Float64 creates a float64 field
func (LoggerFields) Float64(key string, value float64) zap.Field {
	return zap.Float64(key, value)
}

// Bool creates a bool field
func (LoggerFields) Bool(key string, value bool) zap.Field {
	return zap.Bool(key, value)
}

// Duration creates a duration field
func (LoggerFields) Duration(key string, value interface{}) zap.Field {
	switch v := value.(type) {
	case int64:
		return zap.Duration(key, time.Duration(v))
	case time.Duration:
		return zap.Duration(key, v)
	default:
		return zap.String(key, fmt.Sprintf("%v", value))
	}
}

// Error creates an error field
func (LoggerFields) Error(err error) zap.Field {
	return zap.Error(err)
}

// Any creates a field with any value type
func (LoggerFields) Any(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

// Workload creates fields for workload context
func (LoggerFields) Workload(name string, workers, connections int) []zap.Field {
	return []zap.Field{
		zap.String("workload", name),
		zap.Int("workers", workers),
		zap.Int("connections", connections),
	}
}

// Database creates fields for database context
func (LoggerFields) Database(host string, port int, database string) []zap.Field {
	return []zap.Field{
		zap.String("db_host", host),
		zap.Int("db_port", port),
		zap.String("db_name", database),
	}
}

// Plugin creates fields for plugin context
func (LoggerFields) Plugin(name, version string) []zap.Field {
	return []zap.Field{
		zap.String("plugin_name", name),
		zap.String("plugin_version", version),
	}
}

// Metrics creates fields for metrics context
func (LoggerFields) Metrics(tps, qps float64, errors int64) []zap.Field {
	return []zap.Field{
		zap.Float64("tps", tps),
		zap.Float64("qps", qps),
		zap.Int64("errors", errors),
	}
}

// Performance creates fields for performance metrics
func (LoggerFields) Performance(latencyP50, latencyP95, latencyP99 float64) []zap.Field {
	return []zap.Field{
		zap.Float64("latency_p50_ms", latencyP50),
		zap.Float64("latency_p95_ms", latencyP95),
		zap.Float64("latency_p99_ms", latencyP99),
	}
}
