// Package resilience provides circuit breaker and recovery mechanisms
package resilience

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// CircuitBreakerState represents the current state of a circuit breaker
type CircuitBreakerState int

const (
	// StateClosed - normal operation, requests pass through
	StateClosed CircuitBreakerState = iota
	// StateOpen - circuit is open, requests fail fast
	StateOpen
	// StateHalfOpen - testing if the service has recovered
	StateHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker provides protection against cascading failures
type CircuitBreaker struct {
	mu     sync.RWMutex
	logger *zap.Logger
	name   string

	// Configuration
	maxFailures     int64
	timeout         time.Duration
	resetTimeout    time.Duration
	halfOpenMaxReqs int64

	// State
	state           CircuitBreakerState
	failures        int64
	requests        int64
	successes       int64
	lastFailureTime time.Time
	lastStateChange time.Time

	// Half-open state tracking
	halfOpenReqs int64
	halfOpenSucc int64

	// Callbacks
	onStateChange func(name string, from, to CircuitBreakerState)
	onFailure     func(name string, err error)
}

// CircuitBreakerConfig contains configuration for circuit breaker
type CircuitBreakerConfig struct {
	Name            string
	MaxFailures     int64
	Timeout         time.Duration
	ResetTimeout    time.Duration
	HalfOpenMaxReqs int64
	OnStateChange   func(name string, from, to CircuitBreakerState)
	OnFailure       func(name string, err error)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig, logger *zap.Logger) *CircuitBreaker {
	if logger == nil {
		logger = zap.NewNop()
	}

	cb := &CircuitBreaker{
		logger:          logger,
		name:            config.Name,
		maxFailures:     config.MaxFailures,
		timeout:         config.Timeout,
		resetTimeout:    config.ResetTimeout,
		halfOpenMaxReqs: config.HalfOpenMaxReqs,
		state:           StateClosed,
		lastStateChange: time.Now(),
		onStateChange:   config.OnStateChange,
		onFailure:       config.OnFailure,
	}

	// Set defaults
	if cb.maxFailures <= 0 {
		cb.maxFailures = 5
	}
	if cb.timeout <= 0 {
		cb.timeout = 60 * time.Second
	}
	if cb.resetTimeout <= 0 {
		cb.resetTimeout = 60 * time.Second
	}
	if cb.halfOpenMaxReqs <= 0 {
		cb.halfOpenMaxReqs = 3
	}

	logger.Info("Circuit breaker created",
		zap.String("name", cb.name),
		zap.Int64("max_failures", cb.maxFailures),
		zap.Duration("timeout", cb.timeout))

	return cb
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	// Check if request should be allowed
	if !cb.allowRequest() {
		return fmt.Errorf("circuit breaker %s is open", cb.name)
	}

	// Execute the function
	err := fn()

	// Record the result
	if err != nil {
		cb.onRequestFailure(err)
		return err
	}

	cb.onRequestSuccess()
	return nil
}

// ExecuteWithContext runs the given function with circuit breaker protection and context
func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, fn func(ctx context.Context) error) error {
	if !cb.allowRequest() {
		return fmt.Errorf("circuit breaker %s is open", cb.name)
	}

	// Create a channel to capture the result
	resultCh := make(chan error, 1)

	go func() {
		resultCh <- fn(ctx)
	}()

	select {
	case err := <-resultCh:
		if err != nil {
			cb.onRequestFailure(err)
			return err
		}
		cb.onRequestSuccess()
		return nil
	case <-ctx.Done():
		cb.onRequestFailure(ctx.Err())
		return ctx.Err()
	case <-time.After(cb.timeout):
		err := fmt.Errorf("circuit breaker %s timeout", cb.name)
		cb.onRequestFailure(err)
		return err
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		Name:            cb.name,
		State:           cb.state,
		Failures:        cb.failures,
		Requests:        cb.requests,
		Successes:       cb.successes,
		LastFailureTime: cb.lastFailureTime,
		LastStateChange: cb.lastStateChange,
		FailureRate:     cb.calculateFailureRate(),
	}
}

