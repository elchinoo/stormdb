// plugins/vector_plugin/pgvector_sequential.go
// Sequential operations (no indexes) for comprehensive pgvector testing
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// Cleanup removes all test tables and prepares for fresh setup
func (w *ComprehensivePgVectorWorkload) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	log.Printf("🧹 Cleaning up pgvector test environment...")

	// Drop all test tables
	tables := []string{
		"pgvector_test",
		"pgvector_ground_truth",
	}

	for _, table := range tables {
		_, err := db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			log.Printf("Warning: Failed to drop table %s: %v", table, err)
		}
	}

	log.Printf("✅ Cleanup completed")
	return nil
}

// Setup initializes the test environment (without indexes for sequential tests)
func (w *ComprehensivePgVectorWorkload) Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	log.Printf("🚀 Setting up pgvector test environment...")

	// Parse configuration from workload type
	w.parseWorkloadType(cfg.Workload)

	// Ensure pgvector extension is available
	if err := w.ensurePgVectorExtension(ctx, db); err != nil {
		return fmt.Errorf("pgvector extension setup failed: %w", err)
	}

	// Create test tables
	if err := w.createTestTables(ctx, db); err != nil {
		return fmt.Errorf("table creation failed: %w", err)
	}

	// Load precomputed vectors for consistent testing
	if err := w.loadPrecomputedVectors(ctx); err != nil {
		return fmt.Errorf("precomputed vectors loading failed: %w", err)
	}

	// Load baseline data (without creating indexes)
	if err := w.loadBaselineData(ctx, db, cfg); err != nil {
		return fmt.Errorf("baseline data loading failed: %w", err)
	}

	log.Printf("✅ Setup completed successfully")
	return nil
}

// Run executes the main test logic
func (w *ComprehensivePgVectorWorkload) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	log.Printf("🏃 Running pgvector test: %s", w.TestType)

	switch w.TestType {
	case "ingestion":
		return w.runIngestionTest(ctx, db, cfg, metrics)
	case "update":
		return w.runUpdateTest(ctx, db, cfg, metrics)
	case "read":
		return w.runReadTest(ctx, db, cfg, metrics)
	default:
		return fmt.Errorf("unknown test type: %s", w.TestType)
	}
}

// ensurePgVectorExtension checks and enables pgvector extension
func (w *ComprehensivePgVectorWorkload) ensurePgVectorExtension(ctx context.Context, db *pgxpool.Pool) error {
	var extName string
	err := db.QueryRow(ctx, "SELECT extname FROM pg_extension WHERE extname = 'vector'").Scan(&extName)
	if err != nil {
		// Try to create the extension
		_, err = db.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
		if err != nil {
			return fmt.Errorf("pgvector extension is not available and cannot be created: %w", err)
		}
		log.Printf("✅ Created pgvector extension")
	} else {
		log.Printf("✅ pgvector extension is available")
	}
	return nil
}

// createTestTables creates all necessary tables for comprehensive testing
func (w *ComprehensivePgVectorWorkload) createTestTables(ctx context.Context, db *pgxpool.Pool) error {
	log.Printf("📋 Creating test tables...")

	// Main test table
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS pgvector_test (
			id BIGSERIAL PRIMARY KEY,
			name TEXT,
			embedding VECTOR(%d),
			category TEXT,
			metadata JSONB,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`, w.Dimensions)

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to create main test table: %w", err)
	}

	// Ground truth table for accuracy testing
	query = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS pgvector_ground_truth (
			id BIGSERIAL PRIMARY KEY,
			query_vector VECTOR(%d),
			true_neighbors BIGINT[],
			similarity_metric TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`, w.Dimensions)

	if _, err := db.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to create ground truth table: %w", err)
	}

	log.Printf("✅ Created test tables")
	return nil
}

