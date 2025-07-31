// internal/workload/bulk_insert/generator.go
package bulk_insert

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Generator implements the bulk insert workload with progressive scaling
type Generator struct{}

// BulkInsertConfig holds configuration specific to bulk insert workload
type BulkInsertConfig struct {
	// Buffer configuration for producer-consumer pattern
	RingBufferSize  int `mapstructure:"ring_buffer_size"` // Size of the ring buffer (default: 100000)
	ProducerThreads int `mapstructure:"producer_threads"` // Number of producer threads (default: 2)

	// Batch size progression: 1, 100, 1000, 10000, 50000
	BatchSizes []int `mapstructure:"batch_sizes"` // Batch sizes to test (default: [1, 100, 1000, 10000, 50000])

	// Insert method testing: INSERT vs COPY
	TestInsertMethod bool `mapstructure:"test_insert_method"` // Test both INSERT and COPY methods (default: true)

	// Data generation settings
	DataSeed int64 `mapstructure:"data_seed"` // Seed for data generation (0 = random)

	// Performance settings
	MaxMemoryMB int `mapstructure:"max_memory_mb"` // Maximum memory usage in MB (default: 512)

	// Analysis settings
	CollectMetrics bool `mapstructure:"collect_metrics"` // Collect detailed metrics (default: true)
}

// WorkloadState tracks the current state of bulk insert testing
type WorkloadState struct {
	currentBand   int
	currentMethod string // "insert" or "copy"
	currentBatch  int
	totalBands    int

	// Ring buffer for producer-consumer pattern
	ringBuffer *RingBuffer

	// Data generator
	dataGenerator *DataGenerator

	// Statistics
	totalInserted int64

	// Control
	stopProducers context.CancelFunc
	producerWg    sync.WaitGroup
}

// Setup ensures the schema exists (only if --setup or --rebuild)
func (g *Generator) Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	return setupSchema(ctx, db)
}

// Cleanup drops and recreates the table (only on --rebuild)
func (g *Generator) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	if err := cleanupSchema(ctx, db); err != nil {
		return err
	}
	return g.Setup(ctx, db, cfg)
}

// Run executes the bulk insert workload with progressive scaling
func (g *Generator) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	// Parse bulk insert specific configuration
	bulkCfg := g.parseBulkInsertConfig(cfg)

	// Initialize workload state
	state := &WorkloadState{
		dataGenerator: NewDataGenerator(bulkCfg.DataSeed),
		ringBuffer:    NewRingBuffer(bulkCfg.RingBufferSize),
	}

	// Clear table before starting
	if err := truncateTable(ctx, db); err != nil {
		return fmt.Errorf("failed to truncate table: %w", err)
	}

	log.Printf("🚀 Starting bulk insert workload test")
	log.Printf("   Buffer size: %d records", bulkCfg.RingBufferSize)
	log.Printf("   Producer threads: %d", bulkCfg.ProducerThreads)
	log.Printf("   Batch sizes: %v", bulkCfg.BatchSizes)
	log.Printf("   Test methods: %s", g.getTestMethods(bulkCfg))

	// Progressive scaling setup
	if cfg.Progressive.Enabled {
		return g.runProgressiveTest(ctx, db, cfg, metrics, bulkCfg, state)
	} else {
		return g.runStandardTest(ctx, db, cfg, metrics, bulkCfg, state)
	}
}

