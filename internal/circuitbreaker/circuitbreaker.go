package circuitbreaker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/elchinoo/stormdb/internal/logging"
	"go.uber.org/zap"
)

// State represents the circuit breaker state
type State int

const (
	// StateClosed means requests are allowed through
	StateClosed State = iota
	// StateOpen means requests are blocked
	StateOpen
	// StateHalfOpen means limited requests are allowed to test if the service has recovered
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker implements the circuit breaker pattern to prevent cascade failures
type CircuitBreaker struct {
	maxFailures    int
	resetTimeout   time.Duration
	halfOpenLimit  int
	
	// State management
	state         State
	failures      int
	lastFailure   time.Time
	halfOpenCount int
	
	// Metrics
	totalRequests   int64
	successCount    int64
	failureCount    int64
	timeoutCount    int64
	rejectedCount   int64
	
	mutex  sync.RWMutex
	logger logging.StormDBLogger
}

// Config configures circuit breaker behavior
type Config struct {
	MaxFailures   int           // Number of failures before opening
	ResetTimeout  time.Duration // Time to wait before trying half-open
	HalfOpenLimit int           // Max requests to allow in half-open state
	Logger        logging.StormDBLogger
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config Config) *CircuitBreaker {
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 30 * time.Second
	}
	if config.HalfOpenLimit <= 0 {
		config.HalfOpenLimit = 3
	}
	if config.Logger == nil {
		config.Logger = logging.NewDefaultLogger()
	}

	return &CircuitBreaker{
		maxFailures:   config.MaxFailures,
		resetTimeout:  config.ResetTimeout,
		halfOpenLimit: config.HalfOpenLimit,
		state:         StateClosed,
		logger:        config.Logger.With(zap.String("component", "circuit_breaker")),
	}
}

// Execute executes the given operation with circuit breaker protection
func (cb *CircuitBreaker) Execute(operation func() error) error {
	if !cb.allowRequest() {
		cb.mutex.Lock()
		cb.rejectedCount++
		cb.mutex.Unlock()
		
		return &CircuitBreakerError{
			State:   cb.getState(),
			Message: "circuit breaker is open",
		}
	}

	start := time.Now()
	cb.mutex.Lock()
	cb.totalRequests++
	cb.mutex.Unlock()

	err := operation()
	duration := time.Since(start)

	if err != nil {
		cb.recordFailure(err, duration)
		return err
	}

	cb.recordSuccess(duration)
	return nil
}

// ExecuteWithContext executes operation with context and circuit breaker protection
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, operation func(context.Context) error) error {
	if !cb.allowRequest() {
		cb.mutex.Lock()
		cb.rejectedCount++
		cb.mutex.Unlock()
		
		return &CircuitBreakerError{
			State:   cb.getState(),
			Message: "circuit breaker is open",
		}
	}

	start := time.Now()
	cb.mutex.Lock()
	cb.totalRequests++
	cb.mutex.Unlock()

	// Create a channel to capture the operation result
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("operation panicked: %v", r)
			}
		}()
		done <- operation(ctx)
	}()

	var err error
	select {
	case err = <-done:
		// Operation completed
	case <-ctx.Done():
		// Context cancelled/timeout
		err = ctx.Err()
		cb.mutex.Lock()
		cb.timeoutCount++
		cb.mutex.Unlock()
	}

	duration := time.Since(start)

	if err != nil {
		cb.recordFailure(err, duration)
		return err
	}

	cb.recordSuccess(duration)
	return nil
}

// allowRequest determines if a request should be allowed through
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mutex.RLock()
	state := cb.state
	halfOpenCount := cb.halfOpenCount
	cb.mutex.RUnlock()

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		return cb.shouldAttemptReset()
	case StateHalfOpen:
		return halfOpenCount < cb.halfOpenLimit
	default:
		return false
	}
}

// shouldAttemptReset checks if enough time has passed to attempt reset
func (cb *CircuitBreaker) shouldAttemptReset() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if time.Since(cb.lastFailure) >= cb.resetTimeout {
		cb.state = StateHalfOpen
		cb.halfOpenCount = 0
		cb.logger.Info("Circuit breaker transitioning to half-open state",
			zap.Duration("reset_timeout", cb.resetTimeout),
		)
		return true
	}
	return false
}

// recordSuccess records a successful operation
func (cb *CircuitBreaker) recordSuccess(duration time.Duration) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.successCount++

	switch cb.state {
	case StateHalfOpen:
		cb.halfOpenCount++
		if cb.halfOpenCount >= cb.halfOpenLimit {
			// Enough successful requests in half-open, transition to closed
			cb.state = StateClosed
			cb.failures = 0
			cb.halfOpenCount = 0
			cb.logger.Info("Circuit breaker closed after successful half-open test",
				zap.Int("successful_requests", cb.halfOpenLimit),
				zap.Duration("operation_duration", duration),
			)
		}
	case StateClosed:
		// Reset failure count on success
		cb.failures = 0
	}

	cb.logger.Debug("Circuit breaker recorded success",
		zap.String("state", cb.state.String()),
		zap.Duration("duration", duration),
		zap.Int64("total_successes", cb.successCount),
	)
}

