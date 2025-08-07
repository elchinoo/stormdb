package main

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"time"
)

// MetricRecord represents a single metric record from a thread
type MetricRecord struct {
	ThreadID       int       `json:"thread_id"`
	Timestamp      time.Time `json:"timestamp"`
	DtStarted      time.Time `json:"dt_started"`
	DtEnd          time.Time `json:"dt_end"`
	NumConnections int       `json:"num_connections"`
	NumInsert      int64     `json:"num_insert"`
	NumUpdate      int64     `json:"num_update"`
	NumDelete      int64     `json:"num_delete"`
	NumSelect      int64     `json:"num_select"`
	LatencySum     int64     `json:"latency_sum"` // nanoseconds
	LatencyCount   int64     `json:"latency_count"`
	NumRowInsert   int64     `json:"num_row_insert"`
	NumRowUpdate   int64     `json:"num_row_update"`
	NumRowDelete   int64     `json:"num_row_delete"`
	NumRowSelect   int64     `json:"num_row_select"`
	NumErrors      int64     `json:"num_errors"`
}

// ThreadMetricsBuffer maintains metrics for a single thread with fixed-size buffer
type ThreadMetricsBuffer struct {
	threadID      int
	buffer        []MetricRecord
	bufferSize    int
	currentIndex  int
	lastFlush     time.Time
	flushInterval time.Duration
	metricsAPI    *MetricsAPI

	// Current accumulating values
	dtStarted      time.Time
	dtEnd          time.Time
	numConnections int
	numInsert      int64
	numUpdate      int64
	numDelete      int64
	numSelect      int64
	latencySum     int64
	latencyCount   int64
	numRowInsert   int64
	numRowUpdate   int64
	numRowDelete   int64
	numRowSelect   int64
	numErrors      int64

	mutex sync.Mutex
	ctx   context.Context
	done  chan struct{}
}

// MetricsAPI handles the queue and database persistence
type MetricsAPI struct {
	queue     chan MetricRecord
	queueSize int
	db        *sql.DB
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning int64
}

// PerformanceMetrics maintains thread buffers and coordinates metrics collection
type PerformanceMetrics struct {
	metricsAPI     *MetricsAPI
	threadBuffers  map[int]*ThreadMetricsBuffer
	buffersMutex   sync.RWMutex
	nextThreadID   int32
	testStartTime  time.Time
	numConnections int32
}

