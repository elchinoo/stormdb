// Bulk insert workload generator for high-throughput testing
package main

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
	"github.com/spf13/viper"
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
	// Validate inputs to prevent nil pointer dereferences
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}
	if db == nil {
		return fmt.Errorf("database pool is nil")
	}
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if metrics == nil {
		return fmt.Errorf("metrics is nil")
	}

	// Parse bulk insert specific configuration
	bulkCfg := g.parseBulkInsertConfig(cfg)
	if bulkCfg == nil {
		return fmt.Errorf("failed to parse bulk insert configuration")
	}

	// Initialize workload state
	state := &WorkloadState{
		dataGenerator: NewDataGenerator(bulkCfg.DataSeed),
		ringBuffer:    NewRingBuffer(bulkCfg.RingBufferSize),
	}

	// Validate state components
	if state.dataGenerator == nil {
		return fmt.Errorf("failed to create data generator")
	}
	if state.ringBuffer == nil {
		return fmt.Errorf("failed to create ring buffer")
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
	// Validate inputs
	if state == nil {
		return fmt.Errorf("workload state is nil")
	}
	if bulkCfg == nil {
		return fmt.Errorf("bulk config is nil")
	}

	progressiveCfg := cfg.Progressive

	// Calculate test matrix: methods × batch sizes
	methods := g.getTestMethodsList(bulkCfg)
	batchSizes := bulkCfg.BatchSizes

	if len(methods) == 0 {
		return fmt.Errorf("no test methods configured")
	}
	if len(batchSizes) == 0 {
		return fmt.Errorf("no batch sizes configured")
	}

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
				if err := g.runBand(ctx, db, cfg, metrics, bulkCfg, state, method, batchSize, workers, connections, bandIndex, state.totalBands); err != nil {
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
	// Validate inputs
	if state == nil {
		return fmt.Errorf("workload state is nil")
	}
	if bulkCfg == nil {
		return fmt.Errorf("bulk config is nil")
	}

	// For standard test, test all combinations sequentially
	methods := g.getTestMethodsList(bulkCfg)
	batchSizes := bulkCfg.BatchSizes

	if len(methods) == 0 {
		return fmt.Errorf("no test methods configured")
	}
	if len(batchSizes) == 0 {
		return fmt.Errorf("no batch sizes configured")
	}

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
			if err := g.runBand(ctx, db, cfg, metrics, bulkCfg, state, method, batchSize, cfg.Workers, cfg.Connections, 0, 0); err != nil {
				return fmt.Errorf("test %s/%d failed: %w", method, batchSize, err)
			}
		}
	}

	return nil
}

// runBand executes a single test band
func (g *Generator) runBand(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics, bulkCfg *BulkInsertConfig, state *WorkloadState, method string, batchSize, workers, connections, bandIndex, totalBands int) error {
	// Validate inputs
	if state == nil {
		return fmt.Errorf("workload state is nil")
	}
	if state.ringBuffer == nil {
		return fmt.Errorf("ring buffer is nil")
	}
	if bulkCfg == nil {
		return fmt.Errorf("bulk config is nil")
	}
	if workers <= 0 {
		return fmt.Errorf("invalid worker count: %d", workers)
	}
	if batchSize <= 0 {
		return fmt.Errorf("invalid batch size: %d", batchSize)
	}

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
		if producerDataGen == nil {
			return fmt.Errorf("failed to create producer data generator for thread %d", i)
		}
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
	if totalBands > 0 {
		// Progressive test - include band information
		log.Printf("✅ Band %d/%d completed (%s method, batch size %d): produced=%d, consumed=%d, utilization=%.2f%%, wait_time=%v",
			bandIndex, totalBands, method, batchSize, produced, consumed, utilization*100, time.Duration(waitTime))
	} else {
		// Standard test - include method and batch size information
		log.Printf("✅ Test completed (%s method, batch size %d): produced=%d, consumed=%d, utilization=%.2f%%, wait_time=%v",
			method, batchSize, produced, consumed, utilization*100, time.Duration(waitTime))
	}

	return nil
}

