package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testPlugin overrides executeTransaction to always return error
type testPlugin struct {
	TPCCPlugin
}

// executeTransaction always errors
func (p *testPlugin) executeTransaction(db *sql.DB, txType string, warehouseID int) error {
	return fmt.Errorf("forced error")
}

func TestErrorLimitStopsWorkload(t *testing.T) {
	// configure plugin with zero max error rate so first error stops all immediately
	cfg := NewTPCCConfig()
	cfg.Connections = []int{5}
	cfg.Duration = 500 * time.Millisecond
	cfg.ThinkTime = 0
	cfg.StopOnErrorLimit = true
	cfg.MaxErrorRate = 0.0

	p := &testPlugin{}
	p.cfg = cfg
	p.stopChan = make(chan struct{})
	p.metrics = &TPCCMetrics{}
	p.stats = &Stats{}
	// use overridden executor to avoid nil DB
	p.ExecTx = p.executeTransaction

	// run workload, measure runtime
	start := time.Now()
	p.runWorkload(context.Background(), len(cfg.Connections), cfg.ThinkTime)
	elapsed := time.Since(start)
	if elapsed >= cfg.Duration {
		t.Errorf("expected workload to stop early due to forced errors, ran full duration %v", elapsed)
	}
	if atomic.LoadInt64(&p.metrics.Errors) == 0 {
		t.Error("expected at least one error recorded")
	}
}

func TestStatsSnapshotConcurrency(t *testing.T) {
	s := &Stats{}
	// spawn goroutines to record metrics
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				s.Record("insert", time.Nanosecond, 1)
			}
		}()
	}
	// wait for all recorders to finish
	wg.Wait()
	// final snapshot should have recorded many inserts
	snap := s.SnapshotAndReset()
	if snap.NumInsert == 0 {
		t.Error("expected inserts in final snapshot")
	}
}
