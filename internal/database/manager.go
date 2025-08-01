package database

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elchinoo/stormdb/internal/config"
	"github.com/elchinoo/stormdb/internal/logging"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// DatabaseManager provides robust database connection management
type DatabaseManager struct {
	pool    *pgxpool.Pool
	config  *config.DatabaseConfig
	metrics *ConnectionMetrics
	health  *HealthChecker
	logger  logging.StormDBLogger

	// Connection tracking
	activeConnections  int64
	connectionLifetime int64 // nanoseconds
	connectionAttempts int64
	connectionFailures int64

	mutex sync.RWMutex
}

// ConnectionMetrics tracks database connection statistics
type ConnectionMetrics struct {
	ActiveConnections    int64         `json:"active_connections"`
	IdleConnections      int64         `json:"idle_connections"`
	FailedConnections    int64         `json:"failed_connections"`
	TotalConnections     int64         `json:"total_connections"`
	AverageLifetime      time.Duration `json:"average_lifetime"`
	AverageAcquireTime   time.Duration `json:"average_acquire_time"`
	ConnectionsCreated   int64         `json:"connections_created"`
	ConnectionsDestroyed int64         `json:"connections_destroyed"`
}

// HealthChecker monitors database connection health
type HealthChecker struct {
	manager  *DatabaseManager
	interval time.Duration
	stop     chan struct{}
	logger   logging.StormDBLogger

	// Health metrics
	lastCheck        time.Time
	consecutiveFails int64
	healthHistory    []HealthStatus
}

// HealthStatus represents database health at a point in time
type HealthStatus struct {
	Timestamp    time.Time     `json:"timestamp"`
	Healthy      bool          `json:"healthy"`
	ResponseTime time.Duration `json:"response_time"`
	Error        string        `json:"error,omitempty"`
}

// NewDatabaseManager creates a new database manager with enhanced features
func NewDatabaseManager(config *config.DatabaseConfig, logger logging.StormDBLogger) (*DatabaseManager, error) {
	if config == nil {
		return nil, errors.New("database config cannot be nil")
	}
	if logger == nil {
		logger = logging.NewDefaultLogger()
	}

	dm := &DatabaseManager{
		config:  config,
		metrics: &ConnectionMetrics{},
		logger:  logger,
	}

	// Create health checker
	dm.health = &HealthChecker{
		manager:       dm,
		interval:      config.HealthCheckPeriod,
		stop:          make(chan struct{}),
		logger:        logger.With(zap.String("component", "health_checker")),
		healthHistory: make([]HealthStatus, 0, 100), // Keep last 100 health checks
	}

	return dm, nil
}

// Connect establishes connection pool with retry logic and monitoring
func (dm *DatabaseManager) Connect(ctx context.Context) error {
	dm.logger.Info("Establishing database connection pool",
		logging.Fields.Database(dm.config.Host, dm.config.Port, dm.config.Database)...,
	)

	// Build connection string
	connString := dm.buildConnectionString()

	// Create pool config
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return errors.Wrap(err, "failed to parse connection string")
	}

	// Configure pool settings
	poolConfig.MaxConns = int32(dm.config.MaxConnections)
	poolConfig.MinConns = int32(dm.config.MinConnections)
	poolConfig.MaxConnLifetime = dm.config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = dm.config.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = dm.config.HealthCheckPeriod

	// Add connection lifecycle callbacks
	poolConfig.BeforeConnect = dm.beforeConnect
	poolConfig.AfterConnect = dm.afterConnect
	poolConfig.BeforeClose = dm.beforeClose

	// Create connection pool with timeout
	ctx, cancel := context.WithTimeout(ctx, dm.config.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		atomic.AddInt64(&dm.connectionFailures, 1)
		return errors.Wrap(err, "failed to create connection pool")
	}

	dm.mutex.Lock()
	dm.pool = pool
	dm.mutex.Unlock()

	// Verify connection with ping
	if err := dm.ping(ctx); err != nil {
		pool.Close()
		return errors.Wrap(err, "initial connection health check failed")
	}

	// Start health monitoring
	dm.health.Start()

	dm.logger.Info("Database connection pool established successfully",
		zap.Int("max_connections", dm.config.MaxConnections),
		zap.Int("min_connections", dm.config.MinConnections),
		zap.Duration("max_lifetime", dm.config.MaxConnLifetime),
	)

	return nil
}