// producer generates data and feeds it into the ring buffer
func (g *Generator) producer(ctx context.Context, state *WorkloadState, bulkCfg *BulkInsertConfig, wg *sync.WaitGroup, dataGen *DataGenerator) {
	defer wg.Done()

	// Validate inputs to prevent panics
	if state == nil {
		log.Printf("❌ Producer error: state is nil")
		return
	}
	if state.ringBuffer == nil {
		log.Printf("❌ Producer error: ring buffer is nil")
		return
	}
	if dataGen == nil {
		log.Printf("❌ Producer error: data generator is nil")
		return
	}

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

	// Validate inputs to prevent panics
	if state == nil {
		log.Printf("❌ Worker %d error: state is nil", workerID)
		return
	}
	if state.ringBuffer == nil {
		log.Printf("❌ Worker %d error: ring buffer is nil", workerID)
		return
	}
	if db == nil {
		log.Printf("❌ Worker %d error: database pool is nil", workerID)
		return
	}
	if metrics == nil {
		log.Printf("❌ Worker %d error: metrics is nil", workerID)
		return
	}

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

		// Create defensive copies to prevent memory corruption issues
		safeRecords := g.copyRecords(records)
		if safeRecords == nil {
			log.Printf("❌ Worker %d error: failed to create safe record copies", workerID)
			continue
		}

		// Perform the insert operation using dbCtx which doesn't expire during test
		start := time.Now()
		var insertErr error

		switch method {
		case "insert":
			insertErr = g.performBatchInsert(dbCtx, db, safeRecords)
		case "copy":
			insertErr = g.performCopyInsert(dbCtx, db, safeRecords)
		default:
			insertErr = fmt.Errorf("unknown insert method: %s", method)
		}

		duration := time.Since(start)

		// Update metrics with proper nil checking
		if insertErr != nil {
			atomic.AddInt64(&metrics.Errors, 1)
			log.Printf("❌ Worker %d insert error: %v", workerID, insertErr)
		} else {
			atomic.AddInt64(&metrics.TPS, 1)
			atomic.AddInt64(&state.totalInserted, int64(len(records)))

			// Record the actual rows inserted for rows-per-second metrics
			atomic.AddInt64(&metrics.RowsModified, int64(len(records)))
			metrics.RecordTimeSeriesQuery("INSERT", int64(len(records)))

			// Record latency with proper storage for percentile calculations
			latencyNs := duration.Nanoseconds()
			metrics.RecordLatencyWithLimit(latencyNs)
		}
	}
}

// copyRecords creates defensive copies of DataRecord slice to prevent memory corruption
func (g *Generator) copyRecords(records []DataRecord) []DataRecord {
	if len(records) == 0 {
		return nil
	}

	copies := make([]DataRecord, len(records))
	for i, record := range records {
		// Create deep copies of slices to prevent corruption
		tagsCopy := make([]string, len(record.Tags))
		copy(tagsCopy, record.Tags) // Simple copy is sufficient for strings

		// Simple assignment is sufficient for strings - Go strings are immutable
		copies[i] = DataRecord{
			ShortText:    record.ShortText,
			MediumText:   record.MediumText,
			LongText:     record.LongText,
			IntValue:     record.IntValue,
			BigintValue:  record.BigintValue,
			DecimalValue: record.DecimalValue,
			FloatValue:   record.FloatValue,
			EventDate:    record.EventDate,
			EventTime:    record.EventTime,
			IsActive:     record.IsActive,
			Metadata:     record.Metadata,
			DataBlob:     record.DataBlob,
			StatusEnum:   record.StatusEnum, // Simple assignment
			Tags:         tagsCopy,
			ClientIP:     record.ClientIP,
			LocationX:    record.LocationX,
			LocationY:    record.LocationY,
		}
	}
	return copies
}

