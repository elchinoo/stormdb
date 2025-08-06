-- Migration 003: Create test_run_logs table

CREATE TABLE IF NOT EXISTS test_run_logs (
    id            BIGSERIAL PRIMARY KEY,
    test_run_id   BIGINT    REFERENCES test_run(id) ON DELETE CASCADE,
    timestamp     TIMESTAMPTZ NOT NULL DEFAULT now(),
    level         TEXT      NOT NULL,
    message       TEXT      NOT NULL,
    metadata      JSONB,
    created_at    TIMESTAMPTZ DEFAULT now()
);

-- Index for querying recent logs
CREATE INDEX IF NOT EXISTS idx_test_run_logs_run_id_ts ON test_run_logs(test_run_id, timestamp DESC);