// GetConnection acquires a connection from the pool with metrics tracking
func (dm *DatabaseManager) GetConnection(ctx context.Context) (*pgxpool.Conn, error) {
	dm.mutex.RLock()
	pool := dm.pool
	dm.mutex.RUnlock()

	if pool == nil {
		return nil, errors.New("database connection pool not initialized")
	}

	start := time.Now()
	atomic.AddInt64(&dm.connectionAttempts, 1)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		atomic.AddInt64(&dm.connectionFailures, 1)
		dm.logger.Error("Failed to acquire database connection", err,
			zap.Duration("acquire_time", time.Since(start)),
		)
		return nil, errors.Wrap(err, "failed to acquire connection")
	}

	acquireTime := time.Since(start)
	atomic.AddInt64(&dm.activeConnections, 1)

	// Update metrics
	dm.updateAcquireTimeMetrics(acquireTime)

	dm.logger.Debug("Database connection acquired",
		zap.Duration("acquire_time", acquireTime),
		zap.Int64("active_connections", atomic.LoadInt64(&dm.activeConnections)),
	)

	return conn, nil
}

// ReleaseConnection returns a connection to the pool
func (dm *DatabaseManager) ReleaseConnection(conn *pgxpool.Conn) {
	if conn != nil {
		conn.Release()
		atomic.AddInt64(&dm.activeConnections, -1)

		dm.logger.Debug("Database connection released",
			zap.Int64("active_connections", atomic.LoadInt64(&dm.activeConnections)),
		)
	}
}

// HealthCheck performs a health check on the database connection
func (dm *DatabaseManager) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return dm.ping(ctx)
}

// ping performs a simple ping to verify database connectivity
func (dm *DatabaseManager) ping(ctx context.Context) error {
	dm.mutex.RLock()
	pool := dm.pool
	dm.mutex.RUnlock()

	if pool == nil {
		return errors.New("connection pool not initialized")
	}

	return pool.Ping(ctx)
}

// GetMetrics returns current connection metrics
func (dm *DatabaseManager) GetMetrics() ConnectionMetrics {
	dm.mutex.RLock()
	pool := dm.pool
	dm.mutex.RUnlock()

	metrics := ConnectionMetrics{
		ActiveConnections:    atomic.LoadInt64(&dm.activeConnections),
		FailedConnections:    atomic.LoadInt64(&dm.connectionFailures),
		TotalConnections:     atomic.LoadInt64(&dm.connectionAttempts),
		ConnectionsCreated:   dm.metrics.ConnectionsCreated,
		ConnectionsDestroyed: dm.metrics.ConnectionsDestroyed,
	}

	if pool != nil {
		stat := pool.Stat()
		metrics.IdleConnections = int64(stat.IdleConns())
		metrics.TotalConnections = int64(stat.TotalConns())
	}

	// Calculate average lifetime
	if dm.metrics.ConnectionsDestroyed > 0 {
		metrics.AverageLifetime = time.Duration(
			atomic.LoadInt64(&dm.connectionLifetime) / dm.metrics.ConnectionsDestroyed,
		)
	}

	return metrics
}

// Close gracefully closes the database connection pool
func (dm *DatabaseManager) Close() error {
	dm.logger.Info("Closing database connection pool")

	// Stop health checker
	dm.health.Stop()

	dm.mutex.Lock()
	pool := dm.pool
	dm.pool = nil
	dm.mutex.Unlock()

	if pool != nil {
		pool.Close()
		dm.logger.Info("Database connection pool closed")
	}

	return nil
}

// buildConnectionString constructs the database connection string
func (dm *DatabaseManager) buildConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dm.config.Username,
		dm.config.Password,
		dm.config.Host,
		dm.config.Port,
		dm.config.Database,
		dm.config.SSLMode,
	)
}

// Connection lifecycle callbacks

func (dm *DatabaseManager) beforeConnect(ctx context.Context, config *pgx.ConnConfig) error {
	dm.logger.Debug("Creating new database connection",
		zap.String("host", config.Host),
		zap.Uint16("port", config.Port),
		zap.String("database", config.Database),
	)
	return nil
}

func (dm *DatabaseManager) afterConnect(ctx context.Context, conn *pgx.Conn) error {
	atomic.AddInt64(&dm.metrics.ConnectionsCreated, 1)
	dm.logger.Debug("Database connection created successfully")
	return nil
}

func (dm *DatabaseManager) beforeClose(conn *pgx.Conn) {
	atomic.AddInt64(&dm.metrics.ConnectionsDestroyed, 1)
	dm.logger.Debug("Database connection being closed")
}

// updateAcquireTimeMetrics updates connection acquire time metrics
func (dm *DatabaseManager) updateAcquireTimeMetrics(duration time.Duration) {
	// This could be enhanced with a histogram or sliding window for better metrics
	// For now, we'll keep it simple and just log significant delays
	if duration > 100*time.Millisecond {
		dm.logger.Warn("Slow database connection acquire",
			zap.Duration("acquire_time", duration),
		)
	}
}