// validateStatusEnumForSQL performs final validation before SQL execution
func (g *Generator) validateStatusEnumForSQL(status string) string {
	// Use explicit comparisons for maximum safety
	switch status {
	case "pending", "processing", "completed", "failed", "cancelled":
		return status
	default:
		// If invalid, log error and return safe default
		log.Printf("❌ Critical: Invalid enum at SQL execution: %q, substituting 'pending'", status)
		return "pending"
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
			g.validateStatusEnumForSQL(record.StatusEnum),
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
			g.validateStatusEnumForSQL(record.StatusEnum), // Pass directly like other string fields
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
	// Create default configuration to prevent nil pointer issues
	bulkCfg := &BulkInsertConfig{
		RingBufferSize:   100000,
		ProducerThreads:  2,
		BatchSizes:       []int{1, 100, 1000, 10000, 50000},
		TestInsertMethod: true,
		DataSeed:         0,
		MaxMemoryMB:      512,
		CollectMetrics:   true,
	}

	// Parse workload_config section from Viper if available
	if viper.IsSet("workload_config") {
		if ringBufferSize := viper.GetInt("workload_config.ring_buffer_size"); ringBufferSize > 0 {
			bulkCfg.RingBufferSize = ringBufferSize
		}
		if producerThreads := viper.GetInt("workload_config.producer_threads"); producerThreads > 0 {
			bulkCfg.ProducerThreads = producerThreads
		}
		if batchSizes := viper.GetIntSlice("workload_config.batch_sizes"); len(batchSizes) > 0 {
			bulkCfg.BatchSizes = batchSizes
		}
		if viper.IsSet("workload_config.test_insert_method") {
			bulkCfg.TestInsertMethod = viper.GetBool("workload_config.test_insert_method")
		}
		if dataSeed := viper.GetInt64("workload_config.data_seed"); dataSeed != 0 {
			bulkCfg.DataSeed = dataSeed
		}
		if maxMemoryMB := viper.GetInt("workload_config.max_memory_mb"); maxMemoryMB > 0 {
			bulkCfg.MaxMemoryMB = maxMemoryMB
		}
		if viper.IsSet("workload_config.collect_metrics") {
			bulkCfg.CollectMetrics = viper.GetBool("workload_config.collect_metrics")
		}

		log.Printf("📊 Parsed workload config: buffer=%d, producers=%d, batch_sizes=%v, methods=%t, seed=%d, memory=%dMB",
			bulkCfg.RingBufferSize, bulkCfg.ProducerThreads, bulkCfg.BatchSizes, bulkCfg.TestInsertMethod, bulkCfg.DataSeed, bulkCfg.MaxMemoryMB)
	}

	// Validate and sanitize configuration
	if bulkCfg.RingBufferSize <= 0 {
		bulkCfg.RingBufferSize = 100000
	}
	if bulkCfg.ProducerThreads <= 0 {
		bulkCfg.ProducerThreads = 2
	}
	if len(bulkCfg.BatchSizes) == 0 {
		bulkCfg.BatchSizes = []int{1, 100, 1000, 10000, 50000}
	}

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
		// Basic safety checks
		if len(tag) == 0 {
			quoted[i] = "\"\""
			continue
		}

		// Ensure tag is reasonable length
		if len(tag) > 1000 {
			tag = tag[:1000] // Truncate very long tags
		}

		// Simple string replacement for quotes
		escaped := strings.ReplaceAll(tag, "\"", "\\\"")
		quoted[i] = "\"" + escaped + "\""
	}
	return fmt.Sprintf("{%s}", strings.Join(quoted, ","))
}

func (g *Generator) getTestMethods(bulkCfg *BulkInsertConfig) string {
	if bulkCfg == nil {
		return "INSERT only"
	}
	if bulkCfg.TestInsertMethod {
		return "INSERT and COPY"
	}
	return "INSERT only"
}

func (g *Generator) getTestMethodsList(bulkCfg *BulkInsertConfig) []string {
	if bulkCfg == nil {
		return []string{"insert"}
	}
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
