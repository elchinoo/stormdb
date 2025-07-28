// Package metrics provides comprehensive performance measurement and reporting capabilities
// for PostgreSQL benchmarking workloads. It handles real-time metrics collection,
// statistical analysis, and formatted reporting with support for various output formats.
//
// The package implements sophisticated latency histogram tracking, transaction per second
// (TPS) calculation, error rate monitoring, and PostgreSQL-specific metrics collection.
// All metrics are thread-safe and designed for high-throughput concurrent workloads.
//
// Key Features:
//   - Real-time metrics collection and reporting
//   - Latency histogram analysis with percentile calculations
//   - PostgreSQL statistics integration (pg_stat_* views)
//   - Connection mode performance comparison
//   - Thread-safe concurrent metrics aggregation
//   - Multiple output formats (console, structured)
//
// Usage Example:
//
//	// Create metrics collector
//	metrics := &types.Metrics{}
//
//	// Record transaction
//	start := time.Now()
//	// ... execute database operation ...
//	metrics.RecordTransaction(time.Since(start))
//
//	// Generate report
//	metrics.Report(config, metrics)
//
// The package integrates seamlessly with all workload types and provides detailed
// performance insights for PostgreSQL optimization and capacity planning.
package metrics

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/elchinoo/stormdb/internal/util"
	"github.com/elchinoo/stormdb/pkg/types"
)

// Report generates a comprehensive performance report displaying transaction metrics,
// latency analysis, and system statistics. This is the primary reporting function
// called at the end of benchmark runs.
//
// The report includes:
//   - Transaction throughput (TPS) and success rates
//   - Latency percentiles (P50, P95, P99) and histogram
//   - Error rates and categorization
//   - PostgreSQL statistics if available
//   - Connection mode performance comparison
//
// Parameters:
//   - cfg: Configuration containing benchmark parameters and settings
//   - m: Metrics struct containing collected performance data
//
// The function automatically detects the current time and calls ReportWithContext
// with default parameters for standard end-of-run reporting.
func Report(cfg *types.Config, m *types.Metrics) {
	ReportWithContext(cfg, m, false, time.Now())
}

// ReportSummary provides a concise, real-time summary suitable for periodic progress updates
// during long-running benchmarks. It displays essential metrics in a compact format
// optimized for continuous monitoring.
//
// The summary includes:
//   - Elapsed time and transaction counts
//   - Current TPS (transactions per second)
//   - Success rate percentage
//   - P95 latency for performance monitoring
//   - Error count if applicable
//
// This function is typically called every few seconds during benchmark execution
// to provide live feedback on benchmark progress and performance.
//
// Parameters:
//   - cfg: Configuration containing benchmark settings
//   - m: Current metrics snapshot
//   - elapsed: Time elapsed since benchmark start
//
// Output format: "⏱️ 30s: 1,250 txns, 41.7 TPS, 98.5% success, P95: 24.3ms"
func ReportSummary(cfg *types.Config, m *types.Metrics, elapsed time.Duration) {
	elapsedSec := elapsed.Seconds()

	// Calculate success rate for transactions
	totalTransactions := m.TPS + m.TPSAborted
	successRate := 100.0
	if totalTransactions > 0 {
		successRate = float64(m.TPS) / float64(totalTransactions) * 100.0
	}

	// Extract latencies safely for P95
	latencies := m.TransactionDur
	var p95ms float64
	if len(latencies) > 0 {
		pvals := util.CalculatePercentiles(latencies, []int{95})
		p95ms = float64(pvals[0]) / 1e6
	}

	fmt.Printf("⏱️  %.0fs: %s txns, %s TPS, %.1f%% success, P95: %.1fms",
		elapsedSec, formatNumber(totalTransactions), formatFloat(float64(m.TPS)/elapsedSec), successRate, p95ms)

	if m.Errors > 0 {
		fmt.Printf(" [%s errors]", formatNumber(m.Errors))
	}
	fmt.Println()
}

