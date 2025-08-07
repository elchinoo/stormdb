-- Migration 004: Add error tracking to test_run table
-- This allows storing error information for failed test runs

ALTER TABLE test_run ADD COLUMN IF NOT EXISTS error_message TEXT;
ALTER TABLE test_run ADD COLUMN IF NOT EXISTS error_details JSONB;

-- Index for querying failed tests with errors
CREATE INDEX IF NOT EXISTS idx_test_run_failed_with_errors ON test_run(status, error_message) 
WHERE status = 'failed' AND error_message IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN test_run.error_message IS 'Primary error message for failed test runs';
COMMENT ON COLUMN test_run.error_details IS 'Additional error context and stack traces for debugging';
