-- Migration 004: Add active connections and workers tracking to test_run_result

-- Add columns to track the number of active connections and workers at the time each metric was saved
ALTER TABLE test_run_result 
ADD COLUMN IF NOT EXISTS active_connections INTEGER,
ADD COLUMN IF NOT EXISTS active_workers INTEGER;

-- Create indexes for querying by connection and worker counts
CREATE INDEX IF NOT EXISTS idx_test_run_result_active_connections ON test_run_result(active_connections);
CREATE INDEX IF NOT EXISTS idx_test_run_result_active_workers ON test_run_result(active_workers);

-- Create a compound index for metrics with connection/worker analysis
CREATE INDEX IF NOT EXISTS idx_test_run_result_conn_workers ON test_run_result(test_run_id, active_connections, active_workers);

-- Add comments to clarify the purpose of these fields
COMMENT ON COLUMN test_run_result.active_connections IS 'Number of database connections active when this metric was recorded';
COMMENT ON COLUMN test_run_result.active_workers IS 'Number of worker threads/processes active when this metric was recorded';
