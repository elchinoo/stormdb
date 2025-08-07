-- 009_fix_history_data_length.up.sql
-- Fix h_data column length to accommodate longer warehouse and district names

-- The current VARCHAR(24) is too short for the formatted string "Warehouse-N    District-N"
-- which can be up to 25+ characters. According to TPC-C spec, h_data should be at least 24 characters
-- but can be longer to accommodate implementation-specific data.
ALTER TABLE history ALTER COLUMN h_data TYPE VARCHAR(50);