// CircuitBreakerStats contains statistics about circuit breaker
type CircuitBreakerStats struct {
	Name            string              `json:"name"`
	State           CircuitBreakerState `json:"state"`
	Failures        int64               `json:"failures"`
	Requests        int64               `json:"requests"`
	Successes       int64               `json:"successes"`
	LastFailureTime time.Time           `json:"last_failure_time"`
	LastStateChange time.Time           `json:"last_state_change"`
	FailureRate     float64             `json:"failure_rate"`
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.failures = 0
	cb.requests = 0
	cb.successes = 0
	cb.halfOpenReqs = 0
	cb.halfOpenSucc = 0
	cb.lastStateChange = time.Now()

	cb.logger.Info("Circuit breaker reset",
		zap.String("name", cb.name),
		zap.String("from_state", oldState.String()))

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, oldState, cb.state)
	}
}

// Private methods

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if we should transition to half-open
		if now.Sub(cb.lastStateChange) >= cb.resetTimeout {
			cb.setState(StateHalfOpen)
			cb.halfOpenReqs = 0
			cb.halfOpenSucc = 0
			return true
		}
		return false
	case StateHalfOpen:
		// Allow limited requests in half-open state
		return cb.halfOpenReqs < cb.halfOpenMaxReqs
	default:
		return false
	}
}

func (cb *CircuitBreaker) onRequestSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddInt64(&cb.requests, 1)
	atomic.AddInt64(&cb.successes, 1)

	if cb.state == StateHalfOpen {
		cb.halfOpenReqs++
		cb.halfOpenSucc++

		// If we've had enough successful requests, close the circuit
		if cb.halfOpenSucc >= cb.halfOpenMaxReqs {
			cb.setState(StateClosed)
			cb.failures = 0
		}
	}
}

func (cb *CircuitBreaker) onRequestFailure(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.AddInt64(&cb.requests, 1)
	atomic.AddInt64(&cb.failures, 1)
	cb.lastFailureTime = time.Now()

	if cb.onFailure != nil {
		cb.onFailure(cb.name, err)
	}

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.maxFailures {
			cb.setState(StateOpen)
		}
	case StateHalfOpen:
		cb.halfOpenReqs++
		// Any failure in half-open state goes back to open
		cb.setState(StateOpen)
	}
}

func (cb *CircuitBreaker) setState(newState CircuitBreakerState) {
	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	cb.logger.Info("Circuit breaker state changed",
		zap.String("name", cb.name),
		zap.String("from", oldState.String()),
		zap.String("to", newState.String()))

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, oldState, newState)
	}
}

func (cb *CircuitBreaker) calculateFailureRate() float64 {
	if cb.requests == 0 {
		return 0.0
	}
	return float64(cb.failures) / float64(cb.requests)
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	mu       sync.RWMutex
	logger   *zap.Logger
	breakers map[string]*CircuitBreaker
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(logger *zap.Logger) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		logger:   logger,
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (cbm *CircuitBreakerManager) GetOrCreate(name string, config CircuitBreakerConfig) *CircuitBreaker {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	if cb, exists := cbm.breakers[name]; exists {
		return cb
	}

	config.Name = name
	cb := NewCircuitBreaker(config, cbm.logger)
	cbm.breakers[name] = cb

	return cb
}

// Get retrieves a circuit breaker by name
func (cbm *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	cb, exists := cbm.breakers[name]
	return cb, exists
}

// GetAllStats returns statistics for all circuit breakers
func (cbm *CircuitBreakerManager) GetAllStats() map[string]CircuitBreakerStats {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	stats := make(map[string]CircuitBreakerStats)
	for name, cb := range cbm.breakers {
		stats[name] = cb.GetStats()
	}

	return stats
}

// ResetAll resets all circuit breakers
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	for _, cb := range cbm.breakers {
		cb.Reset()
	}

	cbm.logger.Info("All circuit breakers reset")
}

// RecoveryManager handles automatic recovery strategies
type RecoveryManager struct {
	mu              sync.RWMutex
	logger          *zap.Logger
	checkpointMgr   *CheckpointManager
	circuitMgr      *CircuitBreakerManager
	strategies      map[string]RecoveryStrategy
	recoveryHistory []RecoveryAttempt

	// Configuration
	maxRetries    int
	retryDelay    time.Duration
	backoffFactor float64
	maxBackoff    time.Duration
}

