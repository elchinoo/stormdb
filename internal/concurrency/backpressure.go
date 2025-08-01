// Package concurrency provides advanced concurrency control with backpressure and monitoring
package concurrency

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// BackpressureController manages backpressure for connection pools and workers
type BackpressureController struct {
	mu     sync.RWMutex
	logger *zap.Logger

	// Configuration
	maxConnections    int64
	maxWorkers        int64
	maxQueueSize      int64
	targetLatency     time.Duration
	maxLatency        time.Duration
	pressureThreshold float64

	// Current state
	activeConnections int64
	activeWorkers     int64
	queuedRequests    int64
	currentLatency    time.Duration
	pressure          float64

	// Monitoring
	metrics         *ConcurrencyMetrics
	lastAdjustment  time.Time
	adjustmentDelay time.Duration

	// Adaptive scaling
	scalingHistory     []ScalingEvent
	autoScale          bool
	scaleUpThreshold   float64
	scaleDownThreshold float64

	// Callbacks
	onPressureChange func(pressure float64)
	onScalingEvent   func(event ScalingEvent)
}

// ConcurrencyMetrics tracks concurrency-related metrics
type ConcurrencyMetrics struct {
	mu sync.RWMutex

	// Connection metrics
	TotalConnections     int64 `json:"total_connections"`
	ActiveConnections    int64 `json:"active_connections"`
	PeakConnections      int64 `json:"peak_connections"`
	ConnectionsCreated   int64 `json:"connections_created"`
	ConnectionsDestroyed int64 `json:"connections_destroyed"`
	ConnectionErrors     int64 `json:"connection_errors"`

	// Worker metrics
	TotalWorkers      int64 `json:"total_workers"`
	ActiveWorkers     int64 `json:"active_workers"`
	PeakWorkers       int64 `json:"peak_workers"`
	WorkersSpawned    int64 `json:"workers_spawned"`
	WorkersTerminated int64 `json:"workers_terminated"`

	// Queue metrics
	QueueSize      int64 `json:"queue_size"`
	PeakQueueSize  int64 `json:"peak_queue_size"`
	QueuedItems    int64 `json:"queued_items"`
	ProcessedItems int64 `json:"processed_items"`
	DroppedItems   int64 `json:"dropped_items"`

	// Performance metrics
	AverageLatency time.Duration `json:"average_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	P99Latency     time.Duration `json:"p99_latency"`
	Throughput     float64       `json:"throughput"`

	// Pressure metrics
	CurrentPressure float64 `json:"current_pressure"`
	PeakPressure    float64 `json:"peak_pressure"`
	PressureEvents  int64   `json:"pressure_events"`

	// System metrics
	GoroutineCount  int     `json:"goroutine_count"`
	MemoryUsageMB   float64 `json:"memory_usage_mb"`
	CPUUsagePercent float64 `json:"cpu_usage_percent"`

	// Timestamps
	StartTime  time.Time `json:"start_time"`
	LastUpdate time.Time `json:"last_update"`
}

// ScalingEvent represents a scaling action
type ScalingEvent struct {
	Timestamp time.Time     `json:"timestamp"`
	Type      string        `json:"type"`      // "scale_up", "scale_down", "pressure_relief"
	Component string        `json:"component"` // "connections", "workers", "queue"
	Before    int64         `json:"before"`
	After     int64         `json:"after"`
	Reason    string        `json:"reason"`
	Pressure  float64       `json:"pressure"`
	Latency   time.Duration `json:"latency"`
}

// BackpressureConfig contains configuration for backpressure controller
type BackpressureConfig struct {
	MaxConnections     int64
	MaxWorkers         int64
	MaxQueueSize       int64
	TargetLatency      time.Duration
	MaxLatency         time.Duration
	PressureThreshold  float64
	AutoScale          bool
	ScaleUpThreshold   float64
	ScaleDownThreshold float64
	AdjustmentDelay    time.Duration
	OnPressureChange   func(pressure float64)
	OnScalingEvent     func(event ScalingEvent)
}

// NewBackpressureController creates a new backpressure controller
func NewBackpressureController(config BackpressureConfig, logger *zap.Logger) *BackpressureController {
	if logger == nil {
		logger = zap.NewNop()
	}

	bc := &BackpressureController{
		logger:             logger,
		maxConnections:     config.MaxConnections,
		maxWorkers:         config.MaxWorkers,
		maxQueueSize:       config.MaxQueueSize,
		targetLatency:      config.TargetLatency,
		maxLatency:         config.MaxLatency,
		pressureThreshold:  config.PressureThreshold,
		autoScale:          config.AutoScale,
		scaleUpThreshold:   config.ScaleUpThreshold,
		scaleDownThreshold: config.ScaleDownThreshold,
		adjustmentDelay:    config.AdjustmentDelay,
		onPressureChange:   config.OnPressureChange,
		onScalingEvent:     config.OnScalingEvent,
		scalingHistory:     make([]ScalingEvent, 0),
		metrics:            &ConcurrencyMetrics{StartTime: time.Now()},
	}

	// Set defaults
	if bc.maxConnections <= 0 {
		bc.maxConnections = 100
	}
	if bc.maxWorkers <= 0 {
		bc.maxWorkers = int64(runtime.NumCPU() * 2)
	}
	if bc.maxQueueSize <= 0 {
		bc.maxQueueSize = 1000
	}
	if bc.targetLatency <= 0 {
		bc.targetLatency = 100 * time.Millisecond
	}
	if bc.maxLatency <= 0 {
		bc.maxLatency = 1 * time.Second
	}
	if bc.pressureThreshold <= 0 {
		bc.pressureThreshold = 0.8
	}
	if bc.scaleUpThreshold <= 0 {
		bc.scaleUpThreshold = 0.7
	}
	if bc.scaleDownThreshold <= 0 {
		bc.scaleDownThreshold = 0.3
	}
	if bc.adjustmentDelay <= 0 {
		bc.adjustmentDelay = 5 * time.Second
	}

	logger.Info("Backpressure controller created",
		zap.Int64("max_connections", bc.maxConnections),
		zap.Int64("max_workers", bc.maxWorkers),
		zap.Int64("max_queue_size", bc.maxQueueSize),
		zap.Bool("auto_scale", bc.autoScale))

	return bc
}

// AcquireConnection attempts to acquire a connection slot
func (bc *BackpressureController) AcquireConnection() bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Check if we're under pressure
	if bc.pressure > bc.pressureThreshold {
		atomic.AddInt64(&bc.metrics.DroppedItems, 1)
		return false
	}

	// Check connection limit
	if atomic.LoadInt64(&bc.activeConnections) >= bc.maxConnections {
		atomic.AddInt64(&bc.metrics.DroppedItems, 1)
		return false
	}

	// Acquire connection
	atomic.AddInt64(&bc.activeConnections, 1)
	atomic.AddInt64(&bc.metrics.TotalConnections, 1)
	atomic.AddInt64(&bc.metrics.ConnectionsCreated, 1)

	// Update peak connections
	current := atomic.LoadInt64(&bc.activeConnections)
	if current > bc.metrics.PeakConnections {
		bc.metrics.PeakConnections = current
	}

	bc.updatePressure()
	return true
}

// ReleaseConnection releases a connection slot
func (bc *BackpressureController) ReleaseConnection() {
	atomic.AddInt64(&bc.activeConnections, -1)
	atomic.AddInt64(&bc.metrics.ConnectionsDestroyed, 1)

	bc.mu.Lock()
	bc.updatePressure()
	bc.mu.Unlock()
}

// AcquireWorker attempts to acquire a worker slot
func (bc *BackpressureController) AcquireWorker() bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Check worker limit
	if atomic.LoadInt64(&bc.activeWorkers) >= bc.maxWorkers {
		return false
	}

	// Acquire worker
	atomic.AddInt64(&bc.activeWorkers, 1)
	atomic.AddInt64(&bc.metrics.TotalWorkers, 1)
	atomic.AddInt64(&bc.metrics.WorkersSpawned, 1)

	// Update peak workers
	current := atomic.LoadInt64(&bc.activeWorkers)
	if current > bc.metrics.PeakWorkers {
		bc.metrics.PeakWorkers = current
	}

	bc.updatePressure()
	return true
}

// ReleaseWorker releases a worker slot
func (bc *BackpressureController) ReleaseWorker() {
	atomic.AddInt64(&bc.activeWorkers, -1)
	atomic.AddInt64(&bc.metrics.WorkersTerminated, 1)

	bc.mu.Lock()
	bc.updatePressure()
	bc.mu.Unlock()
}

// EnqueueRequest attempts to enqueue a request
func (bc *BackpressureController) EnqueueRequest() bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Check queue limit
	if atomic.LoadInt64(&bc.queuedRequests) >= bc.maxQueueSize {
		atomic.AddInt64(&bc.metrics.DroppedItems, 1)
		return false
	}

	// Enqueue request
	atomic.AddInt64(&bc.queuedRequests, 1)
	atomic.AddInt64(&bc.metrics.QueuedItems, 1)

	// Update peak queue size
	current := atomic.LoadInt64(&bc.queuedRequests)
	bc.metrics.QueueSize = current
	if current > bc.metrics.PeakQueueSize {
		bc.metrics.PeakQueueSize = current
	}

	bc.updatePressure()
	return true
}

// DequeueRequest removes a request from the queue
func (bc *BackpressureController) DequeueRequest() {
	atomic.AddInt64(&bc.queuedRequests, -1)
	atomic.AddInt64(&bc.metrics.ProcessedItems, 1)

	bc.mu.Lock()
	bc.metrics.QueueSize = atomic.LoadInt64(&bc.queuedRequests)
	bc.updatePressure()
	bc.mu.Unlock()
}

// UpdateLatency updates the current latency measurement
func (bc *BackpressureController) UpdateLatency(latency time.Duration) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.currentLatency = latency

	// Update metrics
	if bc.metrics.AverageLatency == 0 {
		bc.metrics.AverageLatency = latency
	} else {
		// Simple moving average
		bc.metrics.AverageLatency = (bc.metrics.AverageLatency + latency) / 2
	}

	// Update percentiles (simplified)
	if latency > bc.metrics.P95Latency {
		bc.metrics.P95Latency = latency
	}
	if latency > bc.metrics.P99Latency {
		bc.metrics.P99Latency = latency
	}

	bc.updatePressure()
	bc.considerScaling()
}

// GetPressure returns the current pressure level (0.0 to 1.0)
func (bc *BackpressureController) GetPressure() float64 {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.pressure
}

// GetMetrics returns current concurrency metrics
func (bc *BackpressureController) GetMetrics() ConcurrencyMetrics {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	// Update system metrics
	bc.metrics.GoroutineCount = runtime.NumGoroutine()
	bc.metrics.ActiveConnections = atomic.LoadInt64(&bc.activeConnections)
	bc.metrics.ActiveWorkers = atomic.LoadInt64(&bc.activeWorkers)
	bc.metrics.CurrentPressure = bc.pressure
	bc.metrics.LastUpdate = time.Now()

	return *bc.metrics
}

// GetScalingHistory returns the scaling event history
func (bc *BackpressureController) GetScalingHistory() []ScalingEvent {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	// Return a copy
	history := make([]ScalingEvent, len(bc.scalingHistory))
	copy(history, bc.scalingHistory)
	return history
}

// Private methods

func (bc *BackpressureController) updatePressure() {
	// Calculate pressure based on multiple factors
	connPressure := float64(atomic.LoadInt64(&bc.activeConnections)) / float64(bc.maxConnections)
	workerPressure := float64(atomic.LoadInt64(&bc.activeWorkers)) / float64(bc.maxWorkers)
	queuePressure := float64(atomic.LoadInt64(&bc.queuedRequests)) / float64(bc.maxQueueSize)

	latencyPressure := 0.0
	if bc.maxLatency > 0 {
		latencyPressure = float64(bc.currentLatency) / float64(bc.maxLatency)
		if latencyPressure > 1.0 {
			latencyPressure = 1.0
		}
	}

	// Weighted combination of pressures
	bc.pressure = (connPressure*0.3 + workerPressure*0.2 + queuePressure*0.3 + latencyPressure*0.2)

	// Update peak pressure
	if bc.pressure > bc.metrics.PeakPressure {
		bc.metrics.PeakPressure = bc.pressure
	}

	// Trigger pressure change callback
	if bc.onPressureChange != nil {
		bc.onPressureChange(bc.pressure)
	}

	// Record pressure events
	if bc.pressure > bc.pressureThreshold {
		atomic.AddInt64(&bc.metrics.PressureEvents, 1)
	}
}

func (bc *BackpressureController) considerScaling() {
	if !bc.autoScale {
		return
	}

	now := time.Now()
	if now.Sub(bc.lastAdjustment) < bc.adjustmentDelay {
		return
	}

	// Scale up if pressure is high
	if bc.pressure > bc.scaleUpThreshold {
		bc.scaleUp()
		bc.lastAdjustment = now
	} else if bc.pressure < bc.scaleDownThreshold {
		bc.scaleDown()
		bc.lastAdjustment = now
	}
}

func (bc *BackpressureController) scaleUp() {
	event := ScalingEvent{
		Timestamp: time.Now(),
		Type:      "scale_up",
		Pressure:  bc.pressure,
		Latency:   bc.currentLatency,
		Reason:    "high_pressure",
	}

	// Scale connections if needed
	if atomic.LoadInt64(&bc.activeConnections) > int64(float64(bc.maxConnections)*0.8) {
		oldMax := bc.maxConnections
		bc.maxConnections = int64(float64(bc.maxConnections) * 1.2)

		event.Component = "connections"
		event.Before = oldMax
		event.After = bc.maxConnections

		bc.logger.Info("Scaled up connections",
			zap.Int64("from", oldMax),
			zap.Int64("to", bc.maxConnections),
			zap.Float64("pressure", bc.pressure))
	}

	// Scale workers if needed
	if atomic.LoadInt64(&bc.activeWorkers) > int64(float64(bc.maxWorkers)*0.8) {
		oldMax := bc.maxWorkers
		bc.maxWorkers = int64(float64(bc.maxWorkers) * 1.2)

		event.Component = "workers"
		event.Before = oldMax
		event.After = bc.maxWorkers

		bc.logger.Info("Scaled up workers",
			zap.Int64("from", oldMax),
			zap.Int64("to", bc.maxWorkers),
			zap.Float64("pressure", bc.pressure))
	}

	bc.scalingHistory = append(bc.scalingHistory, event)

	if bc.onScalingEvent != nil {
		bc.onScalingEvent(event)
	}
}

func (bc *BackpressureController) scaleDown() {
	event := ScalingEvent{
		Timestamp: time.Now(),
		Type:      "scale_down",
		Pressure:  bc.pressure,
		Latency:   bc.currentLatency,
		Reason:    "low_pressure",
	}

	// Scale down connections if we have excess capacity
	if atomic.LoadInt64(&bc.activeConnections) < int64(float64(bc.maxConnections)*0.3) && bc.maxConnections > 10 {
		oldMax := bc.maxConnections
		bc.maxConnections = int64(float64(bc.maxConnections) * 0.8)

		event.Component = "connections"
		event.Before = oldMax
		event.After = bc.maxConnections

		bc.logger.Info("Scaled down connections",
			zap.Int64("from", oldMax),
			zap.Int64("to", bc.maxConnections),
			zap.Float64("pressure", bc.pressure))
	}

	bc.scalingHistory = append(bc.scalingHistory, event)

	if bc.onScalingEvent != nil {
		bc.onScalingEvent(event)
	}
}

// ConnectionPool manages a pool of connections with backpressure
type ConnectionPool struct {
	mu           sync.RWMutex
	logger       *zap.Logger
	backpressure *BackpressureController

	// Pool configuration
	minSize     int
	maxSize     int
	idleTimeout time.Duration

	// Pool state
	connections []PooledConnection
	available   chan PooledConnection
	metrics     *PoolMetrics

	// Lifecycle
	ctx     context.Context
	cancel  context.CancelFunc
	cleanup time.Ticker
}

// PooledConnection represents a connection in the pool
type PooledConnection struct {
	ID         string
	Connection interface{}
	CreatedAt  time.Time
	LastUsed   time.Time
	UsageCount int64
	IsHealthy  bool
}

// PoolMetrics tracks connection pool metrics
type PoolMetrics struct {
	mu sync.RWMutex

	TotalConnections     int64         `json:"total_connections"`
	AvailableConnections int64         `json:"available_connections"`
	BusyConnections      int64         `json:"busy_connections"`
	ConnectionsCreated   int64         `json:"connections_created"`
	ConnectionsDestroyed int64         `json:"connections_destroyed"`
	AcquisitionTime      time.Duration `json:"acquisition_time"`
	PoolHits             int64         `json:"pool_hits"`
	PoolMisses           int64         `json:"pool_misses"`
	HealthCheckFails     int64         `json:"health_check_fails"`
}

// ConnectionFactory creates new connections
type ConnectionFactory func() (interface{}, error)

// ConnectionValidator validates connection health
type ConnectionValidator func(interface{}) bool

// NewConnectionPool creates a new connection pool
func NewConnectionPool(
	factory ConnectionFactory,
	validator ConnectionValidator,
	backpressure *BackpressureController,
	logger *zap.Logger,
) *ConnectionPool {

	ctx, cancel := context.WithCancel(context.Background())

	pool := &ConnectionPool{
		logger:       logger,
		backpressure: backpressure,
		minSize:      5,
		maxSize:      50,
		idleTimeout:  5 * time.Minute,
		connections:  make([]PooledConnection, 0),
		available:    make(chan PooledConnection, 50),
		metrics:      &PoolMetrics{},
		ctx:          ctx,
		cancel:       cancel,
		cleanup:      *time.NewTicker(1 * time.Minute),
	}

	// Start cleanup routine
	go pool.cleanupRoutine()

	return pool
}

// AcquireConnection gets a connection from the pool
func (cp *ConnectionPool) AcquireConnection() (*PooledConnection, error) {
	start := time.Now()

	// Check backpressure
	if !cp.backpressure.AcquireConnection() {
		return nil, fmt.Errorf("connection acquisition rejected due to backpressure")
	}

	// Try to get from available pool
	select {
	case conn := <-cp.available:
		atomic.AddInt64(&cp.metrics.PoolHits, 1)
		conn.LastUsed = time.Now()
		atomic.AddInt64(&conn.UsageCount, 1)
		cp.metrics.AcquisitionTime = time.Since(start)
		return &conn, nil
	default:
		// Pool is empty, create new connection if possible
		atomic.AddInt64(&cp.metrics.PoolMisses, 1)
		// Implementation would create new connection here
		cp.metrics.AcquisitionTime = time.Since(start)
		return nil, fmt.Errorf("pool exhausted")
	}
}

// ReleaseConnection returns a connection to the pool
func (cp *ConnectionPool) ReleaseConnection(conn *PooledConnection) {
	cp.backpressure.ReleaseConnection()

	conn.LastUsed = time.Now()

	select {
	case cp.available <- *conn:
		// Successfully returned to pool
	default:
		// Pool is full, destroy connection
		atomic.AddInt64(&cp.metrics.ConnectionsDestroyed, 1)
	}
}

// GetMetrics returns pool metrics
func (cp *ConnectionPool) GetMetrics() PoolMetrics {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	cp.metrics.TotalConnections = int64(len(cp.connections))
	cp.metrics.AvailableConnections = int64(len(cp.available))
	cp.metrics.BusyConnections = cp.metrics.TotalConnections - cp.metrics.AvailableConnections

	return *cp.metrics
}

// Close closes the connection pool
func (cp *ConnectionPool) Close() {
	cp.cancel()
	cp.cleanup.Stop()

	// Close all connections
	close(cp.available)
	for conn := range cp.available {
		// Close connection implementation
		_ = conn
	}

	cp.logger.Info("Connection pool closed")
}

// Private methods

func (cp *ConnectionPool) cleanupRoutine() {
	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-cp.cleanup.C:
			cp.cleanupIdleConnections()
		}
	}
}

func (cp *ConnectionPool) cleanupIdleConnections() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	now := time.Now()
	cleaned := 0

	// Check available connections for idle timeout
	select {
	case conn := <-cp.available:
		if now.Sub(conn.LastUsed) > cp.idleTimeout {
			// Connection is idle, destroy it
			atomic.AddInt64(&cp.metrics.ConnectionsDestroyed, 1)
			cleaned++
		} else {
			// Put it back
			cp.available <- conn
		}
	default:
		// No connections to check
	}

	if cleaned > 0 {
		cp.logger.Debug("Cleaned up idle connections", zap.Int("count", cleaned))
	}
}
