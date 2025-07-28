package unit_test

import (
	"testing"

	"github.com/elchinoo/stormdb/pkg/types"
)

func TestLatencyHistogram(t *testing.T) {
	metrics := &types.Metrics{}
	metrics.InitializeLatencyHistogram()

	// Test bucket initialization
	if len(metrics.LatencyHistogram) == 0 {
		t.Error("Histogram should be initialized with buckets")
	}

	// Test recording latencies
	testCases := []struct {
		name           string
		latencyNs      int64
		expectedBucket string
	}{
		{"50 microseconds", 50000, "0.1ms"},
		{"0.3 milliseconds", 300000, "0.5ms"},
		{"0.8 milliseconds", 800000, "1.0ms"},
		{"1.5 milliseconds", 1500000, "2.0ms"},
		{"8 milliseconds", 8000000, "10.0ms"},
		{"150 milliseconds", 150000000, "200.0ms"},
		{"2 seconds", 2000000000, "+inf"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bucket := types.GetLatencyBucket(tc.latencyNs)
			if bucket != tc.expectedBucket {
				t.Errorf("Expected bucket %s, got %s for latency %dns", tc.expectedBucket, bucket, tc.latencyNs)
			}

			// Test recording
			initialCount := metrics.LatencyHistogram[bucket]
			metrics.RecordLatency(tc.latencyNs)
			newCount := metrics.LatencyHistogram[bucket]

			if newCount != initialCount+1 {
				t.Errorf("Expected count to increase by 1, got %d -> %d", initialCount, newCount)
			}
		})
	}
}

func TestGetLatencyBucket(t *testing.T) {
	testCases := []struct {
		latencyNs int64
		expected  string
	}{
		{50000, "0.1ms"},     // 0.05ms -> 0.1ms bucket
		{200000, "0.5ms"},    // 0.2ms -> 0.5ms bucket
		{750000, "1.0ms"},    // 0.75ms -> 1.0ms bucket
		{1800000, "2.0ms"},   // 1.8ms -> 2.0ms bucket
		{25000000, "50.0ms"}, // 25ms -> 50.0ms bucket
		{2000000000, "+inf"}, // 2000ms -> +inf bucket
	}

	for _, tc := range testCases {
		result := types.GetLatencyBucket(tc.latencyNs)
		if result != tc.expected {
			t.Errorf("GetLatencyBucket(%d) = %s, expected %s", tc.latencyNs, result, tc.expected)
		}
	}
}

func TestMetricsInitialization(t *testing.T) {
	metrics := &types.Metrics{}

	// Test that histogram is nil initially
	if metrics.LatencyHistogram != nil {
		t.Error("LatencyHistogram should be nil initially")
	}

	// Initialize histogram
	metrics.InitializeLatencyHistogram()

	// Test that all buckets are initialized
	expectedBuckets := []string{"0.1ms", "0.5ms", "1.0ms", "2.0ms", "5.0ms", "10.0ms", "20.0ms", "50.0ms", "100.0ms", "200.0ms", "500.0ms", "1000.0ms", "+inf"}

	for _, bucket := range expectedBuckets {
		if _, exists := metrics.LatencyHistogram[bucket]; !exists {
			t.Errorf("Bucket %s should be initialized", bucket)
		}

		if metrics.LatencyHistogram[bucket] != 0 {
			t.Errorf("Bucket %s should be initialized to 0, got %d", bucket, metrics.LatencyHistogram[bucket])
		}
	}
}