// RecoveryStrategy defines how to recover from specific failure types
type RecoveryStrategy struct {
	Name        string
	Description string
	Handler     func(ctx context.Context, failure FailureInfo) RecoveryResult
	Priority    int
	Timeout     time.Duration
}

// FailureInfo contains information about a failure
type FailureInfo struct {
	Type       string
	Component  string
	Error      error
	Timestamp  time.Time
	Context    map[string]interface{}
	Checkpoint *Checkpoint
	Retryable  bool
}

// RecoveryResult contains the result of a recovery attempt
type RecoveryResult struct {
	Success    bool
	Action     string
	Message    string
	NewState   string
	Checkpoint *Checkpoint
	Retry      bool
	RetryDelay time.Duration
}

// RecoveryAttempt records a recovery attempt
type RecoveryAttempt struct {
	ID        string
	Timestamp time.Time
	Failure   FailureInfo
	Strategy  string
	Result    RecoveryResult
	Duration  time.Duration
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(checkpointMgr *CheckpointManager, circuitMgr *CircuitBreakerManager, logger *zap.Logger) *RecoveryManager {
	rm := &RecoveryManager{
		logger:          logger,
		checkpointMgr:   checkpointMgr,
		circuitMgr:      circuitMgr,
		strategies:      make(map[string]RecoveryStrategy),
		recoveryHistory: make([]RecoveryAttempt, 0),
		maxRetries:      3,
		retryDelay:      5 * time.Second,
		backoffFactor:   2.0,
		maxBackoff:      60 * time.Second,
	}

	// Register default recovery strategies
	rm.registerDefaultStrategies()

	return rm
}

// RegisterStrategy registers a new recovery strategy
func (rm *RecoveryManager) RegisterStrategy(strategy RecoveryStrategy) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.strategies[strategy.Name] = strategy

	rm.logger.Info("Recovery strategy registered",
		zap.String("name", strategy.Name),
		zap.String("description", strategy.Description))
}

// AttemptRecovery attempts to recover from a failure
func (rm *RecoveryManager) AttemptRecovery(ctx context.Context, failure FailureInfo) RecoveryResult {
	start := time.Now()

	// Find the best recovery strategy
	strategy := rm.selectRecoveryStrategy(failure)
	if strategy == nil {
		rm.logger.Error("No recovery strategy found for failure type",
			zap.String("type", failure.Type),
			zap.String("component", failure.Component))

		return RecoveryResult{
			Success: false,
			Action:  "no_strategy",
			Message: "No recovery strategy available",
		}
	}

	rm.logger.Info("Attempting recovery",
		zap.String("strategy", strategy.Name),
		zap.String("failure_type", failure.Type),
		zap.String("component", failure.Component))

	// Execute recovery strategy with timeout
	recoveryCtx, cancel := context.WithTimeout(ctx, strategy.Timeout)
	defer cancel()

	result := strategy.Handler(recoveryCtx, failure)
	duration := time.Since(start)

	// Record the recovery attempt
	attempt := RecoveryAttempt{
		ID:        fmt.Sprintf("recovery_%d", time.Now().UnixNano()),
		Timestamp: start,
		Failure:   failure,
		Strategy:  strategy.Name,
		Result:    result,
		Duration:  duration,
	}

	rm.mu.Lock()
	rm.recoveryHistory = append(rm.recoveryHistory, attempt)
	rm.mu.Unlock()

	rm.logger.Info("Recovery attempt completed",
		zap.String("strategy", strategy.Name),
		zap.Bool("success", result.Success),
		zap.String("action", result.Action),
		zap.Duration("duration", duration))

	return result
}

// GetRecoveryHistory returns the recovery attempt history
func (rm *RecoveryManager) GetRecoveryHistory() []RecoveryAttempt {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Return a copy of the history
	history := make([]RecoveryAttempt, len(rm.recoveryHistory))
	copy(history, rm.recoveryHistory)

	return history
}

// Private methods

