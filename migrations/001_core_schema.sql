-- StormDB v2 Core Schema Migration
-- This migration creates the foundation schema for the modular plugin-based architecture

-- 1. Static catalog tables for metadata
-- These tables store plugin metadata, test types, and metric definitions

CREATE TABLE IF NOT EXISTS test_type (
    id          SERIAL PRIMARY KEY,
    code        TEXT UNIQUE NOT NULL,         -- e.g. 'bulk_insert', 'read_latency'
    name        TEXT NOT NULL,                -- Human-readable name
    description TEXT,                         -- Detailed description
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now(),
    is_active   BOOLEAN     DEFAULT true
);

-- Index for fast lookups by code
CREATE INDEX IF NOT EXISTS idx_test_type_code ON test_type(code);
CREATE INDEX IF NOT EXISTS idx_test_type_active ON test_type(is_active) WHERE is_active = true;

CREATE TABLE IF NOT EXISTS plugin (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,                -- Plugin name (e.g., 'bulk_load')
    version     TEXT NOT NULL,                -- Semantic version (e.g., '1.2.3')
    sha256      TEXT NOT NULL,                -- Build hash for integrity verification
    config_path TEXT,                         -- Path to configuration file
    metadata    JSONB NOT NULL,               -- Full plugin metadata
    created_at  TIMESTAMPTZ DEFAULT now(),
    is_active   BOOLEAN     DEFAULT true,
    UNIQUE(name, version)
);

-- Indexes for plugin discovery and verification
CREATE INDEX IF NOT EXISTS idx_plugin_name ON plugin(name);
CREATE INDEX IF NOT EXISTS idx_plugin_name_version ON plugin(name, version);
CREATE INDEX IF NOT EXISTS idx_plugin_sha256 ON plugin(sha256);
CREATE INDEX IF NOT EXISTS idx_plugin_active ON plugin(is_active) WHERE is_active = true;

CREATE TABLE IF NOT EXISTS test_metric (
    id          SERIAL PRIMARY KEY,
    code        TEXT UNIQUE NOT NULL,         -- Standard codes: ROW_INSERT, ROW_SELECT, etc.
    description TEXT,                         -- What this metric measures
    unit        TEXT NOT NULL,                -- Units: 'rows/s', 'ms', 'bytes', etc.
    created_at  TIMESTAMPTZ DEFAULT now(),
    updated_at  TIMESTAMPTZ DEFAULT now(),
    is_active   BOOLEAN     DEFAULT true
);

-- Index for metric lookups
CREATE INDEX IF NOT EXISTS idx_test_metric_code ON test_metric(code);
CREATE INDEX IF NOT EXISTS idx_test_metric_active ON test_metric(is_active) WHERE is_active = true;

-- 2. Runtime execution tables
-- These tables track test runs and their results

CREATE TYPE run_status AS ENUM ('pending','running','succeeded','failed','aborted');

CREATE TABLE IF NOT EXISTS test_run (
    id            BIGSERIAL PRIMARY KEY,
    test_type_id  INT      REFERENCES test_type(id),
    plugin_id     INT      REFERENCES plugin(id),
    host          VARCHAR(200) NOT NULL,              -- Target database host
    port          INT NOT NULL,                       -- Target database port
    db_name       VARCHAR(200) NOT NULL,              -- Target database name
    name          VARCHAR(200) NOT NULL,              -- Human-readable test run name
    description   TEXT,                               -- Optional test description
    created_at    TIMESTAMPTZ DEFAULT now(),
    start_time    TIMESTAMPTZ,                       -- Actual execution start
    end_time      TIMESTAMPTZ,                       -- Actual execution end
    status        run_status DEFAULT 'pending',
    config        JSONB      NOT NULL DEFAULT '{}'   -- Merged configuration for traceability
);

-- Indexes for test run queries
CREATE INDEX IF NOT EXISTS idx_test_run_status ON test_run(status);
CREATE INDEX IF NOT EXISTS idx_test_run_plugin ON test_run(plugin_id);
CREATE INDEX IF NOT EXISTS idx_test_run_type ON test_run(test_type_id);
CREATE INDEX IF NOT EXISTS idx_test_run_created ON test_run(created_at);
CREATE INDEX IF NOT EXISTS idx_test_run_times ON test_run(start_time, end_time);

CREATE TABLE IF NOT EXISTS test_run_result (
    id             BIGSERIAL PRIMARY KEY,
    test_run_id    BIGINT   REFERENCES test_run(id) ON DELETE CASCADE,
    test_metric_id INT REFERENCES test_metric(id),
    start_time     TIMESTAMPTZ NOT NULL,             -- Measurement window start
    end_time       TIMESTAMPTZ NOT NULL,             -- Measurement window end
    value          DOUBLE PRECISION NOT NULL,        -- Measured value
    tags           JSONB,      -- Flexible metadata (e.g., {"batch_size":1000})
    created_at     TIMESTAMPTZ DEFAULT now()
);

