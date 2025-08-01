// internal/workload/bulk_insert/ring_buffer.go
package main

import (
	"context"
	"sync/atomic"
	"time"
)

// DataRecord represents a single record for bulk insertion
type DataRecord struct {
	ShortText    string
	MediumText   string
	LongText     string
	IntValue     int32
	BigintValue  int64
	DecimalValue float64
	FloatValue   float64
	EventDate    time.Time
	EventTime    time.Time
	IsActive     bool
	Metadata     map[string]interface{}
	DataBlob     []byte
	StatusEnum   string
	Tags         []string
	ClientIP     string
	LocationX    float64
	LocationY    float64
}

// RingBuffer implements a lock-free ring buffer for producer-consumer pattern
// Optimized for high-throughput bulk insert operations
type RingBuffer struct {
	buffer   []DataRecord
	capacity int64
	mask     int64 // capacity - 1, for fast modulo using bitwise AND

	// Atomic counters for lock-free operations
	writeIndex int64 // Next position to write
	readIndex  int64 // Next position to read

	// Write completion flags - one per buffer slot
	writeComplete []int32 // 1 when write is complete, 0 when in progress

	// Statistics
	produced   int64 // Total records produced
	consumed   int64 // Total records consumed
	waitTimeNs int64 // Total nanoseconds spent waiting

	// Control flags
	closed int32 // Set to 1 when no more data will be produced
}

// NewRingBuffer creates a new ring buffer with the specified capacity
// Capacity must be a power of 2 for efficient modulo operations
func NewRingBuffer(capacity int) *RingBuffer {
	// Ensure capacity is a power of 2
	if capacity <= 0 || (capacity&(capacity-1)) != 0 {
		// Round up to next power of 2
		cap := 1
		for cap < capacity {
			cap <<= 1
		}
		capacity = cap
	}

	return &RingBuffer{
		buffer:        make([]DataRecord, capacity),
		capacity:      int64(capacity),
		mask:          int64(capacity - 1),
		writeComplete: make([]int32, capacity),
	}
}

// Push adds a record to the buffer (producer operation)
// Returns true if successful, false if buffer is full
func (rb *RingBuffer) Push(record DataRecord) bool {
	for {
		writeIdx := atomic.LoadInt64(&rb.writeIndex)
		readIdx := atomic.LoadInt64(&rb.readIndex)

		// Check if buffer is full
		if writeIdx-readIdx >= rb.capacity {
			return false
		}

		// Try to claim the write position
		if atomic.CompareAndSwapInt64(&rb.writeIndex, writeIdx, writeIdx+1) {
			// Successfully claimed position
			pos := writeIdx & rb.mask

			// Mark as write in progress (not complete)
			atomic.StoreInt32(&rb.writeComplete[pos], 0)

			// Write the data
			rb.buffer[pos] = record

			// Mark write as complete
			atomic.StoreInt32(&rb.writeComplete[pos], 1)

			atomic.AddInt64(&rb.produced, 1)
			return true
		}
		// Failed to claim, retry
	}
}

// Pop removes a record from the buffer (consumer operation)
// Returns the record and true if successful, zero record and false if buffer is empty
func (rb *RingBuffer) Pop() (DataRecord, bool) {
	for {
		readIdx := atomic.LoadInt64(&rb.readIndex)
		writeIdx := atomic.LoadInt64(&rb.writeIndex)

		// Check if buffer is empty
		if readIdx >= writeIdx {
			return DataRecord{}, false
		}

		// Check if the write at this position is complete
		pos := readIdx & rb.mask
		if atomic.LoadInt32(&rb.writeComplete[pos]) == 0 {
			// Write not complete yet, wait briefly and retry
			time.Sleep(time.Nanosecond * 10)
			continue
		}

		// Try to claim the read position
		if atomic.CompareAndSwapInt64(&rb.readIndex, readIdx, readIdx+1) {
			// Successfully claimed position, read the data
			record := rb.buffer[pos]

			// Mark slot as available for reuse (optional optimization)
			atomic.StoreInt32(&rb.writeComplete[pos], 0)

			atomic.AddInt64(&rb.consumed, 1)
			return record, true
		}
		// Failed to claim, retry
	}
}

// PopBatch removes up to maxRecords from the buffer
// Returns slice of records and the actual count retrieved
func (rb *RingBuffer) PopBatch(maxRecords int) ([]DataRecord, int) {
	records := make([]DataRecord, 0, maxRecords)

	for len(records) < maxRecords {
		record, ok := rb.Pop()
		if !ok {
			break
		}
		records = append(records, record)
	}

	return records, len(records)
}

// PopBatchBlocking waits for records to become available and returns a batch
// Context can be used to cancel the operation
func (rb *RingBuffer) PopBatchBlocking(ctx context.Context, minRecords, maxRecords int, timeout time.Duration) ([]DataRecord, error) {
	start := time.Now()
	deadline := start.Add(timeout)
	records := make([]DataRecord, 0, maxRecords)

	for len(records) < minRecords {
		select {
		case <-ctx.Done():
			return records, ctx.Err()
		default:
		}

		// Check if we've hit the timeout
		if time.Now().After(deadline) {
			break
		}

		// Try to get more records
		batch, count := rb.PopBatch(maxRecords - len(records))
		records = append(records, batch...)

		// If we got some records but not enough, sleep briefly before retrying
		if count == 0 && len(records) < minRecords {
			time.Sleep(time.Microsecond * 100)      // Brief sleep to avoid busy waiting
			atomic.AddInt64(&rb.waitTimeNs, 100000) // 100 microseconds in nanoseconds
		}
	}

	return records, nil
}

// IsClosed returns true if the buffer is closed for writing
func (rb *RingBuffer) IsClosed() bool {
	return atomic.LoadInt32(&rb.closed) == 1
}

// Close marks the buffer as closed for writing
func (rb *RingBuffer) Close() {
	atomic.StoreInt32(&rb.closed, 1)
}

// Size returns the current number of items in the buffer
func (rb *RingBuffer) Size() int64 {
	writeIdx := atomic.LoadInt64(&rb.writeIndex)
	readIdx := atomic.LoadInt64(&rb.readIndex)
	return writeIdx - readIdx
}

// Capacity returns the maximum capacity of the buffer
func (rb *RingBuffer) Capacity() int64 {
	return rb.capacity
}

// Stats returns statistics about buffer usage
func (rb *RingBuffer) Stats() (produced, consumed, waitTimeNs int64, utilization float64) {
	produced = atomic.LoadInt64(&rb.produced)
	consumed = atomic.LoadInt64(&rb.consumed)
	waitTimeNs = atomic.LoadInt64(&rb.waitTimeNs)

	currentSize := rb.Size()
	utilization = float64(currentSize) / float64(rb.capacity)

	return produced, consumed, waitTimeNs, utilization
}

// Reset clears the buffer and resets all counters
func (rb *RingBuffer) Reset() {
	atomic.StoreInt64(&rb.writeIndex, 0)
	atomic.StoreInt64(&rb.readIndex, 0)
	atomic.StoreInt64(&rb.produced, 0)
	atomic.StoreInt64(&rb.consumed, 0)
	atomic.StoreInt64(&rb.waitTimeNs, 0)
	atomic.StoreInt32(&rb.closed, 0)
}
