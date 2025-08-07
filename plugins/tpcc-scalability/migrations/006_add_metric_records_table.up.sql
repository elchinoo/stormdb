-- Create metric_records table for raw thread-level metrics
CREATE TABLE IF NOT EXISTS metric_records (
    id SERIAL PRIMARY KEY,
    thread_id INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    dt_started TIMESTAMP NOT NULL,
    dt_end TIMESTAMP NOT NULL,
    num_connections INTEGER NOT NULL,
    num_insert BIGINT NOT NULL DEFAULT 0,
    num_update BIGINT NOT NULL DEFAULT 0,
    num_delete BIGINT NOT NULL DEFAULT 0,
    num_select BIGINT NOT NULL DEFAULT 0,
    latency_sum BIGINT NOT NULL DEFAULT 0, -- nanoseconds
    latency_count BIGINT NOT NULL DEFAULT 0,
    num_row_insert BIGINT NOT NULL DEFAULT 0,
    num_row_update BIGINT NOT NULL DEFAULT 0,
    num_row_delete BIGINT NOT NULL DEFAULT 0,
    num_row_select BIGINT NOT NULL DEFAULT 0,
    num_errors BIGINT NOT NULL DEFAULT 0
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_metric_records_thread_id ON metric_records(thread_id);
CREATE INDEX IF NOT EXISTS idx_metric_records_timestamp ON metric_records(timestamp);
CREATE INDEX IF NOT EXISTS idx_metric_records_dt_started ON metric_records(dt_started);

-- Create a view for aggregated metrics (similar to old structure)
CREATE OR REPLACE VIEW aggregated_metrics AS
SELECT 
    MIN(dt_started) as dt_started,
    MAX(dt_end) as dt_end,
    MAX(num_connections) as num_connections,
    SUM(num_insert) as num_insert,
    SUM(num_update) as num_update,
    SUM(num_delete) as num_delete,
    SUM(num_select) as num_select,
    SUM(latency_sum) as latency_sum,
    SUM(latency_count) as latency_count,
    SUM(num_row_insert) as num_row_insert,
    SUM(num_row_update) as num_row_update,
    SUM(num_row_delete) as num_row_delete,
    SUM(num_row_select) as num_row_select,
    SUM(num_errors) as num_errors,
    -- Calculated fields
    CASE 
        WHEN SUM(latency_count) > 0 
        THEN (SUM(latency_sum) / SUM(latency_count)) / 1000000.0 -- avg latency in ms
        ELSE 0 
    END as avg_latency_ms,
    CASE 
        WHEN (SUM(num_insert) + SUM(num_update) + SUM(num_delete) + SUM(num_select) + SUM(num_errors)) > 0
        THEN (SUM(num_errors)::float / (SUM(num_insert) + SUM(num_update) + SUM(num_delete) + SUM(num_select) + SUM(num_errors))) * 100
        ELSE 0 
    END as error_rate_pct,
    (SUM(num_insert) + SUM(num_update) + SUM(num_delete) + SUM(num_select)) as total_operations,
    (SUM(num_row_insert) + SUM(num_row_update) + SUM(num_row_delete) + SUM(num_row_select)) as total_rows
FROM metric_records
WHERE dt_started IS NOT NULL 
  AND dt_end IS NOT NULL
GROUP BY DATE_TRUNC('second', timestamp)
ORDER BY MIN(dt_started);