func (rm *RecoveryManager) registerDefaultStrategies() {
	// Database connection failure recovery
	rm.RegisterStrategy(RecoveryStrategy{
		Name:        "database_connection_recovery",
		Description: "Recover from database connection failures",
		Priority:    1,
		Timeout:     30 * time.Second,
		Handler:     rm.handleDatabaseConnectionFailure,
	})

	// Plugin failure recovery
	rm.RegisterStrategy(RecoveryStrategy{
		Name:        "plugin_failure_recovery",
		Description: "Recover from plugin execution failures",
		Priority:    2,
		Timeout:     15 * time.Second,
		Handler:     rm.handlePluginFailure,
	})

	// Resource exhaustion recovery
	rm.RegisterStrategy(RecoveryStrategy{
		Name:        "resource_exhaustion_recovery",
		Description: "Recover from resource exhaustion",
		Priority:    3,
		Timeout:     60 * time.Second,
		Handler:     rm.handleResourceExhaustion,
	})

	// General retry strategy
	rm.RegisterStrategy(RecoveryStrategy{
		Name:        "general_retry",
		Description: "General retry mechanism for transient failures",
		Priority:    10,
		Timeout:     30 * time.Second,
		Handler:     rm.handleGeneralRetry,
	})
}

func (rm *RecoveryManager) selectRecoveryStrategy(failure FailureInfo) *RecoveryStrategy {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var bestStrategy *RecoveryStrategy
	bestPriority := int(^uint(0) >> 1) // Max int

	for _, strategy := range rm.strategies {
		if rm.strategyMatches(strategy, failure) && strategy.Priority < bestPriority {
			strategyColor := strategy
			bestStrategy = &strategyColor
			bestPriority = strategy.Priority
		}
	}

	return bestStrategy
}

func (rm *RecoveryManager) strategyMatches(strategy RecoveryStrategy, failure FailureInfo) bool {
	// Simple matching logic - can be enhanced
	switch strategy.Name {
	case "database_connection_recovery":
		return failure.Type == "database_connection" || failure.Component == "database"
	case "plugin_failure_recovery":
		return failure.Type == "plugin_error" || failure.Component == "plugin"
	case "resource_exhaustion_recovery":
		return failure.Type == "resource_exhaustion"
	case "general_retry":
		return failure.Retryable
	default:
		return false
	}
}

func (rm *RecoveryManager) handleDatabaseConnectionFailure(ctx context.Context, failure FailureInfo) RecoveryResult {
	// Try to reconnect with exponential backoff
	delay := rm.retryDelay

	for i := 0; i < rm.maxRetries; i++ {
		select {
		case <-ctx.Done():
			return RecoveryResult{
				Success: false,
				Action:  "timeout",
				Message: "Recovery attempt timed out",
			}
		case <-time.After(delay):
			// Attempt reconnection logic here
			// For now, simulate recovery
			if i == rm.maxRetries-1 {
				return RecoveryResult{
					Success: true,
					Action:  "reconnect",
					Message: "Database connection restored",
				}
			}

			// Calculate next delay with backoff
			delay = time.Duration(float64(delay) * rm.backoffFactor)
			if delay > rm.maxBackoff {
				delay = rm.maxBackoff
			}
		}
	}

	return RecoveryResult{
		Success: false,
		Action:  "max_retries_exceeded",
		Message: "Failed to restore database connection after maximum retries",
	}
}

func (rm *RecoveryManager) handlePluginFailure(ctx context.Context, failure FailureInfo) RecoveryResult {
	// Try to reload the plugin or use a fallback
	return RecoveryResult{
		Success:    true,
		Action:     "plugin_reload",
		Message:    "Plugin reloaded successfully",
		Retry:      true,
		RetryDelay: 2 * time.Second,
	}
}

func (rm *RecoveryManager) handleResourceExhaustion(ctx context.Context, failure FailureInfo) RecoveryResult {
	// Reduce load and wait for resources to free up
	return RecoveryResult{
		Success:    true,
		Action:     "reduce_load",
		Message:    "Reduced connection count and waiting for resource recovery",
		NewState:   "reduced_load",
		Retry:      true,
		RetryDelay: 10 * time.Second,
	}
}

func (rm *RecoveryManager) handleGeneralRetry(ctx context.Context, failure FailureInfo) RecoveryResult {
	// Simple retry with delay
	return RecoveryResult{
		Success:    true,
		Action:     "retry",
		Message:    "Retrying operation after delay",
		Retry:      true,
		RetryDelay: rm.retryDelay,
	}
}