// NewMetricsAPI creates a new metrics API instance
func NewMetricsAPI(db *sql.DB, queueSize int) *MetricsAPI {
	ctx, cancel := context.WithCancel(context.Background())

	api := &MetricsAPI{
		queue:     make(chan MetricRecord, queueSize),
		queueSize: queueSize,
		db:        db,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start the database persistence worker
	api.wg.Add(1)
	go api.persistenceWorker()
	atomic.StoreInt64(&api.isRunning, 1)

	return api
}

// Stop gracefully shuts down the metrics API
func (api *MetricsAPI) Stop() {
	if !atomic.CompareAndSwapInt64(&api.isRunning, 1, 0) {
		return // already stopped
	}

	api.cancel()
	close(api.queue)
	api.wg.Wait()
}

// QueueMetric queues a metric record for database persistence
func (api *MetricsAPI) QueueMetric(record MetricRecord) bool {
	select {
	case api.queue <- record:
		return true
	case <-api.ctx.Done():
		return false
	default:
		// Queue is full, drop the metric (or implement backpressure)
		return false
	}
}

// persistenceWorker runs in a background goroutine to persist queued metrics
func (api *MetricsAPI) persistenceWorker() {
	defer api.wg.Done()

	ticker := time.NewTicker(500 * time.Millisecond) // Flush twice per second
	defer ticker.Stop()

	var batch []MetricRecord
	maxBatchSize := 50 // Maximum records to batch together

	for {
		select {
		case record, ok := <-api.queue:
			if !ok {
				// Channel closed, flush remaining batch and exit
				if len(batch) > 0 {
					api.persistBatch(batch)
				}
				return
			}

			batch = append(batch, record)

			// Flush if batch is full
			if len(batch) >= maxBatchSize {
				api.persistBatch(batch)
				batch = batch[:0] // Reset slice
			}

		case <-ticker.C:
			// Periodic flush (twice per second)
			if len(batch) > 0 {
				api.persistBatch(batch)
				batch = batch[:0] // Reset slice
			}

		case <-api.ctx.Done():
			// Graceful shutdown, flush remaining batch
			if len(batch) > 0 {
				api.persistBatch(batch)
			}
			return
		}
	}
}

// persistBatch persists a batch of metric records to the database
func (api *MetricsAPI) persistBatch(records []MetricRecord) {
	if len(records) == 0 || api.db == nil {
		return
	}

	// Insert raw metric records into database
	tx, err := api.db.Begin()
	if err != nil {
		return // Log error in production
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO metric_records (
			thread_id, timestamp, dt_started, dt_end, num_connections,
			num_insert, num_update, num_delete, num_select,
			latency_sum, latency_count, num_row_insert, num_row_update,
			num_row_delete, num_row_select, num_errors
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`)
	if err != nil {
		return // Log error in production
	}
	defer stmt.Close()

	for _, record := range records {
		_, err = stmt.Exec(
			record.ThreadID, record.Timestamp, record.DtStarted, record.DtEnd,
			record.NumConnections, record.NumInsert, record.NumUpdate,
			record.NumDelete, record.NumSelect, record.LatencySum,
			record.LatencyCount, record.NumRowInsert, record.NumRowUpdate,
			record.NumRowDelete, record.NumRowSelect, record.NumErrors,
		)
		if err != nil {
			return // Log error in production
		}
	}

	tx.Commit()
}

// NewThreadMetricsBuffer creates a new thread metrics buffer
func NewThreadMetricsBuffer(threadID int, bufferSize int, flushInterval time.Duration, metricsAPI *MetricsAPI) *ThreadMetricsBuffer {
	ctx, cancel := context.WithCancel(context.Background())

	buffer := &ThreadMetricsBuffer{
		threadID:      threadID,
		buffer:        make([]MetricRecord, bufferSize),
		bufferSize:    bufferSize,
		currentIndex:  0,
		lastFlush:     time.Now(),
		flushInterval: flushInterval,
		metricsAPI:    metricsAPI,
		ctx:           ctx,
		done:          make(chan struct{}),
	}

	// Start periodic flush goroutine
	go buffer.periodicFlush(cancel)

	return buffer
}

// periodicFlush runs periodic flushing in a background goroutine
func (tmb *ThreadMetricsBuffer) periodicFlush(cancel context.CancelFunc) {
	defer cancel()
	defer close(tmb.done)

	ticker := time.NewTicker(tmb.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tmb.flushIfNeeded()
		case <-tmb.ctx.Done():
			// Final flush on shutdown
			tmb.forceFlush()
			return
		}
	}
}

// RecordOperation records a single database operation
func (tmb *ThreadMetricsBuffer) RecordOperation(operation string, latency time.Duration, rowsAffected int64, isError bool) {
	tmb.mutex.Lock()
	defer tmb.mutex.Unlock()

	// Update accumulating counters
	tmb.latencySum += latency.Nanoseconds()
	tmb.latencyCount++

	switch operation {
	case "insert":
		tmb.numInsert++
		tmb.numRowInsert += rowsAffected
	case "update":
		tmb.numUpdate++
		tmb.numRowUpdate += rowsAffected
	case "delete":
		tmb.numDelete++
		tmb.numRowDelete += rowsAffected
	case "select":
		tmb.numSelect++
		tmb.numRowSelect += rowsAffected
	}

	if isError {
		tmb.numErrors++
	}

	// Check if buffer should be flushed
	if time.Since(tmb.lastFlush) >= tmb.flushInterval {
		tmb.flushUnsafe()
	}
}

// SetTestBounds sets the test start/end times and connection count
func (tmb *ThreadMetricsBuffer) SetTestBounds(started time.Time, ended time.Time, connections int) {
	tmb.mutex.Lock()
	defer tmb.mutex.Unlock()

	tmb.dtStarted = started
	tmb.dtEnd = ended
	tmb.numConnections = connections
}

// flushIfNeeded flushes the buffer if the flush interval has elapsed
func (tmb *ThreadMetricsBuffer) flushIfNeeded() {
	tmb.mutex.Lock()
	defer tmb.mutex.Unlock()

	if time.Since(tmb.lastFlush) >= tmb.flushInterval {
		tmb.flushUnsafe()
	}
}

// forceFlush forces an immediate flush
func (tmb *ThreadMetricsBuffer) forceFlush() {
	tmb.mutex.Lock()
	defer tmb.mutex.Unlock()

	tmb.flushUnsafe()
}

// flushUnsafe flushes current metrics to the API (caller must hold mutex)
func (tmb *ThreadMetricsBuffer) flushUnsafe() {
	if tmb.latencyCount == 0 {
		return // Nothing to flush
	}

	// Create metric record from current accumulated values
	record := MetricRecord{
		ThreadID:       tmb.threadID,
		Timestamp:      time.Now(),
		DtStarted:      tmb.dtStarted,
		DtEnd:          tmb.dtEnd,
		NumConnections: tmb.numConnections,
		NumInsert:      tmb.numInsert,
		NumUpdate:      tmb.numUpdate,
		NumDelete:      tmb.numDelete,
		NumSelect:      tmb.numSelect,
		LatencySum:     tmb.latencySum,
		LatencyCount:   tmb.latencyCount,
		NumRowInsert:   tmb.numRowInsert,
		NumRowUpdate:   tmb.numRowUpdate,
		NumRowDelete:   tmb.numRowDelete,
		NumRowSelect:   tmb.numRowSelect,
		NumErrors:      tmb.numErrors,
	}

	// Queue the record (non-blocking)
	tmb.metricsAPI.QueueMetric(record)

	// Reset accumulated values
	tmb.numInsert = 0
	tmb.numUpdate = 0
	tmb.numDelete = 0
	tmb.numSelect = 0
	tmb.latencySum = 0
	tmb.latencyCount = 0
	tmb.numRowInsert = 0
	tmb.numRowUpdate = 0
	tmb.numRowDelete = 0
	tmb.numRowSelect = 0
	tmb.numErrors = 0
	tmb.lastFlush = time.Now()
}

// Stop gracefully stops the thread buffer
func (tmb *ThreadMetricsBuffer) Stop() {
	if tmb.ctx.Err() != nil {
		return // Already stopped
	}

	// Signal shutdown and wait for completion
	select {
	case <-tmb.ctx.Done():
	default:
		// Context will be cancelled by periodicFlush goroutine
	}

	<-tmb.done
}

// NewPerformanceMetrics creates a new performance metrics coordinator
func NewPerformanceMetrics(db *sql.DB) *PerformanceMetrics {
	metricsAPI := NewMetricsAPI(db, 1000) // Queue up to 1000 records

	return &PerformanceMetrics{
		metricsAPI:    metricsAPI,
		threadBuffers: make(map[int]*ThreadMetricsBuffer),
		testStartTime: time.Now(),
	}
}

// Start records the test start time and connection count
func (pm *PerformanceMetrics) Start(connections int) {
	pm.testStartTime = time.Now()
	atomic.StoreInt32(&pm.numConnections, int32(connections))

	// Update all existing thread buffers with test bounds
	pm.buffersMutex.RLock()
	for _, buffer := range pm.threadBuffers {
		buffer.SetTestBounds(pm.testStartTime, time.Time{}, connections)
	}
	pm.buffersMutex.RUnlock()
}

// End records the test end time
func (pm *PerformanceMetrics) End() {
	endTime := time.Now()
	connections := int(atomic.LoadInt32(&pm.numConnections))

	// Update all existing thread buffers with end time
	pm.buffersMutex.RLock()
	for _, buffer := range pm.threadBuffers {
		buffer.SetTestBounds(pm.testStartTime, endTime, connections)
		buffer.forceFlush() // Final flush
	}
	pm.buffersMutex.RUnlock()
}

// RegisterWorker creates a new thread metrics buffer
func (pm *PerformanceMetrics) RegisterWorker() int {
	threadID := int(atomic.AddInt32(&pm.nextThreadID, 1))
	connections := int(atomic.LoadInt32(&pm.numConnections))

	// Create buffer with 100 record capacity, flush every 500ms
	buffer := NewThreadMetricsBuffer(threadID, 100, 500*time.Millisecond, pm.metricsAPI)
	buffer.SetTestBounds(pm.testStartTime, time.Time{}, connections)

	pm.buffersMutex.Lock()
	pm.threadBuffers[threadID] = buffer
	pm.buffersMutex.Unlock()

	return threadID
}

// UnregisterWorker removes and stops a thread buffer
func (pm *PerformanceMetrics) UnregisterWorker(threadID int) {
	pm.buffersMutex.Lock()
	buffer, exists := pm.threadBuffers[threadID]
	if exists {
		delete(pm.threadBuffers, threadID)
	}
	pm.buffersMutex.Unlock()

	if exists {
		buffer.Stop()
	}
}

// RecordOperation records an operation for a specific thread
func (pm *PerformanceMetrics) RecordOperation(threadID int, operation string, latency time.Duration, rowsAffected int64, isError bool) {
	pm.buffersMutex.RLock()
	buffer, exists := pm.threadBuffers[threadID]
	pm.buffersMutex.RUnlock()

	if exists {
		buffer.RecordOperation(operation, latency, rowsAffected, isError)
	}
}

// Stop gracefully stops the performance metrics system
func (pm *PerformanceMetrics) Stop() {
	// Stop all thread buffers
	pm.buffersMutex.Lock()
	for threadID, buffer := range pm.threadBuffers {
		buffer.Stop()
		delete(pm.threadBuffers, threadID)
	}
	pm.buffersMutex.Unlock()

	// Stop the metrics API
	pm.metricsAPI.Stop()
}

// MetricsSnapshot represents aggregated metrics for reporting (legacy compatibility)
type MetricsSnapshot struct {
	DtStarted      time.Time
	DtEnd          time.Time
	NumConnections int
	NumInsert      int64
	NumUpdate      int64
	NumDelete      int64
	NumSelect      int64
	LatencySum     int64 // nanoseconds
	LatencyCount   int64
	NumRowInsert   int64
	NumRowUpdate   int64
	NumRowDelete   int64
	NumRowSelect   int64
	NumErrors      int64
}

// Snapshot returns a dummy snapshot for legacy compatibility
// Note: In the new design, metrics are stored as raw records, not aggregated
func (pm *PerformanceMetrics) Snapshot() MetricsSnapshot {
	connections := int(atomic.LoadInt32(&pm.numConnections))

	return MetricsSnapshot{
		DtStarted:      pm.testStartTime,
		DtEnd:          time.Now(),
		NumConnections: connections,
		// Note: Individual metrics would need to be queried from database
		// This is just a placeholder for legacy compatibility
	}
}

// SnapshotAndReset is a no-op in the new design (legacy compatibility)
func (pm *PerformanceMetrics) SnapshotAndReset() MetricsSnapshot {
	return pm.Snapshot()
}

// Reset is a no-op in the new design (legacy compatibility)
func (pm *PerformanceMetrics) Reset() {
	// Individual thread buffers handle their own lifecycle
}

// Legacy compatibility methods for MetricsSnapshot
func (s MetricsSnapshot) TotalOperations() int64 {
	return s.NumInsert + s.NumUpdate + s.NumDelete + s.NumSelect
}

func (s MetricsSnapshot) TotalRows() int64 {
	return s.NumRowInsert + s.NumRowUpdate + s.NumRowDelete + s.NumRowSelect
}

func (s MetricsSnapshot) AverageLatencyMS() float64 {
	if s.LatencyCount == 0 {
		return 0
	}
	return float64(s.LatencySum) / float64(s.LatencyCount) / 1e6
}

func (s MetricsSnapshot) OperationsPerSecond() float64 {
	if s.DtEnd.IsZero() || s.DtStarted.IsZero() {
		return 0
	}
	duration := s.DtEnd.Sub(s.DtStarted).Seconds()
	if duration <= 0 {
		return 0
	}
	return float64(s.TotalOperations()) / duration
}

func (s MetricsSnapshot) ErrorRate() float64 {
	total := s.TotalOperations() + s.NumErrors
	if total == 0 {
		return 0
	}
	return float64(s.NumErrors) / float64(total) * 100
}
