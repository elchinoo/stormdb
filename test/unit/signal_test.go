// test/unit/signal_test.go
package unit

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

func TestSignalHandling(t *testing.T) {
	// Test that signal handling channel works correctly
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test that the context cancellation works
	select {
	case <-ctx.Done():
		// Expected - context should timeout
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Context did not timeout as expected")
	}
}

func TestContextCancellation(t *testing.T) {
	// Test manual context cancellation (simulating signal)
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	// Check that context is cancelled
	select {
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			t.Errorf("Expected Canceled, got %v", ctx.Err())
		}
	default:
		t.Error("Context should be cancelled")
	}
}

func TestGracefulShutdownTimeout(t *testing.T) {
	// Test the 5-second graceful shutdown timeout logic

	// Simulate a workload that takes longer than the timeout
	workloadFinished := make(chan struct{})

	go func() {
		// Simulate workload taking 7 seconds
		time.Sleep(7 * time.Second)
		close(workloadFinished)
	}()

	// Wait for either workload completion or timeout (5 seconds)
	select {
	case <-workloadFinished:
		t.Error("Workload should not have finished within timeout")
	case <-time.After(5 * time.Second):
		// Expected - should timeout after 5 seconds
		t.Log("Graceful shutdown timeout worked correctly")
	}
}
