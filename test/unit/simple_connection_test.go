// Package unit provides unit tests for StormDB components.
// This file contains tests for the connection overhead workload functionality.
package unit

import (
	"sync"
	"testing"

	"github.com/elchinoo/stormdb/pkg/types"
)

// TestConnectionModeMetrics tests the ConnectionModeMetrics functionality
func TestConnectionModeMetrics(t *testing.T) {
	metrics := &types.ConnectionModeMetrics{}

	// Test basic transaction recording
	metrics.Mu.Lock()
	metrics.TPS = 100
	metrics.TPSAborted = 5
	metrics.QPS = 300
	metrics.Errors = 2
	metrics.Mu.Unlock()

	// Verify values
	metrics.Mu.RLock()
	if metrics.TPS != 100 {
		t.Errorf("Expected TPS 100, got %d", metrics.TPS)
	}
	if metrics.TPSAborted != 5 {
		t.Errorf("Expected TPSAborted 5, got %d", metrics.TPSAborted)
	}
	if metrics.QPS != 300 {
		t.Errorf("Expected QPS 300, got %d", metrics.QPS)
	}
	if metrics.Errors != 2 {
		t.Errorf("Expected Errors 2, got %d", metrics.Errors)
	}
	metrics.Mu.RUnlock()
}

// TestConnectionModeMetricsConcurrency tests concurrent access to ConnectionModeMetrics
func TestConnectionModeMetricsConcurrency(t *testing.T) {
	metrics := &types.ConnectionModeMetrics{}
	var wg sync.WaitGroup

	// Start multiple goroutines updating metrics
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				metrics.Mu.Lock()
				metrics.TPS++
				metrics.QPS++
				metrics.TransactionDur = append(metrics.TransactionDur, int64(j*1000))
				metrics.Mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Verify final values
	metrics.Mu.RLock()
	expectedTPS := int64(1000) // 10 goroutines * 100 increments
	expectedQPS := int64(1000)
	expectedDurCount := 1000

	if metrics.TPS != expectedTPS {
		t.Errorf("Expected TPS %d, got %d", expectedTPS, metrics.TPS)
	}
	if metrics.QPS != expectedQPS {
		t.Errorf("Expected QPS %d, got %d", expectedQPS, metrics.QPS)
	}
	if len(metrics.TransactionDur) != expectedDurCount {
		t.Errorf("Expected %d transaction durations, got %d", expectedDurCount, len(metrics.TransactionDur))
	}
	metrics.Mu.RUnlock()
}

