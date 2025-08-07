package main

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"
)

// TestCleanupStopsGoroutines verifies that Stop() cleans up running workload goroutines promptly
func TestCleanupStopsGoroutines(t *testing.T) {
	// Setup plugin with slow ExecTx
	p := &TPCCPlugin{}
	cfg := NewTPCCConfig()
	cfg.Connections = []int{2}
	cfg.ThinkTime = 20 * time.Millisecond
	cfg.StopOnErrorLimit = false
	p.cfg = cfg
	p.stopChan = make(chan struct{})
	p.wg = sync.WaitGroup{}
	p.metrics = &TPCCMetrics{}
	p.stats = &Stats{}
	// override ExecTx to simulate long-running transaction
	p.ExecTx = func(db *sql.DB, txType string, warehouseID int) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	// Run workload in background
	done := make(chan struct{})
	go func() {
		p.runWorkload(context.Background(), len(cfg.Connections), cfg.ThinkTime)
		close(done)
	}()

	// Allow workload to start
	time.Sleep(50 * time.Millisecond)

	// Signal stop
	start := time.Now()
	if err := p.Stop(); err != nil {
		t.Errorf("Stop returned error: %v", err)
	}

	// Wait for runWorkload to exit
	select {
	case <-done:
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Error("runWorkload did not exit after Stop() within expected time")
	}
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Stop took too long: %v", elapsed)
	}
}