// runProgressiveTest executes the workload with progressive scaling
func (g *Generator) runProgressiveTest(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics, bulkCfg *BulkInsertConfig, state *WorkloadState) error {
	progressiveCfg := cfg.Progressive

	// Calculate test matrix: methods × batch sizes
	methods := g.getTestMethodsList(bulkCfg)
	batchSizes := bulkCfg.BatchSizes

	totalCombinations := len(methods) * len(batchSizes)
	bandsPerCombination := progressiveCfg.Bands / totalCombinations
	if bandsPerCombination == 0 {
		bandsPerCombination = 1
	}

	state.totalBands = totalCombinations * bandsPerCombination

	log.Printf("📊 Progressive bulk insert test: %d methods × %d batch sizes × %d bands = %d total bands",
		len(methods), len(batchSizes), bandsPerCombination, state.totalBands)

	bandIndex := 0

	for _, method := range methods {
		for _, batchSize := range batchSizes {
			for band := 0; band < bandsPerCombination; band++ {
				bandIndex++
				state.currentBand = bandIndex
				state.currentMethod = method
				state.currentBatch = batchSize

				// Calculate progressive scaling for this band
				progress := float64(bandIndex-1) / float64(state.totalBands-1)
				workers := g.calculateWorkers(cfg, progress)
				connections := g.calculateConnections(cfg, progress)

				log.Printf("🔄 Band %d/%d: %s method, batch size %d, %d workers, %d connections",
					bandIndex, state.totalBands, method, batchSize, workers, connections)

				// Run this band
				if err := g.runBand(ctx, db, cfg, metrics, bulkCfg, state, method, batchSize, workers, connections); err != nil {
					return fmt.Errorf("band %d failed: %w", bandIndex, err)
				}

				// Cooldown between bands
				if band < bandsPerCombination-1 || (method != methods[len(methods)-1] || batchSize != batchSizes[len(batchSizes)-1]) {
					cooldownDuration, err := time.ParseDuration(progressiveCfg.CooldownDuration)
					if err != nil {
						cooldownDuration = time.Second * 30 // Default cooldown
					}
					time.Sleep(cooldownDuration)
				}
			}
		}
	}

	return nil
}

// runStandardTest executes a standard (non-progressive) test
func (g *Generator) runStandardTest(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics, bulkCfg *BulkInsertConfig, state *WorkloadState) error {
	// For standard test, test all combinations sequentially
	methods := g.getTestMethodsList(bulkCfg)
	batchSizes := bulkCfg.BatchSizes

	durationPer, err := time.ParseDuration(cfg.Duration)
	if err != nil {
		durationPer = time.Minute * 5 // Default duration
	}
	testDuration := durationPer / time.Duration(len(methods)*len(batchSizes))

	for _, method := range methods {
		for _, batchSize := range batchSizes {
			state.currentMethod = method
			state.currentBatch = batchSize

			log.Printf("🔄 Testing %s method with batch size %d for %v", method, batchSize, testDuration)

			// Run this test with calculated duration
			if err := g.runBand(ctx, db, cfg, metrics, bulkCfg, state, method, batchSize, cfg.Workers, cfg.Connections); err != nil {
				return fmt.Errorf("test %s/%d failed: %w", method, batchSize, err)
			}
		}
	}

	return nil
}

// runBand executes a single test band
func (g *Generator) runBand(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics, bulkCfg *BulkInsertConfig, state *WorkloadState, method string, batchSize, workers, connections int) error {
	// Create context for this band
	var duration time.Duration
	var err error

	if cfg.Progressive.Enabled {
		duration, err = time.ParseDuration(cfg.Progressive.TestDuration)
		if err != nil {
			duration = time.Minute * 30 // Default 30 minutes
		}
	} else {
		duration, err = time.ParseDuration(cfg.Duration)
		if err != nil {
			duration = time.Minute * 5 // Default 5 minutes
		}
	}

	bandCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	// Reset ring buffer for this band
	state.ringBuffer.Reset()

	// Start producers
	producerCtx, producerCancel := context.WithCancel(bandCtx)
	state.stopProducers = producerCancel
	defer producerCancel()

	for i := 0; i < bulkCfg.ProducerThreads; i++ {
		state.producerWg.Add(1)
		// Create a separate data generator for each producer thread to avoid race conditions
		// since math/rand.Rand is not thread-safe
		producerDataGen := NewDataGenerator(bulkCfg.DataSeed + int64(i))
		go g.producer(producerCtx, state, bulkCfg, &state.producerWg, producerDataGen)
	}

	// Start consumers (workers)
	var workerWg sync.WaitGroup
	for i := 0; i < workers; i++ {
		workerWg.Add(1)
		// Pass a separate context for database operations that doesn't expire during the test
		dbCtx := context.Background()
		go g.consumer(bandCtx, dbCtx, db, state, method, batchSize, metrics, &workerWg, i)
	}

	// Wait for test completion
	<-bandCtx.Done()

	// Stop producers first
	producerCancel()
	state.producerWg.Wait()

	// Close ring buffer to signal consumers
	state.ringBuffer.Close()

	// Give consumers a grace period to finish their current operations
	// Create a separate context with a reasonable timeout for cleanup
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cleanupCancel()

	// Wait for consumers to finish with grace period
	done := make(chan struct{})
	go func() {
		workerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All workers finished gracefully
	case <-cleanupCtx.Done():
		// Grace period expired, but this is expected behavior
		log.Printf("⚠️  Some workers didn't finish within grace period - this is normal")
	}

	// Log band results
	produced, consumed, waitTime, utilization := state.ringBuffer.Stats()
	log.Printf("✅ Band completed: produced=%d, consumed=%d, utilization=%.2f%%, wait_time=%v",
		produced, consumed, utilization*100, time.Duration(waitTime))

	return nil
}

