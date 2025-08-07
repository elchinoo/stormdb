-- 007_add_missing_tpcc_tables.up.sql
-- Add missing core TPC-C tables required for transaction processing

-- Orders table (note: "order" is a reserved word in PostgreSQL, so we need to quote it)
CREATE TABLE IF NOT EXISTS "order" (
    o_w_id INT NOT NULL,
    o_d_id INT NOT NULL,
    o_id INT NOT NULL,
    o_c_id INT NOT NULL,
    o_entry_d TIMESTAMPTZ NOT NULL,
    o_carrier_id INT,
    o_ol_cnt INT NOT NULL,
    o_all_local INT NOT NULL,
    PRIMARY KEY (o_w_id, o_d_id, o_id),
    FOREIGN KEY (o_w_id, o_d_id) REFERENCES district(d_w_id, d_id),
    FOREIGN KEY (o_w_id, o_d_id, o_c_id) REFERENCES customer(c_w_id, c_d_id, c_id)
);

-- New Order table
CREATE TABLE IF NOT EXISTS new_order (
    no_w_id INT NOT NULL,
    no_d_id INT NOT NULL,
    no_o_id INT NOT NULL,
    PRIMARY KEY (no_w_id, no_d_id, no_o_id),
    FOREIGN KEY (no_w_id, no_d_id, no_o_id) REFERENCES "order"(o_w_id, o_d_id, o_id)
);

-- Order Line table
CREATE TABLE IF NOT EXISTS order_line (
    ol_w_id INT NOT NULL,
    ol_d_id INT NOT NULL,
    ol_o_id INT NOT NULL,
    ol_number INT NOT NULL,
    ol_i_id INT NOT NULL,
    ol_supply_w_id INT NOT NULL,
    ol_delivery_d TIMESTAMPTZ,
    ol_quantity INT NOT NULL,
    ol_amount DECIMAL(6,2) NOT NULL,
    ol_dist_info CHAR(24) NOT NULL,
    PRIMARY KEY (ol_w_id, ol_d_id, ol_o_id, ol_number),
    FOREIGN KEY (ol_w_id, ol_d_id, ol_o_id) REFERENCES "order"(o_w_id, o_d_id, o_id),
    FOREIGN KEY (ol_i_id) REFERENCES item(i_id),
    FOREIGN KEY (ol_supply_w_id) REFERENCES warehouse(w_id)
);

-- History table
CREATE TABLE IF NOT EXISTS history (
    h_c_id INT NOT NULL,
    h_c_d_id INT NOT NULL,
    h_c_w_id INT NOT NULL,
    h_d_id INT NOT NULL,
    h_w_id INT NOT NULL,
    h_date TIMESTAMPTZ NOT NULL,
    h_amount DECIMAL(6,2) NOT NULL,
    h_data VARCHAR(24) NOT NULL,
    FOREIGN KEY (h_c_w_id, h_c_d_id, h_c_id) REFERENCES customer(c_w_id, c_d_id, c_id),
    FOREIGN KEY (h_w_id) REFERENCES warehouse(w_id),
    FOREIGN KEY (h_w_id, h_d_id) REFERENCES district(d_w_id, d_id)
);

-- Create indices for better performance
CREATE INDEX IF NOT EXISTS idx_order_customer ON "order"(o_w_id, o_d_id, o_c_id);
CREATE INDEX IF NOT EXISTS idx_order_entry_d ON "order"(o_entry_d);
CREATE INDEX IF NOT EXISTS idx_new_order_w_d ON new_order(no_w_id, no_d_id);
CREATE INDEX IF NOT EXISTS idx_order_line_w_d_o ON order_line(ol_w_id, ol_d_id, ol_o_id);
CREATE INDEX IF NOT EXISTS idx_history_c_w_d ON history(h_c_w_id, h_c_d_id, h_c_id);
CREATE INDEX IF NOT EXISTS idx_history_date ON history(h_date);