// TestMetricsConnectionModeRecording tests the metrics recording for different connection modes
func TestMetricsConnectionModeRecording(t *testing.T) {
	metrics := &types.Metrics{}

	// Test persistent connection recording
	metrics.RecordConnectionModeTransaction("persistent", true, 1000000) // 1ms
	metrics.RecordConnectionModeQuery("persistent")
	metrics.RecordConnectionModeError("persistent")

	// Test transient connection recording
	metrics.RecordConnectionModeTransaction("transient", false, 2000000) // 2ms
	metrics.RecordConnectionModeQuery("transient")
	metrics.RecordConnectionSetup(500000) // 0.5ms connection setup time

	// Verify persistent metrics
	if metrics.PersistentConnMetrics == nil {
		t.Fatal("PersistentConnMetrics should be initialized")
	}

	metrics.PersistentConnMetrics.Mu.RLock()
	if metrics.PersistentConnMetrics.TPS != 1 {
		t.Errorf("Expected persistent TPS 1, got %d", metrics.PersistentConnMetrics.TPS)
	}
	if metrics.PersistentConnMetrics.QPS != 1 {
		t.Errorf("Expected persistent QPS 1, got %d", metrics.PersistentConnMetrics.QPS)
	}
	if metrics.PersistentConnMetrics.Errors != 1 {
		t.Errorf("Expected persistent Errors 1, got %d", metrics.PersistentConnMetrics.Errors)
	}
	metrics.PersistentConnMetrics.Mu.RUnlock()

	// Verify transient metrics
	if metrics.TransientConnMetrics == nil {
		t.Fatal("TransientConnMetrics should be initialized")
	}

	metrics.TransientConnMetrics.Mu.RLock()
	if metrics.TransientConnMetrics.TPSAborted != 1 {
		t.Errorf("Expected transient TPSAborted 1, got %d", metrics.TransientConnMetrics.TPSAborted)
	}
	if metrics.TransientConnMetrics.QPS != 1 {
		t.Errorf("Expected transient QPS 1, got %d", metrics.TransientConnMetrics.QPS)
	}
	if metrics.TransientConnMetrics.ConnectionCount != 1 {
		t.Errorf("Expected transient ConnectionCount 1, got %d", metrics.TransientConnMetrics.ConnectionCount)
	}
	if len(metrics.TransientConnMetrics.ConnectionSetup) != 1 {
		t.Errorf("Expected 1 connection setup time, got %d", len(metrics.TransientConnMetrics.ConnectionSetup))
	}
	if metrics.TransientConnMetrics.ConnectionSetup[0] != 500000 {
		t.Errorf("Expected connection setup time 500000ns, got %d", metrics.TransientConnMetrics.ConnectionSetup[0])
	}
	metrics.TransientConnMetrics.Mu.RUnlock()
}

// TestConnectionModeUnknown tests handling of unknown connection modes
func TestConnectionModeUnknown(t *testing.T) {
	metrics := &types.Metrics{}

	// These calls should not panic or create metrics for unknown modes
	metrics.RecordConnectionModeTransaction("unknown", true, 1000000)
	metrics.RecordConnectionModeQuery("unknown")
	metrics.RecordConnectionModeError("unknown")

	// Verify no metrics were created
	if metrics.PersistentConnMetrics != nil {
		t.Error("PersistentConnMetrics should not be initialized for unknown mode")
	}
	if metrics.TransientConnMetrics != nil {
		t.Error("TransientConnMetrics should not be initialized for unknown mode")
	}
}

// BenchmarkConnectionModeMetrics benchmarks the performance of connection mode metrics
func BenchmarkConnectionModeMetrics(b *testing.B) {
	metrics := &types.Metrics{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			metrics.RecordConnectionModeTransaction("persistent", true, 1000000)
			metrics.RecordConnectionModeQuery("persistent")
		}
	})
}

// TestConnectionSetupTimeMeasurement tests connection setup time measurement
func TestConnectionSetupTimeMeasurement(t *testing.T) {
	metrics := &types.Metrics{}

	setupTimes := []int64{100000, 200000, 150000, 300000} // Various setup times in ns

	for _, setupTime := range setupTimes {
		metrics.RecordConnectionSetup(setupTime)
	}

	if metrics.TransientConnMetrics == nil {
		t.Fatal("TransientConnMetrics should be initialized")
	}

	metrics.TransientConnMetrics.Mu.RLock()
	if int64(len(metrics.TransientConnMetrics.ConnectionSetup)) != int64(len(setupTimes)) {
		t.Errorf("Expected %d setup times, got %d", len(setupTimes), len(metrics.TransientConnMetrics.ConnectionSetup))
	}

	for i, expectedTime := range setupTimes {
		if metrics.TransientConnMetrics.ConnectionSetup[i] != expectedTime {
			t.Errorf("Expected setup time %d at index %d, got %d", expectedTime, i, metrics.TransientConnMetrics.ConnectionSetup[i])
		}
	}

	if metrics.TransientConnMetrics.ConnectionCount != int64(len(setupTimes)) {
		t.Errorf("Expected connection count %d, got %d", len(setupTimes), metrics.TransientConnMetrics.ConnectionCount)
	}
	metrics.TransientConnMetrics.Mu.RUnlock()
}