// ReportWithContext generates a comprehensive performance report with full contextual information
// including benchmark configuration, timing details, and system state. This is the core
// reporting function that provides detailed analysis suitable for performance evaluation
// and optimization.
//
// The comprehensive report includes:
//   - Benchmark configuration summary (workload, workers, duration)
//   - Transaction throughput metrics (TPS, success rates)
//   - Detailed latency analysis (percentiles, histogram, statistics)
//   - Error analysis and categorization
//   - PostgreSQL system statistics (if available)
//   - Connection mode performance comparison
//   - Timing information and completion status
//
// Parameters:
//   - cfg: Complete benchmark configuration
//   - m: Collected metrics from benchmark execution
//   - interrupted: Whether benchmark was interrupted (affects analysis)
//   - endTime: Timestamp when benchmark completed
//
// This function is called by Report() and provides the foundation for all
// performance analysis and reporting in stormdb.
func ReportWithContext(cfg *types.Config, m *types.Metrics, interrupted bool, endTime time.Time) {
	durationSec := parseDuration(cfg.Duration)

	// Extract latencies safely
	latencies := m.TransactionDur
	pvals := util.CalculatePercentiles(latencies, []int{50, 90, 95, 99})

	// Calculate success rate
	totalTxns := m.TPS + m.TPSAborted
	successRate := 100.0
	if totalTxns > 0 {
		successRate = float64(m.TPS) / float64(totalTxns) * 100.0
	}

	// Header
	fmt.Println("===============================================================================")
	fmt.Println("                         StormDB Benchmark Report")
	fmt.Println("===============================================================================")
	fmt.Printf("Date/Time:       %s\n", endTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration:        %ss       Workers: %d\n", cfg.Duration, cfg.Workers)
	if interrupted {
		fmt.Println("Status:          ⚠️  Test was interrupted before completion")
	}

	// 1. TRANSACTIONS
	fmt.Println("-------------------------------------------------------------------------------")
	fmt.Println("1. TRANSACTIONS")
	fmt.Println("-------------------------------------------------------------------------------")
	fmt.Println(" Metric                          │ Value")
	fmt.Println(" ─────────────────────────────── ┼ ────────────────────────────────────────────")
	fmt.Printf(" Total Transactions              │ %s\n", formatNumber(totalTxns))
	fmt.Printf(" TPS (Committed)                 │ %s\n", formatFloat(float64(m.TPS)/durationSec))
	if m.TPSAborted > 0 {
		fmt.Printf(" TPS (Aborted)                   │ %s\n", formatFloat(float64(m.TPSAborted)/durationSec))
	}
	fmt.Printf(" Success Rate                    │ %.1f%%\n", successRate)

	// 2. QUERIES
	fmt.Println("\n-------------------------------------------------------------------------------")
	fmt.Println("2. QUERIES")
	fmt.Println("-------------------------------------------------------------------------------")
	fmt.Println(" Metric                          │ Value")
	fmt.Println(" ─────────────────────────────── ┼ ────────────────────────────────────────────")
	fmt.Printf(" Total Queries                   │ %s\n", formatNumber(m.QPS))
	fmt.Printf(" QPS (Overall)                   │ %s\n", formatFloat(float64(m.QPS)/durationSec))

	// Query breakdown by type in simple format
	if m.SelectQueries > 0 || m.InsertQueries > 0 || m.UpdateQueries > 0 || m.DeleteQueries > 0 {
		fmt.Println("\n Breakdown by Type:")
		if m.SelectQueries > 0 {
			fmt.Printf("   └ SELECT                      │ %s QPS (%s total)\n",
				formatFloat(float64(m.SelectQueries)/durationSec), formatNumber(m.SelectQueries))
		}
		if m.InsertQueries > 0 {
			fmt.Printf("   └ INSERT                      │ %s QPS (%s total)\n",
				formatFloat(float64(m.InsertQueries)/durationSec), formatNumber(m.InsertQueries))
		}
		if m.UpdateQueries > 0 {
			fmt.Printf("   └ UPDATE                      │ %s QPS (%s total)\n",
				formatFloat(float64(m.UpdateQueries)/durationSec), formatNumber(m.UpdateQueries))
		}
		if m.DeleteQueries > 0 {
			fmt.Printf("   └ DELETE                      │ %s QPS (%s total)\n",
				formatFloat(float64(m.DeleteQueries)/durationSec), formatNumber(m.DeleteQueries))
		}
	}

	// 3. THROUGHPUT
	if m.RowsRead > 0 || m.RowsModified > 0 {
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println("3. THROUGHPUT")
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println(" Metric                          │ Value")
		fmt.Println(" ─────────────────────────────── ┼ ────────────────────────────────────────────")
		if m.RowsRead > 0 {
			fmt.Printf(" Read per second                 │ %s\n", formatFloat(float64(m.RowsRead)/durationSec))
		}
		if m.RowsModified > 0 {
			fmt.Printf(" Modified per second             │ %s\n", formatFloat(float64(m.RowsModified)/durationSec))
		}
	}

	// 4. LATENCY
	if len(latencies) > 0 {
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println("4. LATENCY (milliseconds)")
		fmt.Println("-------------------------------------------------------------------------------")

		// Percentiles in table format
		fmt.Println(" Percentiles:")
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println(" P50(ms)  │ P90(ms)  │ P95(ms)  │ P99(ms)")
		fmt.Println(" ──────── ┼ ──────── ┼ ──────── ┼ ────────")
		fmt.Printf(" %-8.2f │ %-8.2f │ %-8.2f │ %-8.2f\n",
			float64(pvals[0])/1e6, float64(pvals[1])/1e6, float64(pvals[2])/1e6, float64(pvals[3])/1e6)

		// Calculate distribution shape metrics
		distStats := util.CalculateDistributionStats(latencies)

		// Distribution Shape in table format
		fmt.Println("\n Distribution Shape:")
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println(" P25(ms)  │ P75(ms)  │ IQR(ms)  │ MAD(ms)  │ Skewness │ Kurtosis │ CoV")
		fmt.Println(" ──────── ┼ ──────── ┼ ──────── ┼ ──────── ┼ ──────── ┼ ──────── ┼ ────────")
		fmt.Printf(" %-8.2f │ %-8.2f │ %-8.2f │ %-8.2f │ %-8.3f │ %-8.3f │ %-8.3f\n",
			float64(distStats.P25)/1e6, float64(distStats.P75)/1e6, float64(distStats.IQR)/1e6,
			distStats.MAD/1e6, distStats.Skewness, distStats.Kurtosis, distStats.CoV)

		// Transaction Time in table format
		avgMs, minMs, maxMs, stddevMs := util.Stats(latencies)
		avgMsFloat := float64(avgMs) / 1e6
		minMsFloat := float64(minMs) / 1e6
		maxMsFloat := float64(maxMs) / 1e6
		stddevMsFloat := float64(stddevMs) / 1e6

		fmt.Println("\n Transaction Time:")
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println(" Min(ms)  │ Max(ms)  │ Avg(ms)  │ StdDev")
		fmt.Println(" ──────── ┼ ──────── ┼ ──────── ┼ ────────")
		fmt.Printf(" %-8.2f │ %-8.2f │ %-8.2f │ %-8.2f\n", minMsFloat, maxMsFloat, avgMsFloat, stddevMsFloat)

		// Latency Histogram
		if len(m.LatencyHistogram) > 0 {
			fmt.Println("\n Latency Histogram (ms):")

			// Define bucket ranges for visualization
			bucketRanges := []struct {
				name    string
				buckets []string
			}{
				{"0–1", []string{"0.1ms", "0.5ms", "1.0ms"}},
				{"1–5", []string{"2.0ms", "5.0ms"}},
				{"5–10", []string{"10.0ms"}},
				{"10–20", []string{"20.0ms"}},
				{"20+", []string{"50.0ms", "100.0ms", "200.0ms", "500.0ms", "1000.0ms", "+inf"}},
			}

			// Calculate totals for each range
			totalSamples := int64(0)
			for _, count := range m.LatencyHistogram {
				totalSamples += count
			}

			for _, bucketRange := range bucketRanges {
				rangeTotal := int64(0)
				for _, bucket := range bucketRange.buckets {
					if count, exists := m.LatencyHistogram[bucket]; exists {
						rangeTotal += count
					}
				}

				if rangeTotal > 0 {
					percentage := float64(rangeTotal) / float64(totalSamples) * 100.0
					bar := createHistogramBar(percentage, 20)
					fmt.Printf("   %-6s │ %-20s %3.0f%%\n", bucketRange.name, bar, percentage)
				}
			}
		}
	}

	// 5. ERRORS
	fmt.Println("\n-------------------------------------------------------------------------------")
	fmt.Println("5. ERRORS")
	fmt.Println("-------------------------------------------------------------------------------")
	fmt.Println(" Metric                          │ Value")
	fmt.Println(" ─────────────────────────────── ┼ ────────────────────────────────────────────")
	fmt.Printf(" Total Query Errors              │ %s\n", formatNumber(m.Errors))

	if len(m.ErrorTypes) > 0 {
		fmt.Println(" Error Types:")
		for errType, count := range m.ErrorTypes {
			// Show the full error message for better debugging
			fmt.Printf("   └ %-27s │ %s\n", errType, formatNumber(count))
		}
	}

	// Per-transaction breakdown (TPCC-specific)
	if m.NewOrderCount > 0 || m.PaymentCount > 0 || m.OrderStatusCount > 0 {
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println("6. TRANSACTION MIX")
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println(" Transaction Type                │ Value")
		fmt.Println(" ─────────────────────────────── ┼ ────────────────────────────────────────────")
		if m.NewOrderCount > 0 {
			fmt.Printf(" New-Order                       │ %s TPS (%s total)\n",
				formatFloat(float64(m.NewOrderCount)/durationSec), formatNumber(m.NewOrderCount))
		}
		if m.PaymentCount > 0 {
			fmt.Printf(" Payment                         │ %s TPS (%s total)\n",
				formatFloat(float64(m.PaymentCount)/durationSec), formatNumber(m.PaymentCount))
		}
		if m.OrderStatusCount > 0 {
			fmt.Printf(" Order-Status                    │ %s TPS (%s total)\n",
				formatFloat(float64(m.OrderStatusCount)/durationSec), formatNumber(m.OrderStatusCount))
		}
	}

	// Worker breakdown section
	if len(m.WorkerMetrics) > 1 { // Only show if we have multiple workers
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println("WORKER BREAKDOWN")
		fmt.Println("-------------------------------------------------------------------------------")

		// Calculate worker statistics
		workerStats := make([]util.WorkerPerformanceStats, 0, len(m.WorkerMetrics))
		for workerID, worker := range m.WorkerMetrics {
			stats := util.CalculateWorkerStats(
				workerID,
				worker.TPS,
				worker.TPSAborted,
				worker.QPS,
				worker.Errors,
				worker.TransactionDur,
				durationSec,
			)
			workerStats = append(workerStats, stats)
		}

		// Sort by worker ID for consistent display
		sort.Slice(workerStats, func(i, j int) bool {
			return workerStats[i].WorkerID < workerStats[j].WorkerID
		})

		// Table header
		fmt.Printf(" %-6s │ %-8s │ %-8s │ %-8s │ %-8s │ %-8s │ %-6s\n",
			"Worker", "TPS", "QPS", "P50(ms)", "P95(ms)", "Success%", "Errors")
		fmt.Println(" ───────┼──────────┼──────────┼──────────┼──────────┼──────────┼────────")

		// Worker rows
		for _, stats := range workerStats {
			fmt.Printf(" %-6d │ %-8s │ %-8s │ %-8.2f │ %-8.2f │ %-8.1f │ %-6d\n",
				stats.WorkerID,
				formatFloat(stats.TPS),
				formatFloat(stats.QPS),
				stats.P50Latency,
				stats.P95Latency,
				stats.SuccessRate,
				stats.ErrorCount,
			)
		}

		// Calculate variance analysis
		if len(workerStats) > 1 {
			fmt.Println("\n Worker Load Distribution Analysis:")

			// Calculate TPS variance
			var tpsValues []float64
			var qpsValues []float64
			var p50Values []float64

			for _, stats := range workerStats {
				tpsValues = append(tpsValues, stats.TPS)
				qpsValues = append(qpsValues, stats.QPS)
				p50Values = append(p50Values, stats.P50Latency)
			}

			tpsCoV := calculateCoV(tpsValues)
			qpsCoV := calculateCoV(qpsValues)
			p50CoV := calculateCoV(p50Values)

			fmt.Printf("   └ TPS Variance (CoV)            │ %.3f", tpsCoV)
			if tpsCoV > 0.1 {
				fmt.Printf(" ⚠️  High variance detected")
			}
			fmt.Println()

			fmt.Printf("   └ QPS Variance (CoV)            │ %.3f", qpsCoV)
			if qpsCoV > 0.1 {
				fmt.Printf(" ⚠️  High variance detected")
			}
			fmt.Println()

			fmt.Printf("   └ P50 Latency Variance (CoV)    │ %.3f", p50CoV)
			if p50CoV > 0.2 {
				fmt.Printf(" ⚠️  High variance detected")
			}
			fmt.Println()
		}
	}

	// 6. TIME-SERIES ANALYSIS
	if m.TimeSeries != nil && len(m.TimeSeries.Buckets) > 1 {
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println("7. TIME-SERIES ANALYSIS")
		fmt.Println("-------------------------------------------------------------------------------")

		tsStats := util.AnalyzeTimeSeries(m.TimeSeries.Buckets)

		fmt.Println(" Metric                          │ Value")
		fmt.Println(" ─────────────────────────────── ┼ ────────────────────────────────────────────")
		fmt.Printf(" Time Buckets Analyzed           │ %d\n", len(m.TimeSeries.Buckets))
		fmt.Printf(" Collection Period               │ %v\n", m.BucketInterval)

		fmt.Println("\n Load vs Latency Correlations:")
		fmt.Printf("   └ QPS vs Latency (Pearson)      │ %.3f", tsStats.PearsonCorrelation)
		if math.Abs(tsStats.PearsonCorrelation) > 0.7 {
			fmt.Printf(" 🔍 Strong correlation")
		} else if math.Abs(tsStats.PearsonCorrelation) > 0.3 {
			fmt.Printf(" 📊 Moderate correlation")
		}
		fmt.Println()

		fmt.Printf("   └ QPS vs Latency (Spearman)     │ %.3f", tsStats.SpearmanCorrelation)
		if math.Abs(tsStats.SpearmanCorrelation) > 0.7 {
			fmt.Printf(" 🔍 Strong monotonic relationship")
		}
		fmt.Println()

		fmt.Println("\n Load Characteristics:")
		fmt.Printf("   └ Peak QPS                      │ %.2f\n", tsStats.PeakQPS)
		fmt.Printf("   └ Median QPS                    │ %.2f\n", tsStats.MedianQPS)
		fmt.Printf("   └ Peak Latency                  │ %.2f ms\n", tsStats.PeakLatency)
		fmt.Printf("   └ Median Latency                │ %.2f ms\n", tsStats.MedianLatency)

		if len(tsStats.LoadStabilityRegions) > 0 {
			fmt.Println("\n Load Regions Detected:")
			for i, region := range tsStats.LoadStabilityRegions {
				status := "Variable"
				if region.IsStable {
					status = "Stable"
				}
				fmt.Printf("   └ Region %d: %s (QPS: %.1f-%.1f, Latency: %.2f-%.2f ms)\n",
					i+1, status, region.QPSRange[0], region.QPSRange[1],
					region.LatencyRange[0], region.LatencyRange[1])
			}
		}

		// Show regression slope for trend analysis
		if !math.IsNaN(tsStats.LatencySlope) {
			fmt.Printf("\n Trend Analysis:\n")
			fmt.Printf("   └ Latency increase per 100 QPS  │ %.3f ms\n", tsStats.LatencySlope)
			if tsStats.LatencySlope > 10.0 {
				fmt.Printf("   └ Trend: Performance degrades with load 📉\n")
			} else if tsStats.LatencySlope < -1.0 {
				fmt.Printf("   └ Trend: Performance improves with load 📈\n")
			} else {
				fmt.Printf("   └ Trend: Stable performance across load levels ➡️\n")
			}
		}
	}

	// 8. CONNECTION MODE COMPARISON (for connection_overhead workload)
	if m.PersistentConnMetrics != nil || m.TransientConnMetrics != nil {
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println("8. CONNECTION MODE COMPARISON")
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println(" Metric                          │ Persistent        │ Transient         │ Overhead")
		fmt.Println(" ─────────────────────────────── ┼ ───────────────── ┼ ───────────────── ┼ ─────────")

		// Get metrics (thread-safe)
		var persistentTPS, persistentQPS, persistentErrors int64
		var transientTPS, transientQPS, transientErrors int64
		var persistentAvgDur, transientAvgDur float64
		var avgConnSetup float64
		var connCount int64

		if m.PersistentConnMetrics != nil {
			m.PersistentConnMetrics.Mu.RLock()
			persistentTPS = m.PersistentConnMetrics.TPS
			persistentQPS = m.PersistentConnMetrics.QPS
			persistentErrors = m.PersistentConnMetrics.Errors
			if len(m.PersistentConnMetrics.TransactionDur) > 0 {
				var sum int64
				for _, dur := range m.PersistentConnMetrics.TransactionDur {
					sum += dur
				}
				persistentAvgDur = float64(sum) / float64(len(m.PersistentConnMetrics.TransactionDur)) / 1e6 // ns to ms
			}
			m.PersistentConnMetrics.Mu.RUnlock()
		}

		if m.TransientConnMetrics != nil {
			m.TransientConnMetrics.Mu.RLock()
			transientTPS = m.TransientConnMetrics.TPS
			transientQPS = m.TransientConnMetrics.QPS
			transientErrors = m.TransientConnMetrics.Errors
			connCount = m.TransientConnMetrics.ConnectionCount
			if len(m.TransientConnMetrics.TransactionDur) > 0 {
				var sum int64
				for _, dur := range m.TransientConnMetrics.TransactionDur {
					sum += dur
				}
				transientAvgDur = float64(sum) / float64(len(m.TransientConnMetrics.TransactionDur)) / 1e6 // ns to ms
			}
			if len(m.TransientConnMetrics.ConnectionSetup) > 0 {
				var sum int64
				for _, setup := range m.TransientConnMetrics.ConnectionSetup {
					sum += setup
				}
				avgConnSetup = float64(sum) / float64(len(m.TransientConnMetrics.ConnectionSetup)) / 1e6 // ns to ms
			}
			m.TransientConnMetrics.Mu.RUnlock()
		}

		// Calculate overhead percentages
		var tpsOverhead, qpsOverhead, latencyOverhead string
		if persistentTPS > 0 {
			tpsOverhead = fmt.Sprintf("%.1f%%", (float64(persistentTPS-transientTPS)/float64(persistentTPS))*100)
		} else {
			tpsOverhead = "N/A"
		}
		if persistentQPS > 0 {
			qpsOverhead = fmt.Sprintf("%.1f%%", (float64(persistentQPS-transientQPS)/float64(persistentQPS))*100)
		} else {
			qpsOverhead = "N/A"
		}
		if persistentAvgDur > 0 {
			latencyOverhead = fmt.Sprintf("%.1f%%", ((transientAvgDur-persistentAvgDur)/persistentAvgDur)*100)
		} else {
			latencyOverhead = "N/A"
		}

		fmt.Printf(" Transactions/sec                │ %17s │ %17s │ %8s\n",
			formatNumber(persistentTPS), formatNumber(transientTPS), tpsOverhead)
		fmt.Printf(" Queries/sec                     │ %17s │ %17s │ %8s\n",
			formatNumber(persistentQPS), formatNumber(transientQPS), qpsOverhead)
		fmt.Printf(" Errors                          │ %17s │ %17s │ N/A\n",
			formatNumber(persistentErrors), formatNumber(transientErrors))
		fmt.Printf(" Avg Transaction Latency (ms)    │ %17.2f │ %17.2f │ %8s\n",
			persistentAvgDur, transientAvgDur, latencyOverhead)
		if avgConnSetup > 0 {
			fmt.Printf(" Avg Connection Setup (ms)       │ %17s │ %17.2f │ N/A\n",
				"N/A (pooled)", avgConnSetup)
		}
		if connCount > 0 {
			fmt.Printf(" Total Connections Created       │ %17s │ %17s │ N/A\n",
				"N/A (pooled)", formatNumber(connCount))
		}

		fmt.Println("\n Connection Mode Summary:")
		if persistentTPS > transientTPS {
			fmt.Printf("   • Persistent connections are %.1fx faster for transactions\n",
				float64(persistentTPS)/float64(transientTPS))
		}
		if persistentAvgDur < transientAvgDur {
			fmt.Printf("   • Persistent connections have %.1fms lower latency on average\n",
				transientAvgDur-persistentAvgDur)
		}
		if avgConnSetup > 0 {
			fmt.Printf("   • Each transient connection adds %.2fms setup overhead\n", avgConnSetup)
		}
	}

	// 9. POSTGRESQL STATISTICS
	if pgStats := m.GetPgStats(); pgStats != nil && !pgStats.LastUpdated.IsZero() {
		fmt.Println("\n-------------------------------------------------------------------------------")
		fmt.Println("9. POSTGRESQL STATISTICS")
		fmt.Println("-------------------------------------------------------------------------------")
		fmt.Println(" Metric                          │ Value")
		fmt.Println(" ─────────────────────────────── ┼ ────────────────────────────────────────────")

		// Buffer cache statistics
		fmt.Printf(" Buffer Cache Hit Ratio          │ %.1f%% (higher is better)\n", pgStats.BufferCacheHitRatio)
		fmt.Printf(" Blocks Read (disk)              │ %s (cache misses)\n", formatLargeNumber(pgStats.BlocksRead))
		fmt.Printf(" Blocks Hit (cache)              │ %s (cache hits)\n", formatLargeNumber(pgStats.BlocksHit))
		fmt.Printf(" Blocks Written (bgwriter)       │ %s (background writer)\n", formatLargeNumber(pgStats.BlocksWritten))

		// WAL statistics
		if pgStats.WALRecords > 0 || pgStats.WALBytes > 0 {
			fmt.Printf(" WAL Records                     │ %s (transaction log entries)\n", formatLargeNumber(pgStats.WALRecords))
			fmt.Printf(" WAL Bytes                       │ %s (transaction log size)\n", formatBytes(pgStats.WALBytes))
		}

		// Checkpoint statistics
		fmt.Printf(" Checkpoints (requested)         │ %s (manual checkpoints)\n", formatLargeNumber(pgStats.CheckpointsReq))
		fmt.Printf(" Checkpoints (timed)             │ %s (automatic checkpoints)\n", formatLargeNumber(pgStats.CheckpointsTimed))

		// Temporary files (spilling to disk)
		if pgStats.TempFiles > 0 {
			fmt.Printf(" Temporary Files Created         │ %s (work_mem exceeded)\n", formatLargeNumber(pgStats.TempFiles))
			if pgStats.TempBytes > 0 {
				fmt.Printf(" Temporary Bytes                 │ %s (spilled to disk)\n", formatBytes(pgStats.TempBytes))
			}
		}

		// Locking and contention
		if pgStats.Deadlocks > 0 {
			fmt.Printf(" Deadlocks                       │ %s (concurrency conflicts)\n", formatLargeNumber(pgStats.Deadlocks))
		}
		if pgStats.LockWaitCount > 0 {
			fmt.Printf(" Lock Wait Events                │ %s (contention indicators)\n", formatLargeNumber(pgStats.LockWaitCount))
		}

		// Connection statistics
		fmt.Printf(" Active Connections              │ %d / %d (%.1f%% utilization)\n",
			pgStats.ActiveConnections, pgStats.MaxConnections,
			float64(pgStats.ActiveConnections)/float64(pgStats.MaxConnections)*100.0)

		// Autovacuum statistics
		if pgStats.AutovacuumCount > 0 {
			fmt.Printf(" Autovacuum Processes            │ %s (maintenance operations)\n", formatLargeNumber(pgStats.AutovacuumCount))
		}

		// Add explanation for workload-specific statistics
		fmt.Println()
		fmt.Println(" Note: These statistics show precise changes during workload execution.")
		fmt.Println(" Measured from workload start to completion, excluding setup/teardown activity.")

		// pg_stat_statements top queries
		if len(pgStats.TopQueries) > 0 {
			fmt.Println("\n Top Queries by Execution Time:")
			for i, query := range pgStats.TopQueries {
				// Truncate long queries for display
				displayQuery := query.Query
				if len(displayQuery) > 60 {
					displayQuery = displayQuery[:57] + "..."
				}

				fmt.Printf("   %d. %-58s │ %s calls, %.2fms avg\n",
					i+1, displayQuery, formatNumber(query.Calls), query.MeanTime)

				if query.HitPercent > 0 {
					fmt.Printf("      %-58s │ %.1f%% cache hit ratio\n", "", query.HitPercent)
				}
			}
		}

		fmt.Printf("\n Last Updated: %s\n", pgStats.LastUpdated.Format("15:04:05"))
	}

	fmt.Println("\n===============================================================================")
}

func parseDuration(d string) float64 {
	dur, _ := time.ParseDuration(d)
	return dur.Seconds()
}

// formatNumber formats numbers with thousand separators
func formatNumber(n int64) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}

	result := ""
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

