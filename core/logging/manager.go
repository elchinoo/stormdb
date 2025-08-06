// Package logging provides structured logging for StormDB v2
// Supports both JSON and text formats with configurable outputs
package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/elchinoo/stormdb/core"
)

// Manager implements the Logger interface
type Manager struct {
	logger *slog.Logger
	config *core.LoggingConfig
	level  slog.Level
	fields []core.Field
	writer io.Writer
}

// NewManager creates a new logging manager
func NewManager(config *core.LoggingConfig) (*Manager, error) {
	level := parseLogLevel(config.Level)

	// Determine output writer
	writer, err := getWriter(config.Output, config.File)
	if err != nil {
		return nil, fmt.Errorf("failed to setup log writer: %w", err)
	}

	// Create handler based on format
	var handler slog.Handler
	switch strings.ToLower(config.Format) {
	case "json":
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				// Customize timestamp format
				if a.Key == slog.TimeKey {
					a.Value = slog.StringValue(time.Now().Format(time.RFC3339))
				}
				return a
			},
		})
	case "text":
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: level,
		})
	default:
		return nil, fmt.Errorf("unsupported log format: %s", config.Format)
	}

	logger := slog.New(handler)

	return &Manager{
		logger: logger,
		config: config,
		level:  level,
		writer: writer,
	}, nil
}

// Debug logs a debug message
func (m *Manager) Debug(msg string, fields ...core.Field) {
	if m.level <= slog.LevelDebug {
		m.log(slog.LevelDebug, msg, fields...)
	}
}

// Info logs an info message
func (m *Manager) Info(msg string, fields ...core.Field) {
	if m.level <= slog.LevelInfo {
		m.log(slog.LevelInfo, msg, fields...)
	}
}

// Warn logs a warning message
func (m *Manager) Warn(msg string, fields ...core.Field) {
	if m.level <= slog.LevelWarn {
		m.log(slog.LevelWarn, msg, fields...)
	}
}

// Error logs an error message
func (m *Manager) Error(msg string, fields ...core.Field) {
	if m.level <= slog.LevelError {
		m.log(slog.LevelError, msg, fields...)
	}
}

// WithFields returns a new logger with additional fields
func (m *Manager) WithFields(fields ...core.Field) core.Logger {
	newFields := make([]core.Field, len(m.fields)+len(fields))
	copy(newFields, m.fields)
	copy(newFields[len(m.fields):], fields)

	return &Manager{
		logger: m.logger,
		config: m.config,
		level:  m.level,
		fields: newFields,
		writer: m.writer,
	}
}

// WithPlugin returns a new logger with plugin context
func (m *Manager) WithPlugin(pluginName string) core.Logger {
	return m.WithFields(core.Field{Key: "plugin", Value: pluginName})
}

// log is the internal logging method
func (m *Manager) log(level slog.Level, msg string, fields ...core.Field) {
	// Combine permanent fields with temporary fields
	allFields := make([]core.Field, len(m.fields)+len(fields))
	copy(allFields, m.fields)
	copy(allFields[len(m.fields):], fields)

	// Convert fields to slog.Attr
	attrs := make([]slog.Attr, len(allFields))
	for i, field := range allFields {
		attrs[i] = slog.Attr{
			Key:   field.Key,
			Value: slog.AnyValue(field.Value),
		}
	}

	// Log with attributes using context.Background()
	m.logger.LogAttrs(context.Background(), level, msg, attrs...)
}

