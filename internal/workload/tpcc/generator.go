// internal/workload/tpcc/generator.go
package tpcc

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elchinoo/stormdb/internal/util"
	"github.com/elchinoo/stormdb/pkg/types"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Run starts the TPCC workload with multiple workers
func (t *TPCC) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	var wg sync.WaitGroup
	start := time.Now() // ✅ Capture start time

	stopReporting := t.startRealTimeReporter(ctx, cfg, metrics, start)

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			t.worker(ctx, db, cfg, metrics)
		}()
	}

	wg.Wait()
	stopReporting()

	return nil
}

// worker runs the transaction mix in a loop
func (t *TPCC) worker(ctx context.Context, db *pgxpool.Pool, _ *types.Config, metrics *types.Metrics) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()

			var err error
			switch txType := rollTransaction(rng); txType {
			case "new_order":
				err = t.newOrderTx(ctx, db, rng)
				atomic.AddInt64(&metrics.NewOrderCount, 1)
			case "payment":
				err = t.paymentTx(ctx, db, rng)
				atomic.AddInt64(&metrics.PaymentCount, 1)
			case "order_status":
				err = t.orderStatusTx(ctx, db, rng)
				atomic.AddInt64(&metrics.OrderStatusCount, 1)
			default: // think
				atomic.AddInt64(&metrics.ThinkCount, 1)
				time.Sleep(time.Duration(1+rng.Intn(10)) * time.Millisecond)
				continue
			}

			elapsed := time.Since(start).Nanoseconds()

			// Record latency
			metrics.Mu.Lock()
			metrics.TransactionDur = append(metrics.TransactionDur, elapsed)
			metrics.Mu.Unlock()

			if err != nil {
				atomic.AddInt64(&metrics.Errors, 1)
				metrics.Mu.Lock()
				metrics.ErrorTypes[err.Error()]++
				metrics.Mu.Unlock()
			} else {
				atomic.AddInt64(&metrics.TPS, 1)
				atomic.AddInt64(&metrics.QPS, 1)
			}

			// Simulate think time
			time.Sleep(time.Duration(1+rng.Intn(10)) * time.Millisecond)
		}
	}
}

// rollTransaction returns which transaction to run based on TPC-C weighting
func rollTransaction(rng *rand.Rand) string {
	r := rng.Intn(1000)
	switch {
	case r < 450: // 45%
		return "new_order"
	case r < 880: // 43%
		return "payment"
	case r < 920: // 4%
		return "order_status"
	default: // 8%
		return "think"
	}
}

func (t *TPCC) startRealTimeReporter(ctx context.Context, _ *types.Config, metrics *types.Metrics, start time.Time) context.CancelFunc {
	ticker := time.NewTicker(5 * time.Second)
	reportCtx, cancel := context.WithCancel(context.Background())

	go func() {
		defer ticker.Stop()
		var lastTPS, lastQPS, lastErrors int64

		for {
			select {
			case <-ticker.C:
				// Capture current values
				tps := atomic.LoadInt64(&metrics.TPS)
				qps := atomic.LoadInt64(&metrics.QPS)
				errors := atomic.LoadInt64(&metrics.Errors)

				// Compute rates over last 5s
				tpsRate := float64(tps-lastTPS) / 5.0
				qpsRate := float64(qps-lastQPS) / 5.0
				errRate := float64(errors-lastErrors) / 5.0

				// Snapshot latencies under mutex
				metrics.Mu.Lock()
				latencies := make([]int64, len(metrics.TransactionDur))
				copy(latencies, metrics.TransactionDur)
				metrics.Mu.Unlock()

				// Compute percentiles
				p50, p95, p99 := float64(0), float64(0), float64(0)
				if len(latencies) > 0 {
					pcts := util.CalculatePercentiles(latencies, []int{50, 95, 99})
					p50 = float64(pcts[0]) / 1e6 // ns → ms
					p95 = float64(pcts[1]) / 1e6
					p99 = float64(pcts[2]) / 1e6
				}

				// Log real-time stats
				log.Printf("📈 REALTIME [%.0fs] TPS: %.1f | QPS: %.1f | ERR: %.1f/s | Latency: P50=%.2fms P95=%.2fms P99=%.2fms",
					time.Since(start).Seconds(), // we need to define 'start'
					tpsRate, qpsRate, errRate, p50, p95, p99)

				// Update last values
				lastTPS = tps
				lastQPS = qps
				lastErrors = errors

			case <-reportCtx.Done():
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return cancel
}