// loadPrecomputedVectors loads or generates 10% of test vectors for consistent testing
func (w *ComprehensivePgVectorWorkload) loadPrecomputedVectors(ctx context.Context) error {
	precomputedCount := w.BaselineRows / 10 // 10% of baseline data
	filename := fmt.Sprintf("precomputed_vectors_%d_%d.csv", w.Dimensions, precomputedCount)

	// Try to load existing file
	if data, err := w.loadVectorsFromFile(filename); err == nil {
		w.PreloadedData = data
		log.Printf("📁 Loaded %d precomputed vectors from %s", len(data), filename)
		return nil
	}

	// Generate and save new vectors
	log.Printf("🔢 Generating %d precomputed vectors (%d dimensions)...", precomputedCount, w.Dimensions)

	w.PreloadedData = make([][]float32, precomputedCount)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < precomputedCount; i++ {
		w.PreloadedData[i] = w.generateRandomVector(rng)

		if i > 0 && i%10000 == 0 {
			log.Printf("⏳ Generated %d / %d vectors...", i, precomputedCount)
		}
	}

	// Save to file for future use
	if err := w.saveVectorsToFile(filename, w.PreloadedData); err != nil {
		log.Printf("⚠️  Failed to save precomputed vectors: %v", err)
	} else {
		log.Printf("💾 Saved precomputed vectors to %s", filename)
	}

	log.Printf("✅ Generated %d precomputed vectors", len(w.PreloadedData))
	return nil
}

// loadBaselineData loads baseline data into test table using COPY protocol
func (w *ComprehensivePgVectorWorkload) loadBaselineData(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	log.Printf("📊 Loading %d baseline rows using COPY protocol...", w.BaselineRows)

	// Prepare data for COPY
	var data strings.Builder
	rng := rand.New(rand.NewSource(42)) // Use fixed seed for reproducible results

	for i := 0; i < w.BaselineRows; i++ {
		vector := w.generateRandomVector(rng)
		vectorStr := pgvector.NewVector(vector).String()

		data.WriteString(fmt.Sprintf("%d\tbaseline_item_%d\t%s\tcategory_%d\t{\"index\": %d}\n",
			i+1, i, vectorStr, i%100, i))

		if i > 0 && i%50000 == 0 {
			log.Printf("⏳ Prepared %d / %d rows...", i, w.BaselineRows)
		}
	}

	// Use COPY protocol for fast bulk loading
	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	_, err = conn.Conn().PgConn().CopyFrom(ctx,
		strings.NewReader(data.String()),
		"COPY pgvector_test (id, name, embedding, category, metadata) FROM STDIN")
	if err != nil {
		return fmt.Errorf("COPY execution failed: %w", err)
	}

	log.Printf("✅ Loaded %d baseline rows", w.BaselineRows)
	return nil
}

// runIngestionTest tests various data ingestion methods
func (w *ComprehensivePgVectorWorkload) runIngestionTest(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	log.Printf("🔄 Running ingestion test with method: %s", w.IngestionMethod)

	switch w.IngestionMethod {
	case "single":
		return w.runSingleInsertTest(ctx, db, cfg, metrics)
	case "batch":
		return w.runBatchInsertTest(ctx, db, cfg, metrics)
	case "copy":
		return w.runCopyInsertTest(ctx, db, cfg, metrics)
	default:
		return fmt.Errorf("unknown ingestion method: %s", w.IngestionMethod)
	}
}