// producer generates data and feeds it into the ring buffer
func (g *Generator) producer(ctx context.Context, state *WorkloadState, bulkCfg *BulkInsertConfig, wg *sync.WaitGroup, dataGen *DataGenerator) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Generate a record using the thread-specific data generator
		record := dataGen.GenerateRecord()

		// Try to push to ring buffer
		for !state.ringBuffer.Push(record) {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(time.Microsecond * 10) // Brief backoff if buffer is full
			}
		}
	}
}

// consumer reads from ring buffer and performs bulk inserts
func (g *Generator) consumer(ctx context.Context, dbCtx context.Context, db *pgxpool.Pool, state *WorkloadState, method string, batchSize int, metrics *types.Metrics, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Get batch from ring buffer
		records, err := state.ringBuffer.PopBatchBlocking(ctx, 1, batchSize, time.Millisecond*100)
		if err != nil || len(records) == 0 {
			if state.ringBuffer.IsClosed() && state.ringBuffer.Size() == 0 {
				return // No more data to process
			}
			continue
		}

		// Perform the insert operation using dbCtx which doesn't expire during test
		start := time.Now()
		var insertErr error

		switch method {
		case "insert":
			insertErr = g.performBatchInsert(dbCtx, db, records)
		case "copy":
			insertErr = g.performCopyInsert(dbCtx, db, records)
		default:
			insertErr = fmt.Errorf("unknown insert method: %s", method)
		}

		duration := time.Since(start)

		// Update metrics
		if insertErr != nil {
			atomic.AddInt64(&metrics.Errors, 1)
			log.Printf("❌ Worker %d insert error: %v", workerID, insertErr)
		} else {
			atomic.AddInt64(&metrics.TPS, 1)
			atomic.AddInt64(&state.totalInserted, int64(len(records)))

			// Record latency with proper storage for percentile calculations
			latencyNs := duration.Nanoseconds()
			metrics.RecordLatencyWithLimit(latencyNs)
		}
	}
}

// performBatchInsert executes a batch INSERT operation
func (g *Generator) performBatchInsert(ctx context.Context, db *pgxpool.Pool, records []DataRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Build batch insert SQL
	valueStrings := make([]string, len(records))
	valueArgs := make([]interface{}, 0, len(records)*17) // 17 columns per record (excluding external_id with default)

	for i, record := range records {
		valueStrings[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			len(valueArgs)+1, len(valueArgs)+2, len(valueArgs)+3, len(valueArgs)+4,
			len(valueArgs)+5, len(valueArgs)+6, len(valueArgs)+7, len(valueArgs)+8,
			len(valueArgs)+9, len(valueArgs)+10, len(valueArgs)+11, len(valueArgs)+12,
			len(valueArgs)+13, len(valueArgs)+14, len(valueArgs)+15, len(valueArgs)+16,
			len(valueArgs)+17)

		valueArgs = append(valueArgs,
			record.ShortText,
			record.MediumText,
			record.LongText,
			record.IntValue,
			record.BigintValue,
			record.DecimalValue,
			record.FloatValue,
			record.EventDate,
			record.EventTime,
			record.IsActive,
			record.Metadata,
			record.DataBlob,
			record.StatusEnum,
			g.formatStringArray(record.Tags),
			record.ClientIP,
			fmt.Sprintf("(%f,%f)", record.LocationX, record.LocationY),
			time.Now(), // created_timestamp
			// external_id excluded - will use DEFAULT gen_random_uuid()
		)
	}

	sqlQuery := fmt.Sprintf(`
		INSERT INTO bulk_insert_test (
			short_text, medium_text, long_text, int_value, bigint_value,
			decimal_value, float_value, event_date, event_time, is_active,
			metadata, data_blob, status_enum, tags, client_ip, location,
			created_timestamp
		) VALUES %s`, strings.Join(valueStrings, ","))

	_, err := db.Exec(ctx, sqlQuery, valueArgs...)
	return err
}

