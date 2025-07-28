// Package connection_overhead implements a specialized workload for measuring
// and comparing the performance impact of different database connection strategies.
//
// This workload is designed to quantify the overhead associated with persistent
// versus transient database connections in PostgreSQL environments. It provides
// detailed metrics on connection establishment time, transaction throughput,
// and query latency for both connection modes.
//
// The workload generates a mixed set of database operations (SELECT, INSERT,
// UPDATE, DELETE) and executes them using two different connection strategies:
//
//   - Persistent Connections: Use pooled connections that are maintained
//     across multiple operations, minimizing connection overhead
//   - Transient Connections: Establish a new connection for each operation,
//     measuring the full cost of connection establishment and teardown
//
// Key Measurements:
//   - Connection establishment time (for transient connections)
//   - Transaction latency differences between connection modes
//   - Throughput comparison (TPS/QPS) between persistent and transient
//   - Error rates and failure patterns for each connection strategy
//   - Resource utilization patterns and connection count tracking
//
// This workload is particularly useful for:
//   - Connection pool sizing decisions
//   - Understanding connection overhead in high-throughput scenarios
//   - Evaluating the impact of connection management strategies
//   - Performance tuning for applications with varying connection patterns
package simpleconnection

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/elchinoo/stormdb/internal/database"
	"github.com/elchinoo/stormdb/pkg/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConnectionWorkload implements a benchmarking workload specifically
// designed to measure and compare the performance characteristics of different
// database connection management strategies.
//
// The workload maintains a collection of pre-generated database operations
// that are executed using both persistent (pooled) and transient (per-operation)
// connection strategies. This allows for direct comparison of the performance
// impact of connection overhead.
//
// The workload automatically distributes operations between the two connection
// modes (typically 50/50) and collects detailed metrics for each mode separately,
// enabling comprehensive analysis of connection overhead impact on throughput,
// latency, and resource utilization.
type ConnectionWorkload struct {
	operations []Operation // Pre-generated operations to be executed during the benchmark
}

// Operation represents a single database operation with its connection strategy.
// Each operation encapsulates the SQL query, parameters, and the connection
// mode to be used for execution.
type Operation struct {
	Type         string        // Operation type: "select", "insert", "update", "delete"
	Query        string        // SQL query to execute
	Args         []interface{} // Query parameters for prepared statement execution
	UseTransient bool          // Connection strategy: true=transient, false=persistent
}