-- Access-pattern optimized indexes
CREATE INDEX IF NOT EXISTS idx_test_run_result_run_id ON test_run_result(test_run_id);
CREATE INDEX IF NOT EXISTS idx_test_run_result_metric_id ON test_run_result(test_metric_id);
CREATE INDEX IF NOT EXISTS idx_test_run_result_time_range ON test_run_result USING BRIN (start_time);
CREATE INDEX IF NOT EXISTS idx_test_run_result_value ON test_run_result(value);
CREATE INDEX IF NOT EXISTS idx_test_run_result_tags ON test_run_result USING GIN (tags);

-- 3. Initialize standard test metrics
-- These are the core metrics that all plugins should use

INSERT INTO test_metric (code, description, unit) VALUES 
    ('ROW_INSERT', 'Rate of row insertions', 'rows/s'),
    ('ROW_SELECT', 'Rate of row selections', 'rows/s'),
    ('ROW_UPDATE', 'Rate of row updates', 'rows/s'),
    ('ROW_DELETE', 'Rate of row deletions', 'rows/s'),
    ('ROW_COPY', 'Rate of bulk copy operations', 'rows/s'),
    ('LATENCY_AVG', 'Average query latency', 'ms'),
    ('LATENCY_P50', '50th percentile latency', 'ms'),
    ('LATENCY_P95', '95th percentile latency', 'ms'),
    ('LATENCY_P99', '99th percentile latency', 'ms'),
    ('THROUGHPUT', 'Overall throughput', 'ops/s'),
    ('ERROR_RATE', 'Error rate', 'errors/s'),
    ('CONNECTION_COUNT', 'Active database connections', 'connections'),
    ('MEMORY_USAGE', 'Memory utilization', 'bytes')
ON CONFLICT (code) DO NOTHING;

-- 4. Add helpful views for common queries

-- View for test run summaries with plugin and type information
CREATE OR REPLACE VIEW test_run_summary AS
SELECT 
    tr.id,
    tr.name,
    tr.description,
    tr.status,
    tr.created_at,
    tr.start_time,
    tr.end_time,
    EXTRACT(EPOCH FROM (tr.end_time - tr.start_time)) as duration_seconds,
    tt.code as test_type_code,
    tt.name as test_type_name,
    p.name as plugin_name,
    p.version as plugin_version,
    tr.host,
    tr.port,
    tr.db_name
FROM test_run tr
JOIN test_type tt ON tr.test_type_id = tt.id
JOIN plugin p ON tr.plugin_id = p.id;

-- View for result aggregations by test run
CREATE OR REPLACE VIEW test_run_metrics AS
SELECT 
    trr.test_run_id,
    tm.code as metric_code,
    tm.description as metric_description,
    tm.unit as metric_unit,
    COUNT(*) as measurement_count,
    AVG(trr.value) as avg_value,
    MIN(trr.value) as min_value,
    MAX(trr.value) as max_value,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY trr.value) as median_value,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY trr.value) as p95_value,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY trr.value) as p99_value,
    STDDEV(trr.value) as std_dev
FROM test_run_result trr
JOIN test_metric tm ON trr.test_metric_id = tm.id
GROUP BY trr.test_run_id, tm.id, tm.code, tm.description, tm.unit;

-- 5. Add update triggers for timestamp maintenance

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_test_type_updated_at BEFORE UPDATE ON test_type
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_test_metric_updated_at BEFORE UPDATE ON test_metric
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 6. Create performance monitoring views

-- Active test runs with runtime information
CREATE OR REPLACE VIEW active_test_runs AS
SELECT 
    tr.*,
    tt.code as test_type_code,
    p.name as plugin_name,
    EXTRACT(EPOCH FROM (COALESCE(tr.end_time, now()) - tr.start_time)) as runtime_seconds
FROM test_run tr
JOIN test_type tt ON tr.test_type_id = tt.id
JOIN plugin p ON tr.plugin_id = p.id
WHERE tr.status IN ('running', 'pending');

-- System health metrics
CREATE OR REPLACE VIEW system_health AS
SELECT 
    (SELECT COUNT(*) FROM test_run WHERE status = 'running') as running_tests,
    (SELECT COUNT(*) FROM test_run WHERE status = 'pending') as pending_tests,
    (SELECT COUNT(*) FROM test_run WHERE created_at > now() - interval '1 hour') as tests_last_hour,
    (SELECT COUNT(*) FROM plugin WHERE is_active = true) as active_plugins,
    (SELECT COUNT(*) FROM test_type WHERE is_active = true) as active_test_types;