// performCopyInsert executes a COPY operation
func (g *Generator) performCopyInsert(ctx context.Context, db *pgxpool.Pool, records []DataRecord) error {
	if len(records) == 0 {
		return nil
	}

	// Get a connection from the pool
	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Start COPY operation
	copySource := pgx.CopyFromSlice(len(records), func(i int) ([]interface{}, error) {
		record := records[i]
		return []interface{}{
			record.ShortText,
			record.MediumText,
			record.LongText,
			record.IntValue,
			record.BigintValue,
			record.DecimalValue,
			record.FloatValue,
			record.EventDate,
			record.EventTime,
			record.IsActive,
			record.Metadata,
			record.DataBlob,
			record.StatusEnum, // Pass directly like other string fields
			g.formatStringArray(record.Tags),
			record.ClientIP,
			fmt.Sprintf("(%f,%f)", record.LocationX, record.LocationY),
			time.Now(), // created_timestamp
		}, nil
	})

	_, err = conn.Conn().CopyFrom(ctx, pgx.Identifier{"bulk_insert_test"},
		[]string{
			"short_text", "medium_text", "long_text", "int_value", "bigint_value",
			"decimal_value", "float_value", "event_date", "event_time", "is_active",
			"metadata", "data_blob", "status_enum", "tags", "client_ip", "location",
			"created_timestamp",
		}, copySource)

	return err
}

// Helper functions

func (g *Generator) parseBulkInsertConfig(cfg *types.Config) *BulkInsertConfig {
	bulkCfg := &BulkInsertConfig{
		RingBufferSize:   100000,
		ProducerThreads:  2,
		BatchSizes:       []int{1, 100, 1000, 10000, 50000},
		TestInsertMethod: true,
		DataSeed:         0,
		MaxMemoryMB:      512,
		CollectMetrics:   true,
	}

	// Configuration would typically be parsed from YAML/config files
	// For now, using defaults

	// Sort batch sizes
	sort.Ints(bulkCfg.BatchSizes)

	return bulkCfg
}

// formatStringArray converts a string slice to PostgreSQL array format
func (g *Generator) formatStringArray(tags []string) interface{} {
	if len(tags) == 0 {
		return "{}" // Return empty array instead of nil
	}

	// Safety check for reasonable array size
	if len(tags) > 100 {
		// If tags array is unexpectedly large, truncate to prevent memory issues
		tags = tags[:100]
	}

	// Format as PostgreSQL array literal with safe escaping
	quoted := make([]string, len(tags))
	for i, tag := range tags {
		// Safety check for empty or nil tag strings
		if tag == "" {
			quoted[i] = "\"\""
			continue
		}

		// Ensure tag is reasonable length and escape it safely
		if len(tag) > 1000 {
			tag = tag[:1000] // Truncate very long tags
		}
		// Use simple string replacement instead of fmt.Sprintf for safety
		escaped := strings.ReplaceAll(tag, "\"", "\\\"")
		quoted[i] = "\"" + escaped + "\""
	}
	return fmt.Sprintf("{%s}", strings.Join(quoted, ","))
}

func (g *Generator) getTestMethods(bulkCfg *BulkInsertConfig) string {
	if bulkCfg.TestInsertMethod {
		return "INSERT and COPY"
	}
	return "INSERT only"
}

func (g *Generator) getTestMethodsList(bulkCfg *BulkInsertConfig) []string {
	if bulkCfg.TestInsertMethod {
		return []string{"insert", "copy"}
	}
	return []string{"insert"}
}

func (g *Generator) calculateWorkers(cfg *types.Config, progress float64) int {
	if !cfg.Progressive.Enabled {
		return cfg.Workers
	}

	min := float64(cfg.Progressive.MinWorkers)
	max := float64(cfg.Progressive.MaxWorkers)
	return int(min + (max-min)*progress)
}

func (g *Generator) calculateConnections(cfg *types.Config, progress float64) int {
	if !cfg.Progressive.Enabled {
		return cfg.Connections
	}

	min := float64(cfg.Progressive.MinConns)
	max := float64(cfg.Progressive.MaxConns)
	return int(min + (max-min)*progress)
}
