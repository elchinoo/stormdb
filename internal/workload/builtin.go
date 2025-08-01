// Package builtin provides fallback built-in workload implementations
// for use when plugins cannot be loaded (e.g., during testing).
package workload

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/elchinoo/stormdb/pkg/plugin"
	"github.com/elchinoo/stormdb/pkg/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SimpleBuiltinWorkload provides a simple built-in workload for testing
type SimpleBuiltinWorkload struct {
	initialized bool
}

// Setup initializes the simple workload
func (s *SimpleBuiltinWorkload) Setup(ctx context.Context, pool *pgxpool.Pool, config *types.Config) error {
	// Create a simple test table
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS loadtest (
			id SERIAL PRIMARY KEY,
			data TEXT NOT NULL,
			timestamp TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create loadtest table: %w", err)
	}

	// Seed some initial data
	for i := 0; i < config.Scale; i++ {
		_, err = pool.Exec(ctx,
			"INSERT INTO loadtest (data) VALUES ($1)",
			fmt.Sprintf("test-data-%d", i))
		if err != nil {
			return fmt.Errorf("failed to seed data: %w", err)
		}
	}

	s.initialized = true
	return nil
}

// Run executes the simple workload
func (s *SimpleBuiltinWorkload) Run(ctx context.Context, pool *pgxpool.Pool, config *types.Config, metrics *types.Metrics) error {
	if !s.initialized {
		return fmt.Errorf("workload not initialized")
	}

	duration, err := time.ParseDuration(config.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	endTime := time.Now().Add(duration)

	for time.Now().Before(endTime) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Perform a simple operation
			operation := rand.Intn(3)
			start := time.Now()

			var err error
			switch operation {
			case 0: // SELECT
				var count int
				err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM loadtest").Scan(&count)
			case 1: // INSERT
				_, err = pool.Exec(ctx, "INSERT INTO loadtest (data) VALUES ($1)",
					fmt.Sprintf("runtime-data-%d", time.Now().UnixNano()))
			case 2: // UPDATE
				_, err = pool.Exec(ctx, "UPDATE loadtest SET data = $1 WHERE id = $2",
					fmt.Sprintf("updated-%d", time.Now().UnixNano()), rand.Intn(config.Scale)+1)
			}

			latency := time.Since(start)

			if err != nil {
				metrics.Errors++
				if metrics.ErrorTypes == nil {
					metrics.ErrorTypes = make(map[string]int64)
				}
				metrics.ErrorTypes[err.Error()]++
			} else {
				metrics.TPS++
				metrics.QPS++
				// Add latency to histogram (simplified)
				if metrics.LatencyHistogram == nil {
					metrics.LatencyHistogram = make(map[string]int64)
				}
				latencyMs := latency.Milliseconds()
				if latencyMs < 1 {
					metrics.LatencyHistogram["<1ms"]++
				} else if latencyMs < 10 {
					metrics.LatencyHistogram["1-10ms"]++
				} else {
					metrics.LatencyHistogram[">10ms"]++
				}
			}
		}
	}

	return nil
}

// Cleanup cleans up the simple workload
func (s *SimpleBuiltinWorkload) Cleanup(ctx context.Context, pool *pgxpool.Pool, config *types.Config) error {
	_, err := pool.Exec(ctx, "DROP TABLE IF EXISTS loadtest")
	return err
}

// GetName returns the workload name
func (s *SimpleBuiltinWorkload) GetName() string {
	return "simple"
}

// TPCCBuiltinWorkload provides a minimal TPCC built-in workload for testing
type TPCCBuiltinWorkload struct {
	initialized bool
}

// Setup initializes the TPCC workload
func (t *TPCCBuiltinWorkload) Setup(ctx context.Context, pool *pgxpool.Pool, config *types.Config) error {
	// Create minimal TPCC tables
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS warehouse (
			w_id INTEGER PRIMARY KEY,
			w_name VARCHAR(10),
			w_tax DECIMAL(4,4)
		);
		
		CREATE TABLE IF NOT EXISTS orders (
			o_id SERIAL PRIMARY KEY,
			o_w_id INTEGER,
			o_entry_d TIMESTAMP DEFAULT NOW(),
			o_ol_cnt INTEGER
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create TPCC tables: %w", err)
	}

	// Seed minimal data
	for i := 1; i <= config.Scale; i++ {
		_, err = pool.Exec(ctx,
			"INSERT INTO warehouse (w_id, w_name, w_tax) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
			i, fmt.Sprintf("warehouse%d", i), 0.05)
		if err != nil {
			return fmt.Errorf("failed to seed warehouse data: %w", err)
		}
	}

	t.initialized = true
	return nil
}

// Run executes the TPCC workload
func (t *TPCCBuiltinWorkload) Run(ctx context.Context, pool *pgxpool.Pool, config *types.Config, metrics *types.Metrics) error {
	if !t.initialized {
		return fmt.Errorf("workload not initialized")
	}

	duration, err := time.ParseDuration(config.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}

	endTime := time.Now().Add(duration)

	for time.Now().Before(endTime) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Simulate simple TPCC transactions
			start := time.Now()

			// Simple "new order" transaction simulation
			_, err := pool.Exec(ctx,
				"INSERT INTO orders (o_w_id, o_ol_cnt) VALUES ($1, $2)",
				rand.Intn(config.Scale)+1, rand.Intn(10)+1)

			latency := time.Since(start)

			if err != nil {
				metrics.Errors++
				if metrics.ErrorTypes == nil {
					metrics.ErrorTypes = make(map[string]int64)
				}
				metrics.ErrorTypes[err.Error()]++
			} else {
				metrics.TPS++
				metrics.QPS++
				// Add latency to histogram (simplified)
				if metrics.LatencyHistogram == nil {
					metrics.LatencyHistogram = make(map[string]int64)
				}
				latencyMs := latency.Milliseconds()
				if latencyMs < 1 {
					metrics.LatencyHistogram["<1ms"]++
				} else if latencyMs < 10 {
					metrics.LatencyHistogram["1-10ms"]++
				} else {
					metrics.LatencyHistogram[">10ms"]++
				}
			}
		}
	}

	return nil
}

// Cleanup cleans up the TPCC workload
func (t *TPCCBuiltinWorkload) Cleanup(ctx context.Context, pool *pgxpool.Pool, config *types.Config) error {
	_, err := pool.Exec(ctx, "DROP TABLE IF EXISTS orders, warehouse")
	return err
}

// GetName returns the workload name
func (t *TPCCBuiltinWorkload) GetName() string {
	return "tpcc"
}

// GetBuiltinWorkload returns a built-in workload implementation if available
func GetBuiltinWorkload(workloadType string) plugin.Workload {
	switch workloadType {
	case "simple":
		return &SimpleBuiltinWorkload{}
	case "tpcc":
		return &TPCCBuiltinWorkload{}
	default:
		return nil
	}
}