// Close closes any file handles (if logging to file)
func (m *Manager) Close() error {
	if closer, ok := m.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// SetLevel updates the log level dynamically
func (m *Manager) SetLevel(level string) error {
	newLevel := parseLogLevel(level)
	m.level = newLevel

	// Update the config
	m.config.Level = level

	return nil
}

// GetLevel returns the current log level
func (m *Manager) GetLevel() string {
	return m.config.Level
}

// LogPluginEvent logs a structured plugin event
func (m *Manager) LogPluginEvent(pluginName, event string, fields ...core.Field) {
	eventFields := []core.Field{
		{Key: "plugin", Value: pluginName},
		{Key: "event", Value: event},
		{Key: "timestamp", Value: time.Now().Format(time.RFC3339)},
	}
	eventFields = append(eventFields, fields...)

	m.Info("plugin event", eventFields...)
}

// LogTestEvent logs a structured test execution event
func (m *Manager) LogTestEvent(testRunID int64, event string, fields ...core.Field) {
	eventFields := []core.Field{
		{Key: "test_run_id", Value: testRunID},
		{Key: "event", Value: event},
		{Key: "timestamp", Value: time.Now().Format(time.RFC3339)},
	}
	eventFields = append(eventFields, fields...)

	m.Info("test event", eventFields...)
}

// LogMetric logs a performance metric
func (m *Manager) LogMetric(metricName string, value float64, unit string, fields ...core.Field) {
	metricFields := []core.Field{
		{Key: "metric", Value: metricName},
		{Key: "value", Value: value},
		{Key: "unit", Value: unit},
		{Key: "timestamp", Value: time.Now().Format(time.RFC3339)},
	}
	metricFields = append(metricFields, fields...)

	m.Info("metric", metricFields...)
}

// LogError logs an error with stack trace information
func (m *Manager) LogError(err error, msg string, fields ...core.Field) {
	errorFields := []core.Field{
		{Key: "error", Value: err.Error()},
		{Key: "error_type", Value: fmt.Sprintf("%T", err)},
	}
	errorFields = append(errorFields, fields...)

	m.Error(msg, errorFields...)
}

// parseLogLevel converts string level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// getWriter returns the appropriate io.Writer based on output configuration
func getWriter(output, file string) (io.Writer, error) {
	switch strings.ToLower(output) {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "file":
		if file == "" {
			return nil, fmt.Errorf("file output requires file path")
		}
		return os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	default:
		return nil, fmt.Errorf("unsupported output type: %s", output)
	}
}

// StructuredEvent represents a structured log event for complex logging
type StructuredEvent struct {
	Event     string                 `json:"event"`
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context"`
}

// LogStructuredEvent logs a complex structured event
func (m *Manager) LogStructuredEvent(event StructuredEvent) {
	// Convert to fields
	fields := []core.Field{
		{Key: "event", Value: event.Event},
		{Key: "timestamp", Value: event.Timestamp},
	}

	// Add context fields
	for key, value := range event.Context {
		fields = append(fields, core.Field{Key: key, Value: value})
	}

	// Log at appropriate level
	switch strings.ToLower(event.Level) {
	case "debug":
		m.Debug(event.Message, fields...)
	case "info":
		m.Info(event.Message, fields...)
	case "warn", "warning":
		m.Warn(event.Message, fields...)
	case "error":
		m.Error(event.Message, fields...)
	default:
		m.Info(event.Message, fields...)
	}
}

// GetJSONLogger returns a logger that always outputs JSON (useful for APIs)
func (m *Manager) GetJSONLogger() core.Logger {
	jsonHandler := slog.NewJSONHandler(m.writer, &slog.HandlerOptions{
		Level: m.level,
	})

	return &Manager{
		logger: slog.New(jsonHandler),
		config: m.config,
		level:  m.level,
		fields: m.fields,
		writer: m.writer,
	}
}

// Flush ensures all log messages are written (useful for file logging)
func (m *Manager) Flush() error {
	if flusher, ok := m.writer.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

// CreateRequestLogger creates a logger with request context
func (m *Manager) CreateRequestLogger(requestID, userID string) core.Logger {
	return m.WithFields(
		core.Field{Key: "request_id", Value: requestID},
		core.Field{Key: "user_id", Value: userID},
	)
}

// LogConfigChange logs configuration changes
func (m *Manager) LogConfigChange(component, setting string, oldValue, newValue interface{}) {
	m.Info("configuration changed",
		core.Field{Key: "component", Value: component},
		core.Field{Key: "setting", Value: setting},
		core.Field{Key: "old_value", Value: oldValue},
		core.Field{Key: "new_value", Value: newValue},
		core.Field{Key: "timestamp", Value: time.Now().Format(time.RFC3339)},
	)
}

// ToJSON converts a log entry to JSON format (useful for API responses)
func (m *Manager) ToJSON(level, message string, fields ...core.Field) ([]byte, error) {
	entry := map[string]interface{}{
		"level":     level,
		"message":   message,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Add fields
	for _, field := range fields {
		entry[field.Key] = field.Value
	}

	return json.Marshal(entry)
}