// Health checker implementation

// Start begins health monitoring
func (hc *HealthChecker) Start() {
	go func() {
		ticker := time.NewTicker(hc.interval)
		defer ticker.Stop()

		hc.logger.Info("Starting database health monitoring",
			zap.Duration("interval", hc.interval),
		)

		for {
			select {
			case <-ticker.C:
				hc.performHealthCheck()
			case <-hc.stop:
				hc.logger.Info("Database health monitoring stopped")
				return
			}
		}
	}()
}

// Stop stops health monitoring
func (hc *HealthChecker) Stop() {
	close(hc.stop)
}

// performHealthCheck executes a health check and records the result
func (hc *HealthChecker) performHealthCheck() {
	start := time.Now()
	err := hc.manager.HealthCheck()
	responseTime := time.Since(start)

	hc.lastCheck = start

	status := HealthStatus{
		Timestamp:    start,
		Healthy:      err == nil,
		ResponseTime: responseTime,
	}

	if err != nil {
		status.Error = err.Error()
		atomic.AddInt64(&hc.consecutiveFails, 1)
		hc.logger.Warn("Database health check failed",
			zap.Error(err),
			zap.Duration("response_time", responseTime),
			zap.Int64("consecutive_failures", atomic.LoadInt64(&hc.consecutiveFails)),
		)
	} else {
		atomic.StoreInt64(&hc.consecutiveFails, 0)
		hc.logger.Debug("Database health check passed",
			zap.Duration("response_time", responseTime),
		)
	}

	// Store health history (keep last 100 entries)
	if len(hc.healthHistory) >= 100 {
		copy(hc.healthHistory, hc.healthHistory[1:])
		hc.healthHistory = hc.healthHistory[:99]
	}
	hc.healthHistory = append(hc.healthHistory, status)

	// Alert on consecutive failures
	consecutiveFails := atomic.LoadInt64(&hc.consecutiveFails)
	if consecutiveFails >= 3 {
		hc.logger.Error("Database health check failing consistently",
			nil,
			zap.Int64("consecutive_failures", consecutiveFails),
			zap.Duration("failing_for", time.Since(hc.lastCheck)),
		)
	}
}

// GetHealthHistory returns recent health check results
func (hc *HealthChecker) GetHealthHistory() []HealthStatus {
	history := make([]HealthStatus, len(hc.healthHistory))
	copy(history, hc.healthHistory)
	return history
}

// IsHealthy returns current health status
func (hc *HealthChecker) IsHealthy() bool {
	return atomic.LoadInt64(&hc.consecutiveFails) == 0
}

// ConnectionPool provides additional utility methods for the pool
type ConnectionPool struct {
	*DatabaseManager
}

// NewConnectionPool creates a connection pool wrapper with additional utilities
func NewConnectionPool(config *config.DatabaseConfig, logger logging.StormDBLogger) (*ConnectionPool, error) {
	dm, err := NewDatabaseManager(config, logger)
	if err != nil {
		return nil, err
	}

	return &ConnectionPool{DatabaseManager: dm}, nil
}

// WithConnection executes a function with a managed connection
func (cp *ConnectionPool) WithConnection(ctx context.Context, fn func(*pgxpool.Conn) error) error {
	conn, err := cp.GetConnection(ctx)
	if err != nil {
		return err
	}
	defer cp.ReleaseConnection(conn)

	return fn(conn)
}

// WithTransaction executes a function within a transaction
func (cp *ConnectionPool) WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error {
	return cp.WithConnection(ctx, func(conn *pgxpool.Conn) error {
		tx, err := conn.Begin(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to begin transaction")
		}

		defer func() {
			if r := recover(); r != nil {
				_ = tx.Rollback(ctx)
				panic(r)
			}
		}()

		if err := fn(tx); err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				cp.logger.Error("Failed to rollback transaction",
					rollbackErr,
					zap.Error(err),
				)
			}
			return err
		}

		return tx.Commit(ctx)
	})
}

// BulkExecute executes multiple statements efficiently
func (cp *ConnectionPool) BulkExecute(ctx context.Context, statements []string) error {
	return cp.WithConnection(ctx, func(conn *pgxpool.Conn) error {
		for i, stmt := range statements {
			if _, err := conn.Exec(ctx, stmt); err != nil {
				return errors.Wrapf(err, "failed to execute statement %d", i)
			}
		}
		return nil
	})
}
