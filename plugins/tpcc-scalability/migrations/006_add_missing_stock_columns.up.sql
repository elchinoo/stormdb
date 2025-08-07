-- 006_add_missing_stock_columns.up.sql
-- Add missing TPC-C stock table columns for proper transaction support

ALTER TABLE stock 
    ADD COLUMN IF NOT EXISTS s_ytd DECIMAL(8,0) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS s_order_cnt INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS s_remote_cnt INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS s_reorder_qty INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS s_data TEXT NOT NULL DEFAULT 'data';