// formatBytes formats byte values in human-readable units (B, KB, MB, GB, TB)
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	const unit = 1024
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}

	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp+1])
}

// formatLargeNumber formats large numbers with suffixes (K, M, G, etc.)
func formatLargeNumber(n int64) string {
	if n == 0 {
		return "0"
	}

	const unit = 1000
	units := []string{"", "K", "M", "G", "T", "P"}

	if n < unit {
		return formatNumber(n)
	}

	div, exp := int64(unit), 0
	for num := n / unit; num >= unit; num /= unit {
		div *= unit
		exp++
	}

	result := float64(n) / float64(div)
	if result >= 100 {
		return fmt.Sprintf("%.0f%s", result, units[exp+1])
	} else if result >= 10 {
		return fmt.Sprintf("%.1f%s", result, units[exp+1])
	}
	return fmt.Sprintf("%.2f%s", result, units[exp+1])
}

// formatFloat formats float numbers with thousand separators
func formatFloat(f float64) string {
	if f < 1000 {
		return fmt.Sprintf("%.2f", f)
	}

	str := fmt.Sprintf("%.2f", f)
	parts := strings.Split(str, ".")
	integer := parts[0]
	decimal := parts[1]

	result := ""
	for i, c := range integer {
		if i > 0 && (len(integer)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result + "." + decimal
}

// createHistogramBar creates a visual bar for histogram
func createHistogramBar(percentage float64, maxWidth int) string {
	if percentage == 0 {
		return ""
	}

	barWidth := int(percentage * float64(maxWidth) / 100.0)
	if barWidth == 0 && percentage > 0 {
		barWidth = 1
	}

	return strings.Repeat("█", barWidth)
}

// calculateCoV calculates the coefficient of variation for a slice of float64 values
func calculateCoV(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	// Calculate mean
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	if mean == 0 {
		return 0.0
	}

	// Calculate standard deviation
	var sumSq float64
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	stddev := math.Sqrt(sumSq / float64(len(values)))

	// Return coefficient of variation
	return stddev / mean
}