// recordFailure records a failed operation
func (cb *CircuitBreaker) recordFailure(err error, duration time.Duration) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount++
	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == StateHalfOpen {
		// Any failure in half-open state transitions back to open
		cb.state = StateOpen
		cb.halfOpenCount = 0
		cb.logger.Warn("Circuit breaker opened due to failure in half-open state",
			zap.Error(err),
			zap.Duration("duration", duration),
		)
	} else if cb.failures >= cb.maxFailures {
		// Too many failures, open the circuit
		cb.state = StateOpen
		cb.logger.Error("Circuit breaker opened due to excessive failures",
			err,
			zap.Int("failures", cb.failures),
			zap.Int("max_failures", cb.maxFailures),
			zap.Duration("duration", duration),
		)
	}

	cb.logger.Debug("Circuit breaker recorded failure",
		zap.String("state", cb.state.String()),
		zap.Error(err),
		zap.Duration("duration", duration),
		zap.Int("current_failures", cb.failures),
		zap.Int64("total_failures", cb.failureCount),
	)
}

// getState returns the current state safely
func (cb *CircuitBreaker) getState() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() Metrics {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return Metrics{
		State:           cb.state,
		TotalRequests:   cb.totalRequests,
		SuccessCount:    cb.successCount,
		FailureCount:    cb.failureCount,
		TimeoutCount:    cb.timeoutCount,
		RejectedCount:   cb.rejectedCount,
		CurrentFailures: cb.failures,
		LastFailure:     cb.lastFailure,
		SuccessRate:     cb.calculateSuccessRate(),
	}
}

// calculateSuccessRate calculates the current success rate
func (cb *CircuitBreaker) calculateSuccessRate() float64 {
	if cb.totalRequests == 0 {
		return 0.0
	}
	return float64(cb.successCount) / float64(cb.totalRequests)
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.halfOpenCount = 0
	
	cb.logger.Info("Circuit breaker manually reset to closed state")
}

// Metrics contains circuit breaker statistics
type Metrics struct {
	State           State     `json:"state"`
	TotalRequests   int64     `json:"total_requests"`
	SuccessCount    int64     `json:"success_count"`
	FailureCount    int64     `json:"failure_count"`
	TimeoutCount    int64     `json:"timeout_count"`
	RejectedCount   int64     `json:"rejected_count"`
	CurrentFailures int       `json:"current_failures"`
	LastFailure     time.Time `json:"last_failure"`
	SuccessRate     float64   `json:"success_rate"`
}

// CircuitBreakerError represents an error when the circuit breaker is open
type CircuitBreakerError struct {
	State   State
	Message string
}

func (e *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker %s: %s", e.State.String(), e.Message)
}

// IsCircuitBreakerError checks if an error is a circuit breaker error
func IsCircuitBreakerError(err error) bool {
	_, ok := err.(*CircuitBreakerError)
	return ok
}

// MultiCircuitBreaker manages multiple circuit breakers for different operations
type MultiCircuitBreaker struct {
	breakers map[string]*CircuitBreaker
	config   Config
	mutex    sync.RWMutex
	logger   logging.StormDBLogger
}

// NewMultiCircuitBreaker creates a multi-circuit breaker manager
func NewMultiCircuitBreaker(config Config) *MultiCircuitBreaker {
	return &MultiCircuitBreaker{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
		logger:   config.Logger.With(zap.String("component", "multi_circuit_breaker")),
	}
}

// Execute executes an operation with a named circuit breaker
func (mcb *MultiCircuitBreaker) Execute(name string, operation func() error) error {
	breaker := mcb.getOrCreateBreaker(name)
	return breaker.Execute(operation)
}

// ExecuteWithContext executes an operation with context and named circuit breaker
func (mcb *MultiCircuitBreaker) ExecuteWithContext(name string, ctx context.Context, operation func(context.Context) error) error {
	breaker := mcb.getOrCreateBreaker(name)
	return breaker.ExecuteWithContext(ctx, operation)
}

// getOrCreateBreaker gets or creates a circuit breaker for the given name
func (mcb *MultiCircuitBreaker) getOrCreateBreaker(name string) *CircuitBreaker {
	mcb.mutex.RLock()
	breaker, exists := mcb.breakers[name]
	mcb.mutex.RUnlock()

	if exists {
		return breaker
	}

	mcb.mutex.Lock()
	defer mcb.mutex.Unlock()

	// Double-check after acquiring write lock
	if breaker, exists := mcb.breakers[name]; exists {
		return breaker
	}

	// Create new circuit breaker
	breaker = NewCircuitBreaker(mcb.config)
	mcb.breakers[name] = breaker

	mcb.logger.Info("Created new circuit breaker",
		zap.String("name", name),
		zap.Int("max_failures", mcb.config.MaxFailures),
		zap.Duration("reset_timeout", mcb.config.ResetTimeout),
	)

	return breaker
}

// GetMetrics returns metrics for all circuit breakers
func (mcb *MultiCircuitBreaker) GetMetrics() map[string]Metrics {
	mcb.mutex.RLock()
	defer mcb.mutex.RUnlock()

	metrics := make(map[string]Metrics)
	for name, breaker := range mcb.breakers {
		metrics[name] = breaker.GetMetrics()
	}

	return metrics
}

// Reset resets a specific circuit breaker or all if name is empty
func (mcb *MultiCircuitBreaker) Reset(name string) error {
	mcb.mutex.RLock()
	defer mcb.mutex.RUnlock()

	if name == "" {
		// Reset all circuit breakers
		for _, breaker := range mcb.breakers {
			breaker.Reset()
		}
		mcb.logger.Info("Reset all circuit breakers")
		return nil
	}

	breaker, exists := mcb.breakers[name]
	if !exists {
		return fmt.Errorf("circuit breaker %s not found", name)
	}

	breaker.Reset()
	mcb.logger.Info("Reset circuit breaker", zap.String("name", name))
	return nil
}
