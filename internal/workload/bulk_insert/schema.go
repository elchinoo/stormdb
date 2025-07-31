// internal/workload/bulk_insert/schema.go
package bulk_insert

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Schema for bulk insert performance testing with diverse data types
// Designed to stress different PostgreSQL subsystems and storage patterns
const createTableSQL = `
-- Table designed for bulk insert performance testing
-- Includes various data types to test different storage and indexing scenarios
CREATE TABLE IF NOT EXISTS bulk_insert_test (
    -- Primary key with sequence for natural ordering
    id BIGSERIAL PRIMARY KEY,
    
    -- Text data of varying lengths for storage pattern testing
    short_text VARCHAR(50),
    medium_text VARCHAR(500),
    long_text TEXT,
    
    -- Numeric data for arithmetic operations and indexing
    int_value INTEGER,
    bigint_value BIGINT,
    decimal_value DECIMAL(15,4),
    float_value DOUBLE PRECISION,
    
    -- Temporal data for time-series patterns
    created_timestamp TIMESTAMPTZ DEFAULT NOW(),
    event_date DATE,
    event_time TIME,
    
    -- Boolean for filtering patterns
    is_active BOOLEAN DEFAULT TRUE,
    
    -- JSON for semi-structured data testing
    metadata JSONB,
    
    -- Binary data for storage size testing
    data_blob BYTEA,
    
    -- UUID for uniqueness testing
    external_id UUID DEFAULT gen_random_uuid(),
    
    -- Enum type for categorical data
    status_enum bulk_status DEFAULT 'pending',
    
    -- Array type for complex data structures
    tags TEXT[],
    
    -- Network address type for specialized indexing
    client_ip INET,
    
    -- Geometric type for spatial operations
    location POINT
);

-- Custom enum type for status testing
CREATE TYPE IF NOT EXISTS bulk_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'cancelled');

-- Indexes for different access patterns
-- B-tree indexes for range queries
CREATE INDEX IF NOT EXISTS idx_bulk_created_timestamp ON bulk_insert_test(created_timestamp);
CREATE INDEX IF NOT EXISTS idx_bulk_int_value ON bulk_insert_test(int_value);
CREATE INDEX IF NOT EXISTS idx_bulk_status ON bulk_insert_test(status_enum);

-- Composite index for multi-column queries
CREATE INDEX IF NOT EXISTS idx_bulk_status_date ON bulk_insert_test(status_enum, event_date);

-- Partial index for filtering patterns
CREATE INDEX IF NOT EXISTS idx_bulk_active_items ON bulk_insert_test(id) WHERE is_active = true;

-- JSONB GIN index for JSON operations
CREATE INDEX IF NOT EXISTS idx_bulk_metadata_gin ON bulk_insert_test USING GIN(metadata);

-- Hash index for equality lookups
CREATE INDEX IF NOT EXISTS idx_bulk_external_id_hash ON bulk_insert_test USING HASH(external_id);
`

const dropTableSQL = `
DROP TABLE IF EXISTS bulk_insert_test CASCADE;
DROP TYPE IF EXISTS bulk_status CASCADE;
`

// setupSchema creates the table and indexes for bulk insert testing
func setupSchema(ctx context.Context, db *pgxpool.Pool) error {
	log.Printf("🔧 Setting up bulk insert test schema...")
	
	_, err := db.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create bulk insert schema: %w", err)
	}
	
	log.Printf("✅ Bulk insert test schema created successfully")
	return nil
}

// cleanupSchema drops the table and related objects
func cleanupSchema(ctx context.Context, db *pgxpool.Pool) error {
	log.Printf("🗑️  Dropping bulk insert test schema...")
	
	_, err := db.Exec(ctx, dropTableSQL)
	if err != nil {
		return fmt.Errorf("failed to drop bulk insert schema: %w", err)
	}
	
	log.Printf("✅ Bulk insert test schema dropped successfully")
	return nil
}

// getTableStats returns statistics about the bulk insert test table
func getTableStats(ctx context.Context, db *pgxpool.Pool) (int64, error) {
	var count int64
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM bulk_insert_test").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get table count: %w", err)
	}
	return count, nil
}

// truncateTable removes all data from the table without dropping schema
func truncateTable(ctx context.Context, db *pgxpool.Pool) error {
	log.Printf("🔄 Truncating bulk insert test table...")
	
	_, err := db.Exec(ctx, "TRUNCATE TABLE bulk_insert_test RESTART IDENTITY")
	if err != nil {
		return fmt.Errorf("failed to truncate bulk insert table: %w", err)
	}
	
	log.Printf("✅ Bulk insert test table truncated successfully")
	return nil
}
