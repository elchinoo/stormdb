package core

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// ErrorCode represents different types of errors in the system
type ErrorCode string

const (
	ErrorCodePluginExecution    ErrorCode = "PLUGIN_EXECUTION"
	ErrorCodeDatabaseConnection ErrorCode = "DATABASE_CONNECTION"
	ErrorCodeConfigValidation   ErrorCode = "CONFIG_VALIDATION"
	ErrorCodeResourceLimit      ErrorCode = "RESOURCE_LIMIT"
	ErrorCodeTimeout            ErrorCode = "TIMEOUT"
	ErrorCodePermission         ErrorCode = "PERMISSION"
	ErrorCodeInvalidInput       ErrorCode = "INVALID_INPUT"
)

// PluginError represents a plugin-specific error with context
type PluginError struct {
	PluginName    string                 `json:"plugin_name"`
	PluginVersion string                 `json:"plugin_version"`
	Phase         string                 `json:"phase"` // "initialize", "validate", "execute", "cleanup"
	ErrorCode     ErrorCode              `json:"error_code"`
	Message       string                 `json:"message"`
	Retryable     bool                   `json:"retryable"`
	Context       map[string]interface{} `json:"context,omitempty"`
	Cause         error                  `json:"-"` // Original error, not serialized
	Timestamp     time.Time              `json:"timestamp"`
	TestRunID     *int64                 `json:"test_run_id,omitempty"`
}

func (e *PluginError) Error() string {
	return fmt.Sprintf("plugin %s[%s] failed in %s phase: %s",
		e.PluginName, e.PluginVersion, e.Phase, e.Message)
}

func (e *PluginError) Unwrap() error {
	return e.Cause
}

// NewPluginError creates a new plugin error
func NewPluginError(pluginName, pluginVersion, phase string, code ErrorCode, message string, cause error) *PluginError {
	return &PluginError{
		PluginName:    pluginName,
		PluginVersion: pluginVersion,
		Phase:         phase,
		ErrorCode:     code,
		Message:       message,
		Retryable:     isRetryableError(code),
		Context:       make(map[string]interface{}),
		Cause:         cause,
		Timestamp:     time.Now(),
	}
}

// AddContext adds contextual information to the error
func (e *PluginError) AddContext(key string, value interface{}) *PluginError {
	e.Context[key] = value
	return e
}

// SetTestRunID associates the error with a test run
func (e *PluginError) SetTestRunID(id int64) *PluginError {
	e.TestRunID = &id
	return e
}

// isRetryableError determines if an error type is retryable
func isRetryableError(code ErrorCode) bool {
	switch code {
	case ErrorCodeDatabaseConnection, ErrorCodeTimeout, ErrorCodeResourceLimit:
		return true
	case ErrorCodeConfigValidation, ErrorCodePermission, ErrorCodeInvalidInput:
		return false
	default:
		return false
	}
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for plugin execution
type CircuitBreaker struct {
	name             string
	failureThreshold int
	resetTimeout     time.Duration
	state            CircuitBreakerState
	failures         int
	lastFailureTime  time.Time
	mutex            sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            CircuitBreakerClosed,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Check if circuit breaker should be reset
	if cb.state == CircuitBreakerOpen && time.Since(cb.lastFailureTime) > cb.resetTimeout {
		cb.state = CircuitBreakerHalfOpen
		cb.failures = 0
	}

	// Reject execution if circuit is open
	if cb.state == CircuitBreakerOpen {
		return fmt.Errorf("circuit breaker %s is open", cb.name)
	}

	// Execute the function
	err := fn()

	if err != nil {
		cb.failures++
		cb.lastFailureTime = time.Now()

		// Open circuit if threshold is exceeded
		if cb.failures >= cb.failureThreshold {
			cb.state = CircuitBreakerOpen
		}

		return err
	}

	// Reset on success
	if cb.state == CircuitBreakerHalfOpen {
		cb.state = CircuitBreakerClosed
	}
	cb.failures = 0

	return nil
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// RetryPolicy defines retry behavior for failed operations
type RetryPolicy struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Jitter        bool
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:   3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// Retry executes a function with retry logic
func (rp *RetryPolicy) Retry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < rp.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := rp.calculateDelay(attempt)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := fn(); err != nil {
			lastErr = err

			// Check if error is retryable
			if pluginErr, ok := err.(*PluginError); ok && !pluginErr.Retryable {
				return err // Don't retry non-retryable errors
			}

			continue
		}

		return nil // Success
	}

	return fmt.Errorf("operation failed after %d attempts: %w", rp.MaxAttempts, lastErr)
}