// runSingleInsertTest tests single row inserts
func (w *ComprehensivePgVectorWorkload) runSingleInsertTest(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	log.Printf("📝 Single insert test starting...")

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var totalOps int64

	for i := 0; i < cfg.Workers; i++ {
		go func(workerID int) {
			conn, err := db.Acquire(ctx)
			if err != nil {
				log.Printf("Worker %d failed to acquire connection: %v", workerID, err)
				return
			}
			defer conn.Release()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					vector := w.generateRandomVector(rng)

					start := time.Now()
					_, err := conn.Exec(ctx,
						"INSERT INTO pgvector_test (name, embedding, category, metadata) VALUES ($1, $2, $3, $4)",
						fmt.Sprintf("single_test_%d_%d", workerID, atomic.LoadInt64(&totalOps)),
						pgvector.NewVector(vector),
						fmt.Sprintf("category_%d", rng.Intn(100)),
						fmt.Sprintf(`{"worker": %d, "timestamp": "%s"}`, workerID, time.Now().Format(time.RFC3339)),
					)
					duration := time.Since(start)

					if err != nil {
						atomic.AddInt64(&metrics.Errors, 1)
						log.Printf("Insert error: %v", err)
					} else {
						metrics.RecordLatency(duration.Nanoseconds())
						metrics.RecordQuery("INSERT")
						atomic.AddInt64(&totalOps, 1)
					}
				}
			}
		}(i)
	}

	log.Printf("✅ Single insert test completed")
	return nil
}

// runBatchInsertTest tests batch inserts
func (w *ComprehensivePgVectorWorkload) runBatchInsertTest(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	log.Printf("📦 Batch insert test starting (batch size: %d)...", w.BatchSize)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var totalOps int64

	for i := 0; i < cfg.Workers; i++ {
		go func(workerID int) {
			conn, err := db.Acquire(ctx)
			if err != nil {
				log.Printf("Worker %d failed to acquire connection: %v", workerID, err)
				return
			}
			defer conn.Release()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					batch := &pgx.Batch{}

					for j := 0; j < w.BatchSize; j++ {
						vector := w.generateRandomVector(rng)
						batch.Queue(
							"INSERT INTO pgvector_test (name, embedding, category, metadata) VALUES ($1, $2, $3, $4)",
							fmt.Sprintf("batch_test_%d_%d_%d", workerID, atomic.LoadInt64(&totalOps), j),
							pgvector.NewVector(vector),
							fmt.Sprintf("category_%d", rng.Intn(100)),
							fmt.Sprintf(`{"worker": %d, "batch": %d}`, workerID, j),
						)
					}

					start := time.Now()
					results := conn.SendBatch(ctx, batch)

					for j := 0; j < w.BatchSize; j++ {
						_, err := results.Exec()
						if err != nil {
							atomic.AddInt64(&metrics.Errors, 1)
							log.Printf("Batch insert error at position %d: %v", j, err)
						}
					}
					results.Close()
					duration := time.Since(start)

					metrics.RecordLatency(duration.Nanoseconds())
					metrics.RecordQuery("INSERT")
					atomic.AddInt64(&totalOps, int64(w.BatchSize))
				}
			}
		}(i)
	}

	log.Printf("✅ Batch insert test completed")
	return nil
}

// runCopyInsertTest tests COPY protocol inserts
func (w *ComprehensivePgVectorWorkload) runCopyInsertTest(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	log.Printf("📄 COPY insert test starting...")

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var totalOps int64
	var wg sync.WaitGroup

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			conn, err := db.Acquire(ctx)
			if err != nil {
				log.Printf("Worker %d failed to acquire connection: %v", workerID, err)
				return
			}
			defer conn.Release()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Prepare data for COPY
					var data strings.Builder
					batchSize := w.BatchSize

					for j := 0; j < batchSize; j++ {
						vector := w.generateRandomVector(rng)
						vectorStr := pgvector.NewVector(vector).String()

						data.WriteString(fmt.Sprintf("copy_test_%d_%d_%d\t%s\tcategory_%d\t{\"worker\": %d}\n",
							workerID, atomic.LoadInt64(&totalOps), j,
							vectorStr,
							rng.Intn(100),
							workerID))
					}

					start := time.Now()
					_, err := conn.Conn().PgConn().CopyFrom(ctx,
						strings.NewReader(data.String()),
						"COPY pgvector_test (name, embedding, category, metadata) FROM STDIN")
					duration := time.Since(start)

					if err != nil {
						atomic.AddInt64(&metrics.Errors, 1)
						log.Printf("COPY error: %v", err)
					} else {
						metrics.RecordLatency(duration.Nanoseconds())
						metrics.RecordQuery("INSERT")
						atomic.AddInt64(&totalOps, int64(batchSize))
					}
				}
			}
		}(i)
	}

	wg.Wait()
	log.Printf("✅ COPY insert test completed")
	return nil
}

