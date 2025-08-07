-- 005_add_stock_order_point.up.sql
-- Add missing order point column with default values
ALTER TABLE stock
    ADD COLUMN IF NOT EXISTS s_order_point INT NOT NULL DEFAULT 0;
