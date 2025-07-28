// internal/workload/simple/generator.go
package simple

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Generator implements the simple read/write workload
type Generator struct{}

// Setup ensures the schema exists (only if --setup or --rebuild)
func (g *Generator) Setup(ctx context.Context, db *pgxpool.Pool, _ *types.Config) error {
	_, err := db.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS loadtest (
            id BIGINT PRIMARY KEY,
            val TEXT,
            updated TIMESTAMPTZ
        )`)
	if err != nil {
		return err
	}
	return nil
}

// Cleanup drops and recreates the table + loads data (only on --rebuild)
func (g *Generator) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	_, err := db.Exec(ctx, "DROP TABLE IF EXISTS loadtest CASCADE")
	if err != nil {
		return fmt.Errorf("failed to drop loadtest table: %w", err)
	}
	log.Printf("🗑️  Dropped simple loadtest table")

	// Recreate
	if err := g.Setup(ctx, db, cfg); err != nil {
		return fmt.Errorf("failed to recreate schema: %w", err)
	}

	// Seed data
	scale := cfg.Scale
	if scale <= 0 {
		scale = 1000
	}
	for i := 1; i <= scale; i++ {
		_, err := db.Exec(ctx,
			"INSERT INTO loadtest (id, val, updated) VALUES ($1, $2, NOW()) ON CONFLICT (id) DO NOTHING",
			i, fmt.Sprintf("initial_%d", i))
		if err != nil {
			return fmt.Errorf("failed to insert row %d: %w", i, err)
		}
	}
	log.Printf("🌱 Seeded %d rows into loadtest", scale)
	return nil
}

// Run starts the workload
func (g *Generator) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano()))
			g.worker(ctx, db, cfg, rng, metrics)
		}()
	}

	wg.Wait()
	return nil
}

func (g *Generator) worker(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, rng *rand.Rand, metrics *types.Metrics) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()

			var err error
			switch cfg.Workload {
			case "read":
				err = g.doRead(ctx, db, rng, metrics)
			case "write":
				err = g.doWrite(ctx, db, rng, metrics)
			default: // mixed
				if rng.Intn(2) == 0 {
					err = g.doRead(ctx, db, rng, metrics)
				} else {
					err = g.doWrite(ctx, db, rng, metrics)
				}
			}

			elapsed := time.Since(start).Nanoseconds()

			// Record latency
			metrics.Mu.Lock()
			metrics.TransactionDur = append(metrics.TransactionDur, elapsed)
			metrics.Mu.Unlock()

			if err != nil {
				atomic.AddInt64(&metrics.Errors, 1)
				metrics.Mu.Lock()
				metrics.ErrorTypes[err.Error()]++
				metrics.Mu.Unlock()
			} else {
				atomic.AddInt64(&metrics.TPS, 1)
				atomic.AddInt64(&metrics.QPS, 1)
			}

			time.Sleep(time.Millisecond)
		}
	}
}

func (g *Generator) doRead(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand, metrics *types.Metrics) error {
	var val string
	id := rng.Intn(1000) + 1
	row := db.QueryRow(ctx, "SELECT val FROM loadtest WHERE id = $1", id)
	err := row.Scan(&val)
	if err == nil {
		atomic.AddInt64(&metrics.RowsRead, 1)
	}
	return err
}

func (g *Generator) doWrite(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand, metrics *types.Metrics) error {
	id := rng.Intn(1000) + 1
	val := fmt.Sprintf("updated_%d", time.Now().UnixNano())
	_, err := db.Exec(ctx, "UPDATE loadtest SET val = $1, updated = NOW() WHERE id = $2", val, id)
	if err == nil {
		atomic.AddInt64(&metrics.RowsModified, 1)
	}
	return err
}
