-- Migration to align the test_run table with the v2 schema changes.
-- The plugin_ver column is now redundant, as the version is stored in the plugin table
-- and linked via plugin_id. This migration drops the dependent views, removes the
-- column, and then recreates the views with the updated schema.

-- Step 1: Drop the dependent views
DROP VIEW IF EXISTS active_test_runs;
DROP VIEW IF EXISTS test_run_summary;

-- Step 2: Drop the redundant column from the test_run table
ALTER TABLE test_run DROP COLUMN IF EXISTS plugin_ver;

-- Step 3: Recreate the test_run_summary view without the old column
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

-- Step 4: Recreate the active_test_runs view, explicitly listing columns
-- to avoid issues with wildcard selections in the future.
CREATE OR REPLACE VIEW active_test_runs AS
SELECT 
    tr.id,
    tr.test_type_id,
    tr.plugin_id,
    tr.host,
    tr.port,
    tr.db_name,
    tr.name,
    tr.description,
    tr.created_at,
    tr.start_time,
    tr.end_time,
    tr.status,
    tr.config,
    tt.code as test_type_code,
    p.name as plugin_name,
    EXTRACT(EPOCH FROM (COALESCE(tr.end_time, now()) - tr.start_time)) as runtime_seconds
FROM test_run tr
JOIN test_type tt ON tr.test_type_id = tt.id
JOIN plugin p ON tr.plugin_id = p.id
WHERE tr.status IN ('running', 'pending');
