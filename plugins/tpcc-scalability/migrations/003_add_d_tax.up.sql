-- 003_add_d_tax.up.sql
-- Ensure district table has d_tax column with default
ALTER TABLE district ADD COLUMN IF NOT EXISTS d_tax DECIMAL(4,4) NOT NULL DEFAULT 0.0000;
