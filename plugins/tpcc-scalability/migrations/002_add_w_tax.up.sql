-- 002_add_w_tax.up.sql
-- Ensure warehouse table has w_tax column with default
ALTER TABLE warehouse ADD COLUMN IF NOT EXISTS w_tax DECIMAL(4,4) NOT NULL DEFAULT 0.0000;