func (w *ConnectionWorkload) Setup(ctx context.Context, pool *pgxpool.Pool, config *types.Config) error {
	log.Println("Setting up connection overhead workload...")

	// Create test table if it doesn't exist
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS connection_test (
			id SERIAL PRIMARY KEY,
			data VARCHAR(100) NOT NULL,
			value INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`

	_, err := pool.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create test table: %w", err)
	}

	// Insert some initial data if table is empty
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM connection_test").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check table count: %w", err)
	}

	if count == 0 {
		log.Println("Inserting initial test data...")
		for i := 0; i < 1000; i++ {
			_, err = pool.Exec(ctx,
				"INSERT INTO connection_test (data, value) VALUES ($1, $2)",
				fmt.Sprintf("test_data_%d", i), rand.Intn(10000))
			if err != nil {
				return fmt.Errorf("failed to insert initial data: %w", err)
			}
		}
	}

	log.Printf("Connection overhead workload setup completed")
	return nil
}

func (w *ConnectionWorkload) Cleanup(ctx context.Context, pool *pgxpool.Pool, config *types.Config) error {
	log.Println("Cleaning up connection overhead workload...")

	// Clean up test data
	_, err := pool.Exec(ctx, "DROP TABLE IF EXISTS connection_test")
	if err != nil {
		log.Printf("Warning: failed to drop test table: %v", err)
		return err
	}

	return nil
}

func (w *ConnectionWorkload) Run(ctx context.Context, pool *pgxpool.Pool, config *types.Config, metrics *types.Metrics) error {
	duration, err := time.ParseDuration(config.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	// Initialize connection mode metrics
	if metrics.PersistentConnMetrics == nil {
		metrics.PersistentConnMetrics = &types.ConnectionModeMetrics{}
	}
	if metrics.TransientConnMetrics == nil {
		metrics.TransientConnMetrics = &types.ConnectionModeMetrics{}
	}

	log.Printf("Starting connection overhead workload for %v with %d workers", duration, config.Workers)

	// Create context with timeout
	runCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	// Prepare operations (50% persistent, 50% transient)
	w.operations = w.generateOperations()

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go w.worker(runCtx, &wg, i, pool, config, metrics)
	}

	// Wait for completion
	wg.Wait()

	log.Println("Connection overhead workload completed")
	return nil
}

func (w *ConnectionWorkload) generateOperations() []Operation {
	operations := make([]Operation, 0, 1000)

	// Define operation templates
	selectOps := []string{
		"SELECT id, data, value FROM connection_test WHERE id = $1",
		"SELECT COUNT(*) FROM connection_test WHERE value > $1",
		"SELECT data FROM connection_test ORDER BY id LIMIT $1",
		"SELECT AVG(value) FROM connection_test WHERE created_at > $1",
	}

	insertOps := []string{
		"INSERT INTO connection_test (data, value) VALUES ($1, $2)",
	}

	updateOps := []string{
		"UPDATE connection_test SET value = $1 WHERE id = $2",
		"UPDATE connection_test SET data = $1 WHERE value < $2",
	}

	deleteOps := []string{
		"DELETE FROM connection_test WHERE id = $1 AND value > $2",
	}

	// Generate mixed operations
	for i := 0; i < 1000; i++ {
		useTransient := i%2 == 0 // 50% transient, 50% persistent

		switch rand.Intn(4) {
		case 0: // SELECT (60% of operations)
			if rand.Float32() < 0.6 {
				query := selectOps[rand.Intn(len(selectOps))]
				operations = append(operations, Operation{
					Type:         "select",
					Query:        query,
					Args:         w.generateArgsForQuery(query),
					UseTransient: useTransient,
				})
			}
		case 1: // INSERT (20% of operations)
			if rand.Float32() < 0.2 {
				query := insertOps[rand.Intn(len(insertOps))]
				operations = append(operations, Operation{
					Type:         "insert",
					Query:        query,
					Args:         w.generateArgsForQuery(query),
					UseTransient: useTransient,
				})
			}
		case 2: // UPDATE (15% of operations)
			if rand.Float32() < 0.15 {
				query := updateOps[rand.Intn(len(updateOps))]
				operations = append(operations, Operation{
					Type:         "update",
					Query:        query,
					Args:         w.generateArgsForQuery(query),
					UseTransient: useTransient,
				})
			}
		case 3: // DELETE (5% of operations)
			if rand.Float32() < 0.05 {
				query := deleteOps[rand.Intn(len(deleteOps))]
				operations = append(operations, Operation{
					Type:         "delete",
					Query:        query,
					Args:         w.generateArgsForQuery(query),
					UseTransient: useTransient,
				})
			}
		}
	}

	return operations
}

func (w *ConnectionWorkload) generateArgsForQuery(query string) []interface{} {
	paramCount := strings.Count(query, "$")
	args := make([]interface{}, paramCount)

	for i := 0; i < paramCount; i++ {
		switch {
		case strings.Contains(query, "data"):
			args[i] = fmt.Sprintf("test_data_%d", rand.Intn(1000))
		case strings.Contains(query, "LIMIT"):
			args[i] = rand.Intn(50) + 1
		case strings.Contains(query, "created_at"):
			args[i] = time.Now().Add(-time.Duration(rand.Intn(86400)) * time.Second)
		default:
			args[i] = rand.Intn(1000) + 1
		}
	}

	return args
}

func (w *ConnectionWorkload) worker(ctx context.Context, wg *sync.WaitGroup, workerID int, pool *pgxpool.Pool, config *types.Config, metrics *types.Metrics) {
	defer wg.Done()

	rand.Seed(time.Now().UnixNano() + int64(workerID))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Pick a random operation
			if len(w.operations) == 0 {
				continue
			}

			op := w.operations[rand.Intn(len(w.operations))]

			if op.UseTransient {
				w.executeTransientOperation(ctx, workerID, op, pool, config, metrics)
			} else {
				w.executePersistentOperation(ctx, workerID, op, pool, config, metrics)
			}

			// Small delay to prevent overwhelming the database
			time.Sleep(time.Millisecond * time.Duration(rand.Intn(10)+1))
		}
	}
}

func (w *ConnectionWorkload) executePersistentOperation(ctx context.Context, workerID int, op Operation, pool *pgxpool.Pool, config *types.Config, metrics *types.Metrics) {
	start := time.Now()

	// Use connection from pool (persistent)
	conn, err := pool.Acquire(ctx)
	if err != nil {
		metrics.RecordConnectionModeError("persistent")
		metrics.RecordWorkerError(workerID)
		return
	}
	defer conn.Release()

	success := true

	// Execute within transaction for consistency
	tx, err := conn.Begin(ctx)
	if err != nil {
		metrics.RecordConnectionModeError("persistent")
		metrics.RecordWorkerError(workerID)
		return
	}

	// Execute the operation
	if op.Type == "select" {
		rows, err := tx.Query(ctx, op.Query, op.Args...)
		if err != nil {
			success = false
		} else {
			// Consume all rows
			for rows.Next() {
				// Just iterate through results
			}
			rows.Close()
		}
	} else {
		_, err = tx.Exec(ctx, op.Query, op.Args...)
		if err != nil {
			success = false
		}
	}

	// Commit or rollback
	if success {
		err = tx.Commit(ctx)
		if err != nil {
			success = false
		}
	} else {
		_ = tx.Rollback(ctx)
	}

	duration := time.Since(start).Nanoseconds()

	// Record metrics
	metrics.RecordConnectionModeTransaction("persistent", success, duration)
	metrics.RecordConnectionModeQuery("persistent")
	metrics.RecordWorkerTransaction(workerID, success, duration)
	metrics.RecordWorkerQuery(workerID, strings.ToUpper(op.Type))

	if !success {
		metrics.RecordConnectionModeError("persistent")
		metrics.RecordWorkerError(workerID)
	}
}

func (w *ConnectionWorkload) executeTransientOperation(ctx context.Context, workerID int, op Operation, pool *pgxpool.Pool, config *types.Config, metrics *types.Metrics) {
	start := time.Now()

	// Create new connection for each operation (transient)
	connString := database.BuildConnectionString(config)

	// Measure connection setup time
	connStart := time.Now()
	conn, err := pgx.Connect(ctx, connString)
	connSetupTime := time.Since(connStart).Nanoseconds()

	if err != nil {
		metrics.RecordConnectionModeError("transient")
		metrics.RecordWorkerError(workerID)
		return
	}
	defer conn.Close(ctx)

	// Record connection setup metrics
	metrics.RecordConnectionSetup(connSetupTime)

	success := true

	// Execute within transaction for consistency
	tx, err := conn.Begin(ctx)
	if err != nil {
		metrics.RecordConnectionModeError("transient")
		metrics.RecordWorkerError(workerID)
		return
	}

	// Execute the operation
	if op.Type == "select" {
		rows, err := tx.Query(ctx, op.Query, op.Args...)
		if err != nil {
			success = false
		} else {
			// Consume all rows
			for rows.Next() {
				// Just iterate through results
			}
			rows.Close()
		}
	} else {
		_, err = tx.Exec(ctx, op.Query, op.Args...)
		if err != nil {
			success = false
		}
	}

	// Commit or rollback
	if success {
		err = tx.Commit(ctx)
		if err != nil {
			success = false
		}
	} else {
		_ = tx.Rollback(ctx)
	}

	duration := time.Since(start).Nanoseconds()

	// Record metrics
	metrics.RecordConnectionModeTransaction("transient", success, duration)
	metrics.RecordConnectionModeQuery("transient")
	metrics.RecordWorkerTransaction(workerID, success, duration)
	metrics.RecordWorkerQuery(workerID, strings.ToUpper(op.Type))

	if !success {
		metrics.RecordConnectionModeError("transient")
		metrics.RecordWorkerError(workerID)
	}
}