// runUpdateTest tests vector updates
func (w *ComprehensivePgVectorWorkload) runUpdateTest(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	log.Printf("🔄 Update test starting...")

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var totalOps int64

	// Get the range of existing IDs
	var maxID int64
	err := db.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) FROM pgvector_test").Scan(&maxID)
	if err != nil {
		return fmt.Errorf("failed to get max ID: %w", err)
	}

	if maxID == 0 {
		return fmt.Errorf("no data found in pgvector_test table")
	}

	for i := 0; i < cfg.Workers; i++ {
		go func(workerID int) {
			conn, err := db.Acquire(ctx)
			if err != nil {
				log.Printf("Worker %d failed to acquire connection: %v", workerID, err)
				return
			}
			defer conn.Release()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Pick a random existing ID
					targetID := rng.Int63n(maxID) + 1
					vector := w.generateRandomVector(rng)

					start := time.Now()
					_, err := conn.Exec(ctx,
						"UPDATE pgvector_test SET embedding = $1, name = $2 WHERE id = $3",
						pgvector.NewVector(vector),
						fmt.Sprintf("updated_%d_%d", workerID, atomic.LoadInt64(&totalOps)),
						targetID,
					)
					duration := time.Since(start)

					if err != nil {
						atomic.AddInt64(&metrics.Errors, 1)
						log.Printf("Update error: %v", err)
					} else {
						metrics.RecordLatency(duration.Nanoseconds())
						metrics.RecordQuery("UPDATE")
						atomic.AddInt64(&totalOps, 1)
					}
				}
			}
		}(i)
	}

	log.Printf("✅ Update test completed")
	return nil
}

// runReadTest tests vector similarity queries
func (w *ComprehensivePgVectorWorkload) runReadTest(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	log.Printf("🔍 Read test starting (type: %s)...", w.ReadType)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var totalOps int64

	for i := 0; i < cfg.Workers; i++ {
		go func(workerID int) {
			conn, err := db.Acquire(ctx)
			if err != nil {
				log.Printf("Worker %d failed to acquire connection: %v", workerID, err)
				return
			}
			defer conn.Release()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					// Use precomputed vector for consistent testing
					vectorIndex := rng.Intn(len(w.PreloadedData))
					queryVector := w.PreloadedData[vectorIndex]

					var query string
					switch w.SimilarityMetric {
					case "cosine":
						query = "SELECT id, name, embedding <=> $1 as distance FROM pgvector_test ORDER BY embedding <=> $1 LIMIT 10"
					case "inner_product":
						query = "SELECT id, name, embedding <#> $1 as distance FROM pgvector_test ORDER BY embedding <#> $1 DESC LIMIT 10"
					default: // l2
						query = "SELECT id, name, embedding <-> $1 as distance FROM pgvector_test ORDER BY embedding <-> $1 LIMIT 10"
					}

					start := time.Now()
					rows, err := conn.Query(ctx, query, pgvector.NewVector(queryVector))
					if err != nil {
						atomic.AddInt64(&metrics.Errors, 1)
						log.Printf("Query error: %v", err)
						continue
					}

					var results []struct {
						ID       int64
						Name     string
						Distance float32
					}

					for rows.Next() {
						var result struct {
							ID       int64
							Name     string
							Distance float32
						}
						err := rows.Scan(&result.ID, &result.Name, &result.Distance)
						if err != nil {
							log.Printf("Scan error: %v", err)
							continue
						}
						results = append(results, result)
					}
					rows.Close()
					duration := time.Since(start)

					metrics.RecordLatency(duration.Nanoseconds())
					metrics.RecordQuery("SELECT")
					atomic.AddInt64(&totalOps, 1)
				}
			}
		}(i)
	}

	log.Printf("✅ Read test completed")
	return nil
}