// calculateDelay calculates the delay for the given attempt
func (rp *RetryPolicy) calculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(rp.InitialDelay) *
		math.Pow(rp.BackoffFactor, float64(attempt-1)))

	if delay > rp.MaxDelay {
		delay = rp.MaxDelay
	}

	if rp.Jitter {
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)
		delay += jitter
	}

	return delay
}

// ErrorRecoveryManager handles error recovery and reporting
type ErrorRecoveryManager struct {
	circuitBreakers map[string]*CircuitBreaker
	retryPolicy     *RetryPolicy
	logger          Logger
	storage         StorageManager
	mutex           sync.RWMutex
}

// NewErrorRecoveryManager creates a new error recovery manager
func NewErrorRecoveryManager(logger Logger, storage StorageManager) *ErrorRecoveryManager {
	return &ErrorRecoveryManager{
		circuitBreakers: make(map[string]*CircuitBreaker),
		retryPolicy:     DefaultRetryPolicy(),
		logger:          logger,
		storage:         storage,
	}
}

// ExecuteWithRecovery executes a plugin operation with error recovery
func (erm *ErrorRecoveryManager) ExecuteWithRecovery(
	ctx context.Context,
	pluginName string,
	operation func() error,
) error {
	cb := erm.getOrCreateCircuitBreaker(pluginName)

	return erm.retryPolicy.Retry(ctx, func() error {
		return cb.Execute(operation)
	})
}

// getOrCreateCircuitBreaker gets or creates a circuit breaker for a plugin
func (erm *ErrorRecoveryManager) getOrCreateCircuitBreaker(pluginName string) *CircuitBreaker {
	erm.mutex.Lock()
	defer erm.mutex.Unlock()

	if cb, exists := erm.circuitBreakers[pluginName]; exists {
		return cb
	}

	cb := NewCircuitBreaker(pluginName, 5, 30*time.Second)
	erm.circuitBreakers[pluginName] = cb
	return cb
}

// RecordError records an error for analysis and reporting
func (erm *ErrorRecoveryManager) RecordError(ctx context.Context, err *PluginError) {
	erm.logger.Error("plugin error recorded",
		Field{Key: "plugin", Value: err.PluginName},
		Field{Key: "phase", Value: err.Phase},
		Field{Key: "error_code", Value: err.ErrorCode},
		Field{Key: "message", Value: err.Message},
		Field{Key: "retryable", Value: err.Retryable},
	)

	// Store error in database for analysis
	if erm.storage != nil && err.TestRunID != nil {
		errorDetails := map[string]interface{}{
			"error_code": err.ErrorCode,
			"phase":      err.Phase,
			"retryable":  err.Retryable,
			"context":    err.Context,
			"timestamp":  err.Timestamp,
		}

		_ = erm.storage.UpdateTestRunWithError(
			ctx,
			*err.TestRunID,
			StatusFailed,
			err.Message,
			errorDetails,
		)
	}
}

// GetErrorStatistics returns error statistics for monitoring
func (erm *ErrorRecoveryManager) GetErrorStatistics() map[string]interface{} {
	erm.mutex.RLock()
	defer erm.mutex.RUnlock()

	stats := make(map[string]interface{})
	circuitBreakerStats := make(map[string]interface{})

	for name, cb := range erm.circuitBreakers {
		circuitBreakerStats[name] = map[string]interface{}{
			"state":    cb.GetState(),
			"failures": cb.failures,
		}
	}

	stats["circuit_breakers"] = circuitBreakerStats
	stats["retry_policy"] = map[string]interface{}{
		"max_attempts":   erm.retryPolicy.MaxAttempts,
		"initial_delay":  erm.retryPolicy.InitialDelay,
		"max_delay":      erm.retryPolicy.MaxDelay,
		"backoff_factor": erm.retryPolicy.BackoffFactor,
	}

	return stats
}
