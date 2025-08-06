-- StormDB v2 Database Initialization Script
-- This script sets up the database for TPC-C testing and StormDB core

-- Enable pg_stat_statements extension for query monitoring
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Create additional databases for testing if needed
CREATE DATABASE stormdb_core;
CREATE DATABASE tpcc_benchmark;

-- Create dedicated users with appropriate permissions
CREATE USER stormdb_user WITH PASSWORD 'stormdb_password';
CREATE USER tpcc_user WITH PASSWORD 'tpcc_password';

-- Grant permissions to StormDB user
GRANT ALL PRIVILEGES ON DATABASE stormdb_core TO stormdb_user;
GRANT ALL PRIVILEGES ON DATABASE tpcc_test TO stormdb_user;
GRANT ALL PRIVILEGES ON DATABASE tpcc_benchmark TO stormdb_user;

-- Grant permissions to TPC-C user
GRANT ALL PRIVILEGES ON DATABASE tpcc_test TO tpcc_user;
GRANT ALL PRIVILEGES ON DATABASE tpcc_benchmark TO tpcc_user;

-- Configure database settings for performance testing
ALTER DATABASE tpcc_test SET work_mem = '4MB';
ALTER DATABASE tpcc_test SET maintenance_work_mem = '64MB';
ALTER DATABASE tpcc_test SET checkpoint_completion_target = 0.9;
ALTER DATABASE tpcc_test SET wal_buffers = '16MB';
ALTER DATABASE tpcc_test SET default_statistics_target = 100;

ALTER DATABASE tpcc_benchmark SET work_mem = '8MB';
ALTER DATABASE tpcc_benchmark SET maintenance_work_mem = '128MB';
ALTER DATABASE tpcc_benchmark SET checkpoint_completion_target = 0.9;
ALTER DATABASE tpcc_benchmark SET wal_buffers = '32MB';
ALTER DATABASE tpcc_benchmark SET default_statistics_target = 100;

-- Create monitoring views for performance analysis
\c tpcc_test

-- View for monitoring active connections
CREATE OR REPLACE VIEW v_active_connections AS
SELECT 
    pid,
    usename,
    application_name,
    client_addr,
    state,
    query_start,
    state_change,
    CASE 
        WHEN state = 'active' THEN 
            EXTRACT(EPOCH FROM (NOW() - query_start))
        ELSE NULL 
    END as query_duration_seconds,
    LEFT(query, 100) as query_preview
FROM pg_stat_activity
WHERE state != 'idle'
ORDER BY query_start;

-- View for monitoring database statistics
CREATE OR REPLACE VIEW v_database_stats AS
SELECT 
    datname,
    numbackends,
    xact_commit,
    xact_rollback,
    ROUND(100.0 * xact_rollback / NULLIF(xact_commit + xact_rollback, 0), 2) as rollback_ratio,
    blks_read,
    blks_hit,
    ROUND(100.0 * blks_hit / NULLIF(blks_read + blks_hit, 0), 2) as cache_hit_ratio,
    tup_returned,
    tup_fetched,
    tup_inserted,
    tup_updated,
    tup_deleted
FROM pg_stat_database
WHERE datname = current_database();

-- View for monitoring table statistics
CREATE OR REPLACE VIEW v_table_stats AS
SELECT 
    schemaname,
    tablename,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch,
    n_tup_ins,
    n_tup_upd,
    n_tup_del,
    n_live_tup,
    n_dead_tup,
    ROUND(100.0 * n_dead_tup / NULLIF(n_live_tup + n_dead_tup, 0), 2) as dead_tuple_ratio
FROM pg_stat_user_tables
ORDER BY n_live_tup DESC;

-- Grant access to monitoring views
GRANT SELECT ON v_active_connections TO stormdb_user, tpcc_user;
GRANT SELECT ON v_database_stats TO stormdb_user, tpcc_user;
GRANT SELECT ON v_table_stats TO stormdb_user, tpcc_user;

-- Switch to stormdb_core database and apply core schema
\c stormdb_core

-- The core schema will be applied by StormDB migrations
-- This just ensures the database exists and is ready

COMMENT ON DATABASE stormdb_core IS 'StormDB v2 Core Database - Stores test metadata, results, and plugin information';
COMMENT ON DATABASE tpcc_test IS 'TPC-C Test Database - Default database for TPC-C scalability tests';
COMMENT ON DATABASE tpcc_benchmark IS 'TPC-C Benchmark Database - Additional database for benchmark comparisons';
